package cmd

import (
	"context"
	"testing"

	"cdr.dev/slog/sloggers/slogtest/assert"

	"cdr.dev/coder-cli/coder-sdk"
)

func Test_tags(t *testing.T) {
	t.Skip("TODO: wait for dedicated test API server / DB so we can create an org")
	ctx := context.Background()

	skipIfNoAuth(t)

	res := execute(t, nil, "tags", "ls")
	res.error(t)

	ensureImageImported(ctx, t, testCoderClient, "ubuntu", "latest")

	res = execute(t, nil, "tags", "ls", "--image=ubuntu", "--org=default")
	res.success(t)

	var tags []coder.ImageTag
	res = execute(t, nil, "tags", "ls", "--image=ubuntu", "--org=default", "--output=json")
	res.success(t)
	res.stdoutUnmarshals(t, &tags)
	assert.True(t, "> 0 tags", len(tags) > 0)
}
