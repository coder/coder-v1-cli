package sync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rjeczalik/notify"
	"go.coder.com/flog"
	"golang.org/x/sync/semaphore"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/internal/entclient"
	"cdr.dev/coder-cli/wush"
)

// Sync runs a live sync daemon.
type Sync struct {
	// Init sets whether the sync will do the initial init and then return fast.
	Init bool
	// LocalDir is an absolute path.
	LocalDir string
	// RemoteDir is an absolute path.
	RemoteDir string
	entclient.Environment
	*entclient.Client
}

func (s Sync) syncPaths(delete bool, local, remote string) error {
	self := os.Args[0]

	args := []string{"-zz",
		"-a",
		"--delete",
		"-e", self + " sh", local, s.Environment.Name + ":" + remote,
	}
	if delete {
		args = append([]string{"--delete"}, args...)
	}
	if os.Getenv("DEBUG_RSYNC") != "" {
		args = append([]string{"--progress"}, args...)
	}

	// See https://unix.stackexchange.com/questions/188737/does-compression-option-z-with-rsync-speed-up-backup
	// on compression level.
	// (AB): compression sped up the initial sync of the enterprise repo by 30%, leading me to believe it's
	// good in general for codebases.
	cmd := exec.Command("rsync", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		return xerrors.Errorf("rsync: %w", err)
	}
	return nil
}

func (s Sync) remoteRm(remote string) error {
	conn, err := s.Client.DialWush(s.Environment, nil, "rm", "-rf", remote)
	if err != nil {
		return err
	}
	defer conn.Close(websocket.CloseNormalClosure, "")
	wc := wush.NewClient(context.Background(), conn)
	go io.Copy(os.Stdout, wc.Stderr)
	go io.Copy(os.Stderr, wc.Stdout)
	code, err := wc.Wait()
	if err != nil {
		return xerrors.Errorf("wush failure: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("rm exit status: %v", code)
	}
	return nil
}

// initSync performs the initial synchronization of the directory.
func (s Sync) initSync() error {
	flog.Info("doing initial sync (%v -> %v)", s.LocalDir, s.RemoteDir)

	start := time.Now()
	// Delete old files on initial sync (e.g git checkout).
	err := s.syncPaths(true, s.LocalDir+"/.", s.RemoteDir)
	if err == nil {
		flog.Info("finished initial sync (%v)", time.Since(start).Truncate(time.Millisecond))
	}
	return err
}

func (s Sync) convertPath(local string) string {
	relLocalPath, err := filepath.Rel(s.LocalDir, local)
	if err != nil {
		panic(err)
	}
	return filepath.Join(
		s.RemoteDir,
		relLocalPath,
	)
}

func (s Sync) handleCreate(localPath string) error {
	target := s.convertPath(localPath)
	err := s.syncPaths(false, localPath, target)
	if err != nil {
		_, statErr := os.Stat(localPath)
		// File was quickly deleted.
		if os.IsNotExist(statErr) {
			return nil
		}

		return err
	}
	return nil
}

func (s Sync) handleDelete(localPath string) error {
	return s.remoteRm(s.convertPath(localPath))
}

func (s Sync) handleRename(localPath string) error {
	// The rename operation is sent in two events, one
	// for the old (gone) file and one for the new file.
	// Catching both would require complex state.
	// Instead, we turn it into a Create or Delete based
	// on file existence.
	info, err := os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return s.handleDelete(localPath)
		}
		return err
	}
	if info.IsDir() {
		// Without this, the directory will be created as a subdirectory.
		localPath += "/."
	}
	return s.handleCreate(localPath)
}

func (s Sync) work(ev timedEvent) {
	var (
		localPath = ev.Path()
		err       error
	)
	switch ev.Event() {
	case notify.Write, notify.Create:
		err = s.handleCreate(localPath)
	case notify.Rename:
		err = s.handleRename(localPath)
	case notify.Remove:
		err = s.handleDelete(localPath)
	default:
		flog.Info("unhandled event %v %+v", ev.Event(), ev.Path())
	}

	log := fmt.Sprintf("%v %v (%v)",
		ev.Event(), filepath.Base(localPath), time.Since(ev.CreatedAt).Truncate(time.Millisecond*10),
	)
	if err != nil {
		flog.Error(log+": %v", err)
	} else {
		flog.Success(log)
	}
}

