package integration

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"cdr.dev/coder-cli/pkg/tcli"
)

func TestSecrets(t *testing.T) {
	t.Parallel()
	run(t, "secrets-cli-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		headlessLogin(ctx, t, c)

		c.Run(ctx, "coder secrets ls").Assert(t,
			tcli.Success(),
		)

		name, value := randString(8), randString(8)

		c.Run(ctx, "coder secrets create").Assert(t,
			tcli.Error(),
		)

		// this tests the "Value:" prompt fallback
		c.Run(ctx, fmt.Sprintf("echo %s | coder secrets create %s --from-prompt", value, name)).Assert(t,
			tcli.Success(),
			tcli.StderrEmpty(),
		)

		c.Run(ctx, "coder secrets ls").Assert(t,
			tcli.Success(),
			tcli.StderrEmpty(),
			tcli.StdoutMatches("Value"),
			tcli.StdoutMatches(regexp.QuoteMeta(name)),
		)

		c.Run(ctx, "coder secrets view "+name).Assert(t,
			tcli.Success(),
			tcli.StderrEmpty(),
			tcli.StdoutMatches(regexp.QuoteMeta(value)),
		)

		c.Run(ctx, "coder secrets rm").Assert(t,
			tcli.Error(),
		)
		c.Run(ctx, "coder secrets rm "+name).Assert(t,
			tcli.Success(),
		)
		c.Run(ctx, "coder secrets view "+name).Assert(t,
			tcli.Error(),
			tcli.StdoutEmpty(),
		)

		name, value = randString(8), randString(8)

		c.Run(ctx, fmt.Sprintf("coder secrets create %s --from-literal %s", name, value)).Assert(t,
			tcli.Success(),
			tcli.StderrEmpty(),
		)

		c.Run(ctx, "coder secrets view "+name).Assert(t,
			tcli.Success(),
			tcli.StdoutMatches(regexp.QuoteMeta(value)),
		)

		name, value = randString(8), randString(8)
		c.Run(ctx, fmt.Sprintf("echo %s > ~/secret.json", value)).Assert(t,
			tcli.Success(),
		)
		c.Run(ctx, fmt.Sprintf("coder secrets create %s --from-file ~/secret.json", name)).Assert(t,
			tcli.Success(),
		)
		c.Run(ctx, "coder secrets view "+name).Assert(t,
			tcli.Success(),
			tcli.StdoutMatches(regexp.QuoteMeta(value)),
		)
	})
}
