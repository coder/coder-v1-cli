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
	"time"

	"github.com/gorilla/websocket"
	"github.com/rjeczalik/notify"
	"go.coder.com/flog"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/internal/entclient"
	"cdr.dev/coder-cli/wush"
)

// Sync runs a live sync daemon.
type Sync struct {
	// Init sets whether the sync will do the initial init and then return fast.
	Init      bool
	LocalDir  string
	RemoteDir string
	entclient.Environment
	*entclient.Client
}

func (s Sync) syncPaths(delete bool, local, remote string) error {
	self := os.Args[0]

	args := []string{"-zz",
		"-a", "--progress",
		"--delete",
		"-e", self + " sh", local, s.Environment.Name + ":" + remote,
	}
	if delete {
		args = append([]string{"--delete"}, args...)
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
	err := s.syncPaths(true, s.LocalDir, s.RemoteDir)
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

func (s Sync) work(ev notify.EventInfo) {
	var (
		localPath  = ev.Path()
		remotePath = s.convertPath(localPath)
		err        error
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

	if err != nil {
		flog.Error("%v: %v -> %v: %v", ev.Event(), localPath, remotePath, err)
	} else {
		flog.Success("%v: %v -> %v", ev.Event(), localPath, remotePath)
	}
}

func setConsoleTitle(title string) {
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
	return
	}
	fmt.Printf("\033]0;%s\007", title)
}

// maxinflightInotify sets the maximum number of inotifies before the sync just restarts.
// Syncing a large amount of small files (e.g .git or node_modules) is impossible to do performantly
// with individual rsyncs.
const maxInflightInotify = 16

var ErrRestartSync = errors.New("the sync exited because it was overloaded, restart it")

func (s Sync) Run() error {
	setConsoleTitle("⏳ syncing project")
	err := s.initSync()
	if err != nil {
		return err
	}

	if s.Init {
		return nil
	}

	// This queue is twice as large as the max in flight so we can check when it's full reliably.
	events := make(chan notify.EventInfo, maxInflightInotify*2)
	// Set up a recursive watch.
	err = notify.Watch(path.Join(s.LocalDir, "..."), events, notify.All)
	if err != nil {
		return xerrors.Errorf("create watch: %w", err)
	}
	defer notify.Stop(events)


	const watchingFilesystemTitle = "🛰 watching filesystem"
	setConsoleTitle(watchingFilesystemTitle)

	flog.Info("watching %s for changes", s.LocalDir)
	for ev := range events {
		if len(events) > maxInflightInotify {
			return ErrRestartSync
		}

		setConsoleTitle("🚀 updating " + filepath.Base(ev.Path()))
		s.work(ev)
		setConsoleTitle(watchingFilesystemTitle)
	}

	return nil
}
