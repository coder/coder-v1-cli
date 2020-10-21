package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rjeczalik/notify"
	"golang.org/x/sync/semaphore"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/activity"
	"cdr.dev/coder-cli/internal/clog"
	"cdr.dev/wsep"
)

// Sync runs a live sync daemon.
type Sync struct {
	// Init sets whether the sync will do the initial init and then return fast.
	Init bool
	// LocalDir is an absolute path.
	LocalDir string
	// RemoteDir is an absolute path.
	RemoteDir string
	// DisableMetrics disables activity metric pushing.
	DisableMetrics bool

	Env    coder.Environment
	Client *coder.Client
}

// See https://lxadm.com/Rsync_exit_codes#List_of_standard_rsync_exit_codes.
const (
	rsyncExitCodeIncompat   = 2
	rsyncExitCodeDataStream = 12
)

func (s Sync) syncPaths(delete bool, local, remote string) error {
	self := os.Args[0]

	args := []string{"-zz",
		"-a",
		"--delete",
		"-e", self + " sh", local, s.Env.Name + ":" + remote,
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
	cmd.Stderr = ioutil.Discard
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == rsyncExitCodeIncompat {
				return xerrors.Errorf("no compatible rsync on remote machine: rsync: %w", err)
			} else if exitError.ExitCode() == rsyncExitCodeDataStream {
				return xerrors.Errorf("protocol datastream error or no remote rsync found: %w", err)
			} else {
				return xerrors.Errorf("rsync: %w", err)
			}
		}
		return xerrors.Errorf("rsync: %w", err)
	}
	return nil
}

func (s Sync) remoteCmd(ctx context.Context, prog string, args ...string) error {
	conn, err := s.Client.DialWsep(ctx, &s.Env)
	if err != nil {
		return xerrors.Errorf("dial websocket: %w", err)
	}
	defer func() { _ = conn.Close(websocket.CloseNormalClosure, "") }() // Best effort.

	execer := wsep.RemoteExecer(conn)
	process, err := execer.Start(ctx, wsep.Command{
		Command: prog,
		Args:    args,
	})
	if err != nil {
		return xerrors.Errorf("exec remote process: %w", err)
	}
	// NOTE: If the copy routine fail, it will result in `process.Wait` to unblock and report an error.
	go func() { _, _ = io.Copy(os.Stdout, process.Stdout()) }() // Best effort.
	go func() { _, _ = io.Copy(os.Stderr, process.Stderr()) }() // Best effort.

	if err := process.Wait(); err != nil {
		if code, ok := err.(wsep.ExitError); ok {
			return xerrors.Errorf("%s exit status: %d", prog, code)
		}
		return xerrors.Errorf("execution failure: %w", err)
	}

	return nil
}

// initSync performs the initial synchronization of the directory.
func (s Sync) initSync() error {
	clog.LogInfo(fmt.Sprintf("doing initial sync (%s -> %s)", s.LocalDir, s.RemoteDir))

	start := time.Now()
	// Delete old files on initial sync (e.g git checkout).
	// Add the "/." to the local directory so rsync doesn't try to place the directory
	// into the remote dir.
	if err := s.syncPaths(true, s.LocalDir+"/.", s.RemoteDir); err != nil {
		return err
	}
	clog.LogSuccess(
		fmt.Sprintf("finished initial sync (%s)", time.Since(start).Truncate(time.Millisecond)),
	)
	return nil
}

func (s Sync) convertPath(local string) string {
	relLocalPath, err := filepath.Rel(s.LocalDir, local)
	if err != nil {
		panic(err)
	}
	return filepath.Join(s.RemoteDir, relLocalPath)
}

func (s Sync) handleCreate(localPath string) error {
	target := s.convertPath(localPath)

	if err := s.syncPaths(false, localPath, target); err != nil {
		// File was quickly deleted.
		if _, e1 := os.Stat(localPath); os.IsNotExist(e1) { // NOTE: Discard any other stat error and just expose the syncPath one.
			return nil
		}
		return err
	}
	return nil
}