var ErrRestartSync = errors.New("the sync exited because it was overloaded, restart it")

// workEventGroup converges a group of events to prevent duplicate work.
func (s Sync) workEventGroup(evs []timedEvent) {
	cache := make(eventCache)
	for _, ev := range evs {
		cache.Add(ev)
	}

	// We want to process events concurrently but safely for speed.
	// Because the event cache prevents duplicate events for the same file, race conditions of that type
	// are impossible.
	// What is possible is a dependency on a previous Rename or Create. For example, if a directory is renamed
	// and then a file is moved to it. AFAIK this dependecy only exists with Directories.
	// So, we sequentially process the list of directory Renames and Creates, and then concurrently
	// perform all Writes.
	for _, ev := range cache.SequentialEvents() {
		s.work(ev)
	}

	sem := semaphore.NewWeighted(8)

	var wg sync.WaitGroup
	for _, ev := range cache.ConcurrentEvents() {
		setConsoleTitle(fmtUpdateTitle(ev.Path()))

		wg.Add(1)
		sem.Acquire(context.Background(), 1)
		ev := ev
		go func() {
			defer sem.Release(1)
			defer wg.Done()
			s.work(ev)
		}()
	}

	wg.Wait()
}

const (
	// maxinflightInotify sets the maximum number of inotifies before the sync just restarts.
	// Syncing a large amount of small files (e.g .git or node_modules) is impossible to do performantly
	// with individual rsyncs.
	maxInflightInotify = 8
	maxEventDelay      = time.Second * 7
	// maxAcceptableDispatch is the maximum amount of time before an event should begin its journey to the server.
	// This sets a lower bound for perceivable latency, but the higher it is, the better the optimization.
	maxAcceptableDispatch = time.Millisecond * 50
)

// Run starts the sync synchronously.
// Use this command to debug what wasn't sync'd correctly:
// rsync -e "coder sh" -nicr ~/Projects/cdr/coder-cli/. ammar:/home/coder/coder-cli/
func (s Sync) Run() error {
	events := make(chan notify.EventInfo, maxInflightInotify)
	// Set up a recursive watch.
	// We do this before the initial sync so we can capture any changes that may have happened during sync.
	err := notify.Watch(path.Join(s.LocalDir, "..."), events, notify.All)
	if err != nil {
		return xerrors.Errorf("create watch: %w", err)
	}
	defer notify.Stop(events)

	setConsoleTitle("⏳ syncing project")
	err = s.initSync()
	if err != nil {
		return err
	}

	if s.Init {
		return nil
	}


	flog.Info("watching %s for changes", s.LocalDir)

	var droppedEvents uint64
	// Timed events lets us track how long each individual file takes to update.
	timedEvents := make(chan timedEvent, cap(events))
	go func() {
		defer close(timedEvents)
		for event := range events {
			select {
			case timedEvents <- timedEvent{
				CreatedAt: time.Now(),
				EventInfo: event,
			}:
			default:
				if atomic.AddUint64(&droppedEvents, 1) == 1 {
					flog.Info("dropped event, sync should restart soon")
				}
			}
		}
	}()

	var (
		eventGroup         []timedEvent
		dispatchEventGroup = time.NewTicker(maxAcceptableDispatch)
	)
	defer dispatchEventGroup.Stop()
	for {
		const watchingFilesystemTitle = "🛰 watching filesystem"
		setConsoleTitle(watchingFilesystemTitle)

		select {
		case ev := <-timedEvents:
			if atomic.LoadUint64(&droppedEvents) > 0 {
				return ErrRestartSync
			}

			eventGroup = append(eventGroup, ev)
		case <-dispatchEventGroup.C:
			if len(eventGroup) == 0 {
				continue
			}
			// We're too backlogged and should restart the sync.
			if time.Since(eventGroup[0].CreatedAt) > maxEventDelay {
				return ErrRestartSync
			}
			s.workEventGroup(eventGroup)
			eventGroup = eventGroup[:0]
		}
	}
}
