package integration

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"cdr.dev/coder-cli/ci/tcli"
	"cdr.dev/slog/sloggers/slogtest/assert"
)

func run(t *testing.T, container string, execute func(t *testing.T, ctx context.Context, runner *tcli.ContainerRunner)) {
	t.Run(container, func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		defer cancel()

		c, err := tcli.NewContainerRunner(ctx, &tcli.ContainerConfig{
			Image: "codercom/enterprise-dev",
			Name:  container,
		})
		assert.Success(t, "new run container", err)
		defer c.Close()

		err = c.CopyFiles(ctx, map[string]string{
			binpath: "/bin/coder",
		})
		assert.Success(t, "copy binary into testing container", err)

		execute(t, ctx, c)
	})
}

func TestCoderCLI(t *testing.T) {
	t.Parallel()
	run(t, "test-coder-cli", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		c.Run(ctx, "which coder").Assert(t,
			tcli.Success(),
			tcli.StdoutMatches("/usr/sbin/coder"),
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

		c.Run(ctx, "coder envs").Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, "coder envs ls").Assert(t,
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

		c.Run(ctx, "coder envs ls").Assert(t,
			tcli.Error(),
		)
	})

}

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