func (s Sync) handleDelete(localPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	return s.remoteCmd(ctx, "rm", "-rf", s.convertPath(localPath))
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
		clog.LogInfo(fmt.Sprintf("unhandled event %+v %s", ev.Event(), ev.Path()))
	}

	log := fmt.Sprintf("%v %s (%s)",
		ev.Event(), filepath.Base(localPath), time.Since(ev.CreatedAt).Truncate(time.Millisecond*10),
	)
	if err != nil {
		clog.Log(clog.Error(fmt.Sprintf("%s: %s", log, err)))
	} else {
		clog.LogSuccess(log)
	}
}

// ErrRestartSync describes a known error case that can be solved by re-starting the command
var ErrRestartSync = errors.New("the sync exited because it was overloaded, restart it")

// workEventGroup converges a group of events to prevent duplicate work.
func (s Sync) workEventGroup(evs []timedEvent) {
	cache := eventCache{}
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
		// TODO: Document why this error is discarded. See https://github.com/cdr/coder-cli/issues/122 for reference.
		_ = sem.Acquire(context.Background(), 1)

		ev := ev // Copy the event in the scope to make sure the go routine use the proper value.
		go func() {
			defer sem.Release(1)
			defer wg.Done()
			s.work(ev)
		}()
	}

	wg.Wait()
}

const (
	// maxinflightInotify sets the maximum number of inotifies before the
	// sync just restarts. Syncing a large amount of small files (e.g .git
	// or node_modules) is impossible to do performantly with individual
	// rsyncs.
	maxInflightInotify = 8
	maxEventDelay      = 7 * time.Second
	// maxAcceptableDispatch is the maximum amount of time before an event
	// should begin its journey to the server. This sets a lower bound for
	// perceivable latency, but the higher it is, the better the
	// optimization.
	maxAcceptableDispatch = 50 * time.Millisecond
)

// Version returns remote protocol version as a string.
// Or, an error if one exists.
func (s Sync) Version() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := s.Client.DialWsep(ctx, &s.Env)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close(websocket.CloseNormalClosure, "") }() // Best effort.

	execer := wsep.RemoteExecer(conn)
	process, err := execer.Start(ctx, wsep.Command{
		Command: "rsync",
		Args:    []string{"--version"},
	})
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	_, _ = io.Copy(buf, process.Stdout()) // Ignore error, if any, it would be handled by the process.Wait return.

	if err := process.Wait(); err != nil {
		return "", err
	}

	firstLine, err := buf.ReadString('\n')
	if err != nil {
		return "", err
	}

	versionString := strings.Split(firstLine, "protocol version ")

	return versionString[1], nil
}

// Run starts the sync synchronously.
// Use this command to debug what wasn't sync'd correctly:
// rsync -e "coder sh" -nicr ~/Projects/cdr/coder-cli/. ammar:/home/coder/coder-cli/
func (s Sync) Run() error {
	events := make(chan notify.EventInfo, maxInflightInotify)
	// Set up a recursive watch.
	// We do this before the initial sync so we can capture any changes
	// that may have happened during sync.
	if err := notify.Watch(path.Join(s.LocalDir, "..."), events, notify.All); err != nil {
		return xerrors.Errorf("create watch: %w", err)
	}
	defer notify.Stop(events)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := s.remoteCmd(ctx, "mkdir", "-p", s.RemoteDir); err != nil {
		return xerrors.Errorf("create remote directory: %w", err)
	}

	ap := activity.NewPusher(s.Client, s.Env.ID, activityName)
	ap.Push(ctx)

	setConsoleTitle("⏳ syncing project")
	if err := s.initSync(); err != nil {
		return err
	}

	if s.Init {
		return nil
	}

	clog.LogInfo(fmt.Sprintf("watching %s for changes", s.LocalDir))

	var droppedEvents uint64
	// Timed events lets us track how long each individual file takes to
	// update.
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
					clog.LogInfo("dropped event, sync should restart soon")
				}
			}
		}
	}()

	var eventGroup []timedEvent

	dispatchEventGroup := time.NewTicker(maxAcceptableDispatch)
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
			ap.Push(context.TODO())
		}
	}
}

const activityName = "sync"
