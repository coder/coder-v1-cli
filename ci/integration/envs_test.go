package integration

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"testing"

	"cdr.dev/coder-cli/ci/tcli"
	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/slog/sloggers/slogtest/assert"
	"github.com/google/go-cmp/cmp"
)

func TestEnvsCLI(t *testing.T) {
	t.Parallel()

	run(t, "coder-cli-env-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		headlessLogin(ctx, t, c)

		// Minimum args not received.
		c.Run(ctx, "coder envs create").Assert(t,
			tcli.StderrMatches(regexp.QuoteMeta("accepts 1 arg(s), received 0")),
			tcli.Error(),
		)

		// Successfully output help.
		c.Run(ctx, "coder envs create --help").Assert(t,
			tcli.Success(),
			tcli.StdoutMatches(regexp.QuoteMeta("Create a new environment under the active user.")),
			tcli.StderrEmpty(),
		)

		// Image unset.
		c.Run(ctx, "coder envs create test-env").Assert(t,
			tcli.StderrMatches(regexp.QuoteMeta("fatal: required flag(s) \"image\" not set")),
			tcli.Error(),
		)

		// Image not imported.
		c.Run(ctx, "coder envs create test-env --image doesntmatter").Assert(t,
			tcli.StderrMatches(regexp.QuoteMeta("fatal: image not found - did you forget to import this image?")),
			tcli.Error(),
		)

		name := randString(10)
		cpu := 2.3
		c.Run(ctx, fmt.Sprintf("coder envs create %s --image ubuntu --cpu %f", name, cpu)).Assert(t,
			tcli.Success(),
		)

		t.Cleanup(func() {
			run(t, "coder-envs-edit-cleanup", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
				headlessLogin(ctx, t, c)
				c.Run(ctx, fmt.Sprintf("coder envs rm %s --force", name)).Assert(t)
			})
		})

		c.Run(ctx, "coder envs ls").Assert(t,
			tcli.Success(),
			tcli.StdoutMatches(regexp.QuoteMeta(name)),
		)

		var env coder.Environment
		c.Run(ctx, fmt.Sprintf(`coder envs ls -o json | jq '.[] | select(.name == "%s")'`, name)).Assert(t,
			tcli.Success(),
			tcli.StdoutJSONUnmarshal(&env),
		)
		assert.Equal(t, "environment cpu was correctly set", cpu, float64(env.CPUCores), floatComparer)

		c.Run(ctx, fmt.Sprintf("coder envs watch-build %s", name)).Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, fmt.Sprintf("coder envs rm %s --force", name)).Assert(t,
			tcli.Success(),
		)
	})

	run(t, "coder-cli-env-edit-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		headlessLogin(ctx, t, c)

		name := randString(10)
		c.Run(ctx, fmt.Sprintf("coder envs create %s --image ubuntu --follow", name)).Assert(t,
			tcli.Success(),
		)
		t.Cleanup(func() {
			run(t, "coder-envs-edit-cleanup", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
				headlessLogin(ctx, t, c)
				c.Run(ctx, fmt.Sprintf("coder envs rm %s --force", name)).Assert(t)
			})
		})

		cpu := 2.1
		c.Run(ctx, fmt.Sprintf(`coder envs edit %s --cpu %f --follow`, name, cpu)).Assert(t,
			tcli.Success(),
		)

		var env coder.Environment
		c.Run(ctx, fmt.Sprintf(`coder envs ls -o json | jq '.[] | select(.name == "%s")'`, name)).Assert(t,
			tcli.Success(),
			tcli.StdoutJSONUnmarshal(&env),
		)
		assert.Equal(t, "cpu cores were updated", cpu, float64(env.CPUCores), floatComparer)

		c.Run(ctx, fmt.Sprintf("coder envs rm %s --force", name)).Assert(t,
			tcli.Success(),
		)
	})
}

var floatComparer = cmp.Comparer(func(x, y float64) bool {
	delta := math.Abs(x - y)
	mean := math.Abs(x+y) / 2.0
	return delta/mean < 0.001
})
