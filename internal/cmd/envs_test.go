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

//nolint
func Test_envs_ls_by_provider(t *testing.T) {
	for _, test := range []struct {
		name    string
		command []string
		assert  func(r result)
	}{
		{
			name:    "simple list",
			command: []string{"envs", "ls", "--provider", "built-in"},
			assert:  func(r result) { r.success(t) },
		},
		{
			name:    "list as json",
			command: []string{"envs", "ls", "--provider", "built-in", "--output", "json"},
			assert: func(r result) {
				var envs []coder.Environment
				r.stdoutUnmarshals(t, &envs)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.assert(execute(t, nil, test.command...))
		})
	}
}
