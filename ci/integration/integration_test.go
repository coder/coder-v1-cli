package integration

import (
	"context"
	"testing"
	"time"

	"cdr.dev/coder-cli/ci/tcli"
)

func TestTCli(t *testing.T) {
	ctx := context.Background()

	container := tcli.NewRunContainer(ctx, "", "test-container")

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
