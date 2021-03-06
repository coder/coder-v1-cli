package coder_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/slog/sloggers/slogtest/assert"
)

func TestUsers(t *testing.T) {
	t.Parallel()

	const username = "root"
	const name = "Charlie Root"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Users is a GET", http.MethodGet, r.Method)
		assert.Equal(t, "Path matches", "/api/v0/users", r.URL.Path)

		users := []map[string]interface{}{
			{
				"id":                 "default",
				"email":              "root@user.com",
				"username":           username,
				"name":               name,
				"roles":              []coder.Role{coder.SiteAdmin},
				"temporary_password": false,
				"login_type":         coder.LoginTypeBuiltIn,
				"key_regenerated_at": time.Now(),
				"created_at":         time.Now(),
				"updated_at":         time.Now(),
			},
		}

		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(users)
		assert.Success(t, "error encoding JSON", err)
	}))
	t.Cleanup(func() {
		server.Close()
	})

	u, err := url.Parse(server.URL)
	assert.Success(t, "failed to parse test server URL", err)

	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL: u,
		Token:   "JcmErkJjju-KSrztst0IJX7xGJhKQPtfv",
	})
	assert.Success(t, "failed to create coder.Client", err)

	users, err := client.Users(context.Background())
	assert.Success(t, "error getting Users", err)
	assert.True(t, "users should return a single user", len(users) == 1)
	assert.Equal(t, "expected user's name to match", name, users[0].Name)
	assert.Equal(t, "expected user's username to match", username, users[0].Username)
}

func TestUserUpdatePassword(t *testing.T) {
	t.Parallel()

	const oldPassword = "vt9g9rxsptrq"
	const newPassword = "wmf39jw2f7pk"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Users is a PATCH", http.MethodPatch, r.Method)
		assert.Equal(t, "Path matches", "/api/v0/users/me", r.URL.Path)

		expected := map[string]interface{}{
			"old_password": oldPassword,
			"password":     newPassword,
		}
		var request map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&request)
		assert.Success(t, "error decoding JSON", err)
		assert.Equal(t, "unexpected request data", expected, request)

		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		server.Close()
	})

	u, err := url.Parse(server.URL)
	assert.Success(t, "failed to parse test server URL", err)

	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL:    u,
		HTTPClient: server.Client(),
		Token:      "JcmErkJjju-KSrztst0IJX7xGJhKQPtfv",
	})
	assert.Success(t, "failed to create coder.Client", err)

	err = client.UpdateUser(context.Background(), "me", coder.UpdateUserReq{
		UserPasswordSettings: &coder.UserPasswordSettings{
			OldPassword: oldPassword,
			Password:    newPassword,
			Temporary:   false,
		},
	})
	assert.Success(t, "error when updating password", err)
}
