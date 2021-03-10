package cmd

import (
	"testing"
)

func Test_providers_ls(t *testing.T) {
	res := execute(t, nil, "providers", "ls")
	res.success(t)
}
