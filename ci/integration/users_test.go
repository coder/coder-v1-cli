package integration

import (
	"context"
	"testing"

	"cdr.dev/coder-cli/ci/tcli"
	"cdr.dev/coder-cli/internal/entclient"
	"cdr.dev/slog/sloggers/slogtest/assert"
)

func TestUsers(t *testing.T) {
	t.Parallel()
	run(t, "users-cli-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		c.Run(ctx, "which coder").Assert(t,
			tcli.Success(),
			tcli.StdoutMatches("/usr/sbin/coder"),
			tcli.StderrEmpty(),
		)

		headlessLogin(ctx, t, c)

		var user entclient.User
		c.Run(ctx, `coder users ls --output json | jq -c '.[] | select( .username == "charlie")'`).Assert(t,
			tcli.Success(),
			tcli.StdoutJSONUnmarshal(&user),
		)
		assert.Equal(t, "user email is as expected", "charlie@coder.com", user.Email)
		assert.Equal(t, "username is as expected", "Charlie", user.Name)

		c.Run(ctx, "coder users ls --output human | grep charlie").Assert(t,
			tcli.Success(),
			tcli.StdoutMatches("charlie"),
		)

		c.Run(ctx, "coder logout").Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, "coder users ls").Assert(t,
			tcli.Error(),
		)
	})
}
