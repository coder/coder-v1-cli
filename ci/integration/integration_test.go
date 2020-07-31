package integration

import (
	"context"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"cdr.dev/coder-cli/ci/tcli"
	"cdr.dev/coder-cli/internal/entclient"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest/assert"
)

func TestCoderCLI(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	c, err := tcli.NewContainerRunner(ctx, &tcli.ContainerConfig{
		Image: "codercom/enterprise-dev",
		Name:  "coder-cli-tests",
		BindMounts: map[string]string{
			binpath: "/bin/coder",
		},
	})
	assert.Success(t, "new run container", err)
	defer c.Close()

	c.Run(ctx, "which coder").Assert(t,
		tcli.Success(),
		tcli.StdoutMatches("/usr/sbin/coder"),
		tcli.StderrEmpty(),
	)

	c.Run(ctx, "coder version").Assert(t,
		tcli.StderrEmpty(),
		tcli.Success(),
		tcli.StdoutMatches("linux"),
	)

	c.Run(ctx, "coder help").Assert(t,
		tcli.Success(),
		tcli.StderrMatches("Commands:"),
		tcli.StderrMatches("Usage: coder"),
		tcli.StdoutEmpty(),
	)

	headlessLogin(ctx, t, c)

	c.Run(ctx, "coder envs").Assert(t,
		tcli.Success(),
	)

	c.Run(ctx, "coder urls").Assert(t,
		tcli.Error(),
	)

	c.Run(ctx, "coder sync").Assert(t,
		tcli.Error(),
	)

	c.Run(ctx, "coder sh").Assert(t,
		tcli.Error(),
	)

	var user entclient.User
	c.Run(ctx, `coder users ls -o json | jq -c '.[] | select( .username == "charlie")'`).Assert(t,
		tcli.Success(),
		stdoutUnmarshalsJSON(&user),
	)
	assert.Equal(t, "user email is as expected", "charlie@coder.com", user.Email)
	assert.Equal(t, "username is as expected", "Charlie", user.Name)

	c.Run(ctx, "coder users ls -o human | grep charlie").Assert(t,
		tcli.Success(),
		tcli.StdoutMatches("charlie"),
	)

	c.Run(ctx, "coder logout").Assert(t,
		tcli.Success(),
	)

	c.Run(ctx, "coder envs").Assert(t,
		tcli.Error(),
	)
}

func stdoutUnmarshalsJSON(target interface{}) tcli.Assertion {
	return func(t *testing.T, r *tcli.CommandResult) {
		slog.Helper()
		err := json.Unmarshal(r.Stdout, target)
		assert.Success(t, "json unmarshals", err)
	}
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
