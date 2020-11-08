package integration

import (
	"context"
	"testing"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/tcli"
	"cdr.dev/slog/sloggers/slogtest/assert"
)

func TestTags(t *testing.T) {
	t.Parallel()
	run(t, "tags-cli-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		headlessLogin(ctx, t, c)
		client := cleanupClient(ctx, t)

		ensureImageImported(ctx, t, client, "ubuntu")

		c.Run(ctx, "coder tags ls").Assert(t,
			tcli.Error(),
		)
		c.Run(ctx, "coder tags ls --image ubuntu --org default").Assert(t,
			tcli.Success(),
		)
		var tags []coder.ImageTag
		c.Run(ctx, "coder tags ls --image ubuntu --org default --output json").Assert(t,
			tcli.Success(),
			tcli.StdoutJSONUnmarshal(&tags),
		)
		assert.True(t, "> 0 tags", len(tags) > 0)

		// TODO(@cmoog) add create and rm integration tests
	})
}
