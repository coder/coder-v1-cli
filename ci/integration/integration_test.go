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

func TestTCli(t *testing.T) {
	ctx := context.Background()

	container, err := tcli.NewContainerRunner(ctx, &tcli.ContainerConfig{
		Image: "ubuntu:latest",
		Name:  "test-container",
		BindMounts: map[string]string{
			binpath: "/bin/coder",
		},
	})
	assert.Success(t, "new run container", err)
	defer container.Close()

	container.Run(ctx, "echo testing").Assert(t,
		tcli.Success(),
		tcli.StderrEmpty(),
		tcli.StdoutMatches("esting"),
	)

	container.Run(ctx, "sleep 1.5 && echo 1>&2 stderr-message").Assert(t,
		tcli.Success(),
		tcli.StdoutEmpty(),
		tcli.StderrMatches("message"),
		tcli.DurationGreaterThan(time.Second),
	)

	cmd := exec.CommandContext(ctx, "cat")
	cmd.Stdin = strings.NewReader("testing")

	container.RunCmd(cmd).Assert(t,
		tcli.Success(),
		tcli.StderrEmpty(),
		tcli.StdoutMatches("testing"),
	)

	container.Run(ctx, "which coder").Assert(t,
		tcli.Success(),
		tcli.StdoutMatches("/bin/coder"),
		tcli.StderrEmpty(),
	)

	container.Run(ctx, "coder version").Assert(t,
		tcli.StderrEmpty(),
		tcli.Success(),
		tcli.StdoutMatches("linux"),
	)
}

func TestHostRunner(t *testing.T) {
	var (
		c   tcli.HostRunner
		ctx = context.Background()
	)

	c.Run(ctx, "echo testing").Assert(t,
		tcli.Success(),
		tcli.StderrEmpty(),
		tcli.StdoutMatches("testing"),
	)

	wd, err := os.Getwd()
	assert.Success(t, "get working dir", err)

	c.Run(ctx, "pwd").Assert(t,
		tcli.Success(),
		tcli.StderrEmpty(),
		tcli.StdoutMatches(wd),
	)
}

func TestCoderCLI(t *testing.T) {
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

	creds := login(ctx, t)
	c.Run(ctx, fmt.Sprintf("mkdir -p ~/.config/coder && echo -ne %s > ~/.config/coder/session", creds.token)).Assert(t,
		tcli.Success(),
	)
	c.Run(ctx, fmt.Sprintf("echo -ne %s > ~/.config/coder/url", creds.url)).Assert(t,
		tcli.Success(),
	)

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
