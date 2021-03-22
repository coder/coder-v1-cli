package cmd

import (
	"testing"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/slog/sloggers/slogtest/assert"
)

func Test_images(t *testing.T) {
	res := execute(t, nil, "images", "--help")
	res.success(t)

	res = execute(t, nil, "images", "ls")
	res.success(t)

	var images []coder.Image
	res = execute(t, nil, "images", "ls", "--output=json")
	res.success(t)
	res.stdoutUnmarshals(t, &images)
	assert.True(t, "more than 0 images", len(images) > 0)

	res = execute(t, nil, "images", "ls", "--org=doesntexist")
	res.error(t)
	res.stderrContains(t, "org name \"doesntexist\" not found")
}
