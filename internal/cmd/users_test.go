package cmd

import (
	"testing"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
)

func Test_users(t *testing.T) {
	skipIfNoAuth(t)

	var users []coder.User
	res := execute(t, nil, "users", "ls", "--output=json")
	res.success(t)
	res.stdoutUnmarshals(t, &users)
	assertAdmin(t, users)

	res = execute(t, nil, "users", "ls", "--output=human")
	res.success(t)
}

func assertAdmin(t *testing.T, users []coder.User) {
	for _, u := range users {
		if u.Username == "admin" {
			return
		}
	}
	slogtest.Fatal(t, "did not find admin user", slog.F("users", users))
}
