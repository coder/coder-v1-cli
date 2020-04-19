package sync

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/rjeczalik/notify"
	"go.coder.com/flog"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/internal/client"
	"cdr.dev/coder-cli/wush"
)

// Sync runs a live sync daemon.
type Sync struct {
	// Init sets whether the sync will do the initial init and then return fast.
	Init      bool
	LocalDir  string
	RemoteDir string
	*client.Client
	client.Environment

	barWriter io.Writer
}

func (s Sync) pushFile(ctx context.Context, path string) error {
	conn, err := s.Wush(s.Environment, "scp", "-qprt", s.RemoteDir)
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	wc := wush.Dial(ctx, conn)

	// This starts scp in local mode
	cmd := exec.Command("scp", "-qprf", path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()

		_, err = io.Copy(io.MultiWriter(stdin, &debugWriter{
			Prefix: "s->c",
			W:      os.Stderr,
		}), wc.Stdout)
		if err != nil {
			flog.Error("s->c copy fail: %v", err)
		}
	}()

	cmd.Stdout = io.MultiWriter(s.barWriter, wc.Stdin, &debugWriter{
		Prefix: "c->s",
		W:      os.Stderr,
	})
	go io.Copy(os.Stderr, wc.Stderr)
	err = cmd.Run()
	if err != nil {
		return xerrors.Errorf("scp: %w", err)
	}
	return nil
}

func (s Sync) pushFileLog(ctx context.Context, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	start := time.Now()
	fmt.Printf("transferring %v...\t", info.Name())
	err = s.pushFile(ctx, path)
	if err != nil {
		fmt.Printf("failed\n")
		return err
	}
	fmt.Printf("done (%0.3fs)\n",
		time.Since(start).Seconds(),
	)
	return nil
}

// initSync performs the initial synchronization of the directory.
func (s Sync) initSync(ctx context.Context) error {
	flog.Info("doing initial sync (%v -> %v)", s.LocalDir, s.RemoteDir)

	bar := pb.StartNew(0)
	bar.Start()
	bar.SetWidth(100)
	defer bar.Finish()

	s.barWriter = bar.NewProxyWriter(ioutil.Discard)

	start := time.Now()
	err := filepath.Walk(s.LocalDir, func(path string, info os.FileInfo, err error) error {
		if path == s.LocalDir {
			// scp can't resolve the self directory
			return nil
		}
		return s.pushFile(ctx, path)
	})
	if err == nil {
		bar.Finish()
		flog.Info("finished initial sync (%v)", time.Since(start).Truncate(time.Millisecond))
	}
	return err
}

func (s Sync) Run() error {
	ctx := context.Background()

	err := s.initSync(ctx)
	if err != nil {
		return err
	}

	if s.Init {
		return nil
	}

	events := make(chan notify.EventInfo, 8)
	// Set up a recursive watch.
	err = notify.Watch(path.Join(s.LocalDir, "..."), events, notify.All)
	if err != nil {
		return xerrors.Errorf("create watch: %w", err)
	}
	defer notify.Stop(events)

	flog.Info("watching %s for changes", s.LocalDir)
	for ev := range events {
		return s.pushFileLog(ctx, ev.Path())
	}
	return nil
}
