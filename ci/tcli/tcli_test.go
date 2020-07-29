package tcli_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"cdr.dev/coder-cli/ci/tcli"
	"cdr.dev/slog/sloggers/slogtest/assert"
)

func TestTCli(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	container, err := tcli.NewContainerRunner(ctx, &tcli.ContainerConfig{
		Image: "ubuntu:latest",
		Name:  "test-container",
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
}
func TestHostRunner(t *testing.T) {
	t.Parallel()
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
