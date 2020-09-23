package integration

import (
	"context"
	"testing"

	"cdr.dev/coder-cli/ci/tcli"
	"cdr.dev/coder-cli/coder-sdk"
)

func TestSSH(t *testing.T) {
	t.Parallel()
	run(t, "ssh-coder-cli-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		headlessLogin(ctx, t, c)

		// TODO remove this once we can create an environment if there aren't any
		var envs []coder.Environment
		c.Run(ctx, "coder envs ls --output json").Assert(t,
			tcli.Success(),
			tcli.StdoutJSONUnmarshal(&envs),
		)

		assert := tcli.Success()

		// if we don't have any environments, "coder config-ssh" will fail
		if len(envs) == 0 {
			assert = tcli.Error()
		}
		c.Run(ctx, "coder config-ssh").Assert(t,
			assert,
		)
	})
}
