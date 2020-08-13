package integration

import (
	"context"
	"testing"

	"cdr.dev/coder-cli/ci/tcli"
)

func TestSSH(t *testing.T) {
	t.Parallel()
	run(t, "ssh-coder-cli-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		headlessLogin(ctx, t, c)
		c.Run(ctx, "coder config-ssh").Assert(t,
			tcli.Success(),
		)
	})
}
