package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cdr.dev/coder-cli/ci/tcli"
	"cdr.dev/slog/sloggers/slogtest/assert"
)

func build(path string) error {
	cmd := exec.Command(
		"sh", "-c",
		fmt.Sprintf("cd ../../ && go build -o %s ./cmd/coder", path),
	)
	cmd.Env = append(os.Environ(), "GOOS=linux", "CGO_ENABLED=0")

	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

var binpath string

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

// write session tokens to the given container runner
func headlessLogin(ctx context.Context, t *testing.T, runner *tcli.ContainerRunner) {
	creds := login(ctx, t)
	cmd := exec.CommandContext(ctx, "mkdir -p ~/.config/coder && cat > ~/.config/coder/session")

	// !IMPORTANT: be careful that this does not appear in logs
	cmd.Stdin = strings.NewReader(creds.token)
	runner.RunCmd(cmd).Assert(t,
		tcli.Success(),
	)
	runner.Run(ctx, fmt.Sprintf("echo -ne %s > ~/.config/coder/url", creds.url)).Assert(t,
		tcli.Success(),
	)
}

func TestCoderCLI(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	c, err := tcli.NewContainerRunner(ctx, &tcli.ContainerConfig{
		Image: "codercom/enterprise-dev",
		Name:  "coder-cli-tests",
		BindMounts: map[string]string{
			binpath: "/bin/coder",
		},
	})
	assert.Success(t, "new run container", err)
	defer c.Close()

	c.Run(ctx, "which coder").Assert(t,
		tcli.Success(),
		tcli.StdoutMatches("/usr/sbin/coder"),
		tcli.StderrEmpty(),
	)

	c.Run(ctx, "coder version").Assert(t,
		tcli.StderrEmpty(),
		tcli.Success(),
		tcli.StdoutMatches("linux"),
	)

	c.Run(ctx, "coder help").Assert(t,
		tcli.Success(),
		tcli.StderrMatches("Commands:"),
		tcli.StderrMatches("Usage: coder"),
		tcli.StdoutEmpty(),
	)

	headlessLogin(ctx, t, c)

	c.Run(ctx, "coder envs").Assert(t,
		tcli.Success(),
	)

	c.Run(ctx, "coder urls").Assert(t,
		tcli.Error(),
	)

	c.Run(ctx, "coder sync").Assert(t,
		tcli.Error(),
	)

	c.Run(ctx, "coder sh").Assert(t,
		tcli.Error(),
	)

	c.Run(ctx, "coder logout").Assert(t,
		tcli.Success(),
	)

	c.Run(ctx, "coder envs").Assert(t,
		tcli.Error(),
	)
}
