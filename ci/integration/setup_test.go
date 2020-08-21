package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cdr.dev/coder-cli/ci/tcli"
	"golang.org/x/xerrors"
)

// binpath is populated during package initialization with a path to the coder binary
var binpath string

// initialize integration tests by building the coder-cli binary
func init() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	binpath = filepath.Join(cwd, "bin", "coder")
	err = build(binpath)
	if err != nil {
		panic(err)
	}
}

// build the coder-cli binary and move to the integration testing bin directory
func build(path string) error {
	tar := fmt.Sprintf("coder-cli-linux-amd64.tar.gz")
	dir := filepath.Dir(path)
	cmd := exec.Command(
		"sh", "-c",
		fmt.Sprintf(
			"cd ../../ && mkdir -p %s && ./ci/steps/build.sh && cp ./ci/bin/%s %s/ && tar -xzf %s -C %s",
			dir, tar, dir, filepath.Join(dir, tar), dir),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return xerrors.Errorf("build coder-cli (%v): %w", string(out), err)
	}
	return nil
}

// write session tokens to the given container runner
func headlessLogin(ctx context.Context, t *testing.T, runner *tcli.ContainerRunner) {
	creds := login(ctx, t)
	cmd := exec.CommandContext(ctx, "sh", "-c", "mkdir -p ~/.config/coder && cat > ~/.config/coder/session")

	// !IMPORTANT: be careful that this does not appear in logs
	cmd.Stdin = strings.NewReader(creds.token)
	runner.RunCmd(cmd).Assert(t,
		tcli.Success(),
	)
	runner.Run(ctx, fmt.Sprintf("echo -ne %s > ~/.config/coder/url", creds.url)).Assert(t,
		tcli.Success(),
	)
}
