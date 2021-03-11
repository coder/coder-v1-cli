package cmd

import (
	"testing"

	"cdr.dev/coder-cli/coder-sdk"
)

func Test_envs_ls(t *testing.T) {
	skipIfNoAuth(t)
	res := execute(t, nil, "envs", "ls")
	res.success(t)

	res = execute(t, nil, "envs", "ls", "--output=json")
	res.success(t)

	var envs []coder.Environment
	res.stdoutUnmarshals(t, &envs)
}
