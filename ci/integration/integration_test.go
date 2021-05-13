package integration

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"cdr.dev/slog/sloggers/slogtest/assert"

	"cdr.dev/coder-cli/pkg/tcli"
)

func run(t *testing.T, container string, execute func(t *testing.T, ctx context.Context, runner *tcli.ContainerRunner)) {
	t.Run(container, func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()

		c, err := tcli.NewContainerRunner(ctx, &tcli.ContainerConfig{
			Image: "coder-cli-integration:latest",
			Name:  container,
			BindMounts: map[string]string{
				binpath: "/bin/coder",
			},
		})
		assert.Success(t, "new run container", err)
		defer c.Close()

		execute(t, ctx, c)
	})
}

func TestCoderCLI(t *testing.T) {
	t.Parallel()
	run(t, "test-coder-cli", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		c.Run(ctx, "which coder").Assert(t,
			tcli.Success(),
			tcli.StderrEmpty(),
		)

		c.Run(ctx, "coder --version").Assert(t,
			tcli.StderrEmpty(),
			tcli.Success(),
			tcli.StdoutMatches("linux"),
		)

		c.Run(ctx, "coder --help").Assert(t,
			tcli.Success(),
			tcli.StdoutMatches("Available Commands"),
		)

		headlessLogin(ctx, t, c)

		c.Run(ctx, "coder ws").Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, "coder ws ls").Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, "coder ws ls -o json").Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, "coder tokens").Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, "coder tokens ls").Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, "coder tokens ls -o json").Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, "coder urls").Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, "coder sync").Assert(t,
			tcli.Error(),
		)

		c.Run(ctx, "coder sh").Assert(t,
			tcli.Error(),
		)

		c.Run(ctx, "coder logout").Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, "coder ws ls").Assert(t,
			tcli.Error(),
		)

		c.Run(ctx, "coder tokens ls").Assert(t,
			tcli.Error(),
		)
	})
}

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
