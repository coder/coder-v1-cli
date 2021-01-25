package integration

import (
	"context"
	"testing"

	"cdr.dev/slog/sloggers/slogtest/assert"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/tcli"
)

func TestUsers(t *testing.T) {
	t.Parallel()
	run(t, "users-cli-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		c.Run(ctx, "which coder").Assert(t,
			tcli.Success(),
			tcli.StderrEmpty(),
		)

		headlessLogin(ctx, t, c)

		var user coder.User
		c.Run(ctx, `coder users ls --output json | jq -c '.[] | select( .username == "admin")'`).Assert(t,
			tcli.Success(),
			tcli.StdoutJSONUnmarshal(&user),
		)
		assert.Equal(t, "user email is as expected", "admin", user.Email)
		assert.Equal(t, "name is as expected", "admin", user.Name)

		c.Run(ctx, "coder users ls --output human | grep admin").Assert(t,
			tcli.Success(),
			tcli.StdoutMatches("admin"),
		)

		c.Run(ctx, "coder logout").Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, "coder users ls").Assert(t,
			tcli.Error(),
		)
	})
}
