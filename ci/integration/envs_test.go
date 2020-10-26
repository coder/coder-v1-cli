package integration

import (
	"context"
	"regexp"
	"testing"

	"cdr.dev/coder-cli/ci/tcli"
)

// From Coder organization images
// const ubuntuImgID = "5f443b16-30652892427b955601330fa5"

func TestEnvsCLI(t *testing.T) {
	t.Parallel()

	run(t, "coder-cli-env-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		headlessLogin(ctx, t, c)

		// Ensure binary is present.
		c.Run(ctx, "which coder").Assert(t,
			tcli.Success(),
			tcli.StdoutMatches("/usr/sbin/coder"),
			tcli.StderrEmpty(),
		)

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

		// TODO(Faris) : uncomment this when we can safely purge the environments
		// the integrations tests would create in the sidecar
		// Successfully create environment.
		// c.Run(ctx, "coder envs create --image "+ubuntuImgID+" test-ubuntu").Assert(t,
		// 	tcli.Success(),
		// 	// why does flog.Success write to stderr?
		// 	tcli.StderrMatches(regexp.QuoteMeta("Successfully created environment \"test-ubuntu\"")),
		// )

		// TODO(Faris) : uncomment this when we can safely purge the environments
		// the integrations tests would create in the sidecar
		// Successfully provision environment with fractional resource amounts
		// c.Run(ctx, fmt.Sprintf(`coder envs create -i %s -c 1.2 -m 1.4 non-whole-resource-amounts`, ubuntuImgID)).Assert(t,
		// 	tcli.Success(),
		// 	tcli.StderrMatches(regexp.QuoteMeta("Successfully created environment \"non-whole-resource-amounts\"")),
		// )
	})
}
