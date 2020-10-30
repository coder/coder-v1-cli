package integration

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"testing"
	"time"

	"cdr.dev/coder-cli/ci/tcli"
	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"cdr.dev/slog/sloggers/slogtest/assert"
	"github.com/google/go-cmp/cmp"
)

func cleanupClient(t *testing.T, ctx context.Context) *coder.Client {
	creds := login(ctx, t)

	u, err := url.Parse(creds.url)
	assert.Success(t, "parse base url", err)

	return &coder.Client{BaseURL: u, Token: creds.token}
}

func cleanupEnv(t *testing.T, client *coder.Client, envID string) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		slogtest.Info(t, "cleanuping up environment", slog.F("env_id", envID))
		_ = client.DeleteEnvironment(ctx, envID)
	}
}

func TestEnvsCLI(t *testing.T) {
	t.Parallel()

	run(t, "coder-cli-env-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		headlessLogin(ctx, t, c)
		client := cleanupClient(t, ctx)

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

		c.Run(ctx, "coder envs ls").Assert(t,
			tcli.Success(),
			tcli.StdoutMatches(regexp.QuoteMeta(name)),
		)

		var env coder.Environment
		c.Run(ctx, fmt.Sprintf(`coder envs ls -o json | jq '.[] | select(.name == "%s")'`, name)).Assert(t,
			tcli.Success(),
			tcli.StdoutJSONUnmarshal(&env),
		)

		// attempt to cleanup the environment even if tests fail
		t.Cleanup(cleanupEnv(t, client, env.ID))

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
		client := cleanupClient(t, ctx)

		name := randString(10)
		c.Run(ctx, fmt.Sprintf("coder envs create %s --image ubuntu --follow", name)).Assert(t,
			tcli.Success(),
		)

		var env coder.Environment
		c.Run(ctx, fmt.Sprintf(`coder envs ls -o json | jq '.[] | select(.name == "%s")'`, name)).Assert(t,
			tcli.Success(),
			tcli.StdoutJSONUnmarshal(&env),
		)

		// attempt to cleanup the environment even if tests fail
		t.Cleanup(cleanupEnv(t, client, env.ID))

		cpu := 2.1
		c.Run(ctx, fmt.Sprintf(`coder envs edit %s --cpu %f --follow`, name, cpu)).Assert(t,
			tcli.Success(),
		)

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
