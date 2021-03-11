package cmd

import (
	"testing"
)

func Test_providers_ls(t *testing.T) {
	skipIfNoAuth(t)
	res := execute(t, nil, "providers", "ls")
	res.success(t)
}
