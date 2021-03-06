package integration

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"cdr.dev/slog/sloggers/slogtest/assert"
	"github.com/google/go-cmp/cmp"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/tcli"
)

func cleanupClient(ctx context.Context, t *testing.T) coder.Client {
	creds := login(ctx, t)

	u, err := url.Parse(creds.url)
	assert.Success(t, "parse base url", err)

	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL: u,
		Token:   creds.token,
	})
	assert.Success(t, "failed to create coder.Client", err)
	return client
}

func cleanupEnv(t *testing.T, client coder.Client, envID string) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		slogtest.Info(t, "cleanuping up environment", slog.F("env_id", envID))
		_ = client.DeleteEnvironment(ctx, envID)
	}
}

// this is a stopgap until we have support for a `coder images` subcommand
// until then, we want can use the coder.Client to ensure our integration tests
// work on fresh deployments.
func ensureImageImported(ctx context.Context, t *testing.T, client coder.Client, img string) {
	orgs, err := client.Organizations(ctx)
	assert.Success(t, "get orgs", err)

	var org *coder.Organization
search:
	for _, o := range orgs {
		for _, m := range o.Members {
			if m.Email == os.Getenv("CODER_EMAIL") {
				o := o
				org = &o
				break search
			}
		}
	}
	if org == nil {
		slogtest.Fatal(t, "failed to find org of current user")
		return // help the linter out a bit
	}

	registries, err := client.Registries(ctx, org.ID)
	assert.Success(t, "get registries", err)

	var dockerhubID string
	for _, r := range registries {
		if r.Registry == "index.docker.io" {
			dockerhubID = r.ID
		}
	}
	assert.True(t, "docker hub registry found", dockerhubID != "")

	imgs, err := client.OrganizationImages(ctx, org.ID)
	assert.Success(t, "get org images", err)
	found := false
	for _, i := range imgs {
		if i.Repository == img {
			found = true
		}
	}
	if !found {
		// ignore this error for now as it causes a race with other parallel tests
		_, _ = client.ImportImage(ctx, coder.ImportImageReq{
			RegistryID:      &dockerhubID,
			OrgID:           org.ID,
			Repository:      img,
			Tag:             "latest",
			DefaultCPUCores: 2.5,
			DefaultDiskGB:   22,
			DefaultMemoryGB: 3,
		})
	}
}

func TestEnvsCLI(t *testing.T) {
	t.Parallel()

	run(t, "coder-cli-env-tests", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		headlessLogin(ctx, t, c)
		client := cleanupClient(ctx, t)

		// Minimum args not received.
		c.Run(ctx, "coder envs create").Assert(t,
			tcli.StderrMatches(regexp.QuoteMeta("accepts 1 arg(s), received 0")),
			tcli.Error(),
		)

		// Successfully output help.
		c.Run(ctx, "coder envs create --help").Assert(t,
			tcli.Success(),
			tcli.StdoutMatches(regexp.QuoteMeta("Create a new Coder environment.")),
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

		ensureImageImported(ctx, t, client, "ubuntu")

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
		client := cleanupClient(ctx, t)

		ensureImageImported(ctx, t, client, "ubuntu")

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
