package integration

import (
	"context"
	"regexp"
	"testing"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/tcli"
)

func TestImagesCLI(t *testing.T) {
	t.Parallel()

	run(t, "coder-cli-images-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		headlessLogin(ctx, t, c)

		// Successfully output help.
		c.Run(ctx, "coder images --help").Assert(t,
			tcli.Success(),
			tcli.StdoutMatches(regexp.QuoteMeta("Manage existing images and/or import new ones.")),
			tcli.StderrEmpty(),
		)

		// OK - human output
		c.Run(ctx, "coder images ls").Assert(t,
			tcli.Success(),
		)

		imgs := []coder.Image{}
		// OK - json output
		c.Run(ctx, "coder images ls --output json").Assert(t,
			tcli.Success(),
			tcli.StdoutJSONUnmarshal(&imgs),
		)

		// Org not found
		c.Run(ctx, "coder images ls --org doesntexist").Assert(t,
			tcli.Error(),
			tcli.StderrMatches(regexp.QuoteMeta("org name \"doesntexist\" not found\n\n")),
		)
	})
}
