package cmd

import (
	"testing"
)

func TestEnvsCommand(t *testing.T) {
	res := execute(t, []string{"envs", "ls"}, nil)
	res.success(t)
}
