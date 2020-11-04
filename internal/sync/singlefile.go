package sync

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/wsep"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
)

// SingleFile copies the given file into the remote dir or remote path of the given coder.Environment.
func SingleFile(ctx context.Context, local, remoteDir string, env *coder.Environment, client *coder.Client) error {
	conn, err := client.DialWsep(ctx, env.ID)
	if err != nil {
		return xerrors.Errorf("dial remote execer: %w", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "normal closure") }()

	if strings.HasSuffix(remoteDir, string(filepath.Separator)) {
		remoteDir += filepath.Base(local)
	}

	execer := wsep.RemoteExecer(conn)
	cmd := fmt.Sprintf(`[ -d %s ] && cat > %s/%s || cat > %s`, remoteDir, remoteDir, filepath.Base(local), remoteDir)
	process, err := execer.Start(ctx, wsep.Command{
		Command: "sh",
		Args:    []string{"-c", cmd},
		Stdin:   true,
	})
	if err != nil {
		return xerrors.Errorf("start sync command: %w", err)
	}

	sourceFile, err := os.Open(local)
	if err != nil {
		return xerrors.Errorf("open source file: %w", err)
	}

	go func() { _, _ = io.Copy(ioutil.Discard, process.Stdout()) }()
	go func() { _, _ = io.Copy(ioutil.Discard, process.Stderr()) }()
	go func() {
		stdin := process.Stdin()
		defer stdin.Close()
		_, _ = io.Copy(stdin, sourceFile)
	}()

	if err := process.Wait(); err != nil {
		return xerrors.Errorf("copy process: %w", err)
	}
	return nil
}
