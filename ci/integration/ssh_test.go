package integration

import (
	"context"
	"testing"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/tcli"
)

func TestSSH(t *testing.T) {
	t.Parallel()
	run(t, "ssh-coder-cli-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		headlessLogin(ctx, t, c)

		// TODO remove this once we can create a workspace if there aren't any
		var workspaces []coder.Workspace
		c.Run(ctx, "coder ws ls --output json").Assert(t,
			tcli.Success(),
			tcli.StdoutJSONUnmarshal(&workspaces),
		)

		assert := tcli.Success()

		// if we don't have any workspaces, "coder config-ssh" will fail
		if len(workspaces) == 0 {
			assert = tcli.Error()
		}
		c.Run(ctx, "coder config-ssh").Assert(t,
			assert,
		)
	})
}
