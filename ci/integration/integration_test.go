package integration

import (
	"bytes"
	"context"
	"io/ioutil"
	"math/rand"
	"os/exec"
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
			// use this bind mount just to fix the user perms issue
			//
			// we'll overwrite this value with the proper binary value later to fix the case
			// where we're using docker in docker and bind mounts aren't correct
			BindMounts: map[string]string{
				binpath: "/bin/coder",
			},
		})
		assert.Success(t, "new run container", err)
		defer c.Close()

		// read the test binary
		contents, err := ioutil.ReadFile(binpath)
		assert.Success(t, "read coder cli binary", err)

		// inject the test binary into the container runner
		// this is preferable to the bind mount given it's docker in docker limitation
		cmd := exec.CommandContext(ctx, "sh", "-c", "sudo cat - > /bin/coder")
		cmd.Stdin = bytes.NewReader(contents)
		c.RunCmd(cmd).Assert(t,
			tcli.Success(),
			tcli.StderrEmpty(),
		)

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
