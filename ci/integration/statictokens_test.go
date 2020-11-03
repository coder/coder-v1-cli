package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"cdr.dev/coder-cli/pkg/tcli"
)

func TestStaticAuth(t *testing.T) {
	t.Parallel()
	run(t, "static-auth-test", func(t *testing.T, ctx context.Context, c *tcli.ContainerRunner) {
		headlessLogin(ctx, t, c)

		c.Run(ctx, "coder tokens ls").Assert(t,
			tcli.Success(),
		)

		var result *tcli.CommandResult
		tokenName := randString(5)
		c.Run(ctx, "coder tokens create "+tokenName).Assert(t,
			tcli.Success(),
			tcli.GetResult(&result),
		)

		c.Run(ctx, "rm -rf ~/.config/coder")

		cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("export CODER_URL=%s && export CODER_TOKEN=$(cat) && coder envs ls", os.Getenv("CODER_URL")))
		cmd.Stdin = strings.NewReader(string(result.Stdout))
		c.RunCmd(cmd).Assert(t,
			tcli.Success(),
		)

		c.Run(ctx, "coder envs ls").Assert(t,
			tcli.Success(),
		)
	})
}
