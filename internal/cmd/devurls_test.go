package cmd

import (
	"testing"
)

func Test_devurls(t *testing.T) {
	skipIfNoAuth(t)
	res := execute(t, nil, "urls", "ls")
	res.error(t)
}
