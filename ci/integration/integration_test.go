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

func build(t *testing.T, path string) {
	cmd := exec.Command(
		"sh", "-c",
		fmt.Sprintf("cd ../../ && go build -o %s ./cmd/coder", path),
	)
	cmd.Env = append(os.Environ(), "GOOS=linux", "CGO_ENABLED=0")

	out, err := cmd.CombinedOutput()
	t.Logf("%s", string(out))
	assert.Success(t, "build go binary", err)
}

func TestTCli(t *testing.T) {
	ctx := context.Background()

	cwd, err := os.Getwd()
	assert.Success(t, "get working dir", err)

	binpath := filepath.Join(cwd, "bin", "coder")
	build(t, binpath)

	container, err := tcli.NewRunContainer(ctx, &tcli.ContainerConfig{
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
}