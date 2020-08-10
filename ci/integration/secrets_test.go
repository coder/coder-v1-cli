package integration

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"cdr.dev/coder-cli/ci/tcli"
	"cdr.dev/slog/sloggers/slogtest/assert"
)

func TestSecrets(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	c, err := tcli.NewContainerRunner(ctx, &tcli.ContainerConfig{
		Image: "codercom/enterprise-dev",
		Name:  "secrets-cli-tests",
		BindMounts: map[string]string{
			binpath: "/bin/coder",
		},
	})
	assert.Success(t, "new run container", err)
	defer c.Close()

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
	//
	c.Run(ctx, "coder secrets view "+name).Assert(t,
		tcli.Success(),
		tcli.StdoutMatches(regexp.QuoteMeta(value)),
	)
}
