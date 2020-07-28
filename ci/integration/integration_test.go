package integration

import (
	"context"
	"testing"
	"time"

	"cdr.dev/coder-cli/ci/tcli"
	"cdr.dev/slog/sloggers/slogtest/assert"
)

func TestTCli(t *testing.T) {
	ctx := context.Background()

	container, err := tcli.NewRunContainer(ctx, "ubuntu:latest", "test-container")
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
}
