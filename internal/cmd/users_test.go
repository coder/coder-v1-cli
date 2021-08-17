package cmd

import (
	"testing"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"

	"cdr.dev/coder-cli/coder-sdk"
)

func Test_users(t *testing.T) {
	skipIfNoAuth(t)

	var users []coder.User
	res := execute(t, nil, "users", "ls", "--output=json")
	res.success(t)
	res.stdoutUnmarshals(t, &users)
	assertCICD(t, users)

	res = execute(t, nil, "users", "ls", "--output=human")
	res.success(t)
}

func assertCICD(t *testing.T, users []coder.User) {
	for _, u := range users {
		if u.Username == "cicd" {
			return
		}
	}
	slogtest.Fatal(t, "did not find cicd user", slog.F("users", users))
}
