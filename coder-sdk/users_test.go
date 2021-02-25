package coder_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cdr.dev/coder-cli/coder-sdk"
)

func TestUsers(t *testing.T) {
	t.Parallel()

	const username = "root"
	const name = "Charlie Root"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method, "Users is a GET")
		require.Equal(t, "/api/v0/users", r.URL.Path)

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
		require.NoError(t, err, "error encoding JSON")
	}))
	t.Cleanup(func() {
		server.Close()
	})

	u, err := url.Parse(server.URL)
	require.NoError(t, err, "failed to parse test server URL")

	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL: u,
		Token:   "JcmErkJjju-KSrztst0IJX7xGJhKQPtfv",
	})
	require.NoError(t, err, "failed to create coder.Client")

	users, err := client.Users(context.Background())
	require.NoError(t, err, "error getting Users")
	require.Len(t, users, 1, "users should return a single user")
	require.Equal(t, name, users[0].Name)
	require.Equal(t, username, users[0].Username)
}

func TestUserUpdatePassword(t *testing.T) {
	t.Parallel()

	const oldPassword = "vt9g9rxsptrq"
	const newPassword = "wmf39jw2f7pk"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPatch, r.Method, "Users is a PATCH")
		require.Equal(t, "/api/v0/users/me", r.URL.Path)

		expected := map[string]interface{}{
			"old_password": oldPassword,
			"password":     newPassword,
		}
		var request map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err, "error decoding JSON")
		require.EqualValues(t, expected, request, "unexpected request data")

		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		server.Close()
	})

	u, err := url.Parse(server.URL)
	require.NoError(t, err, "failed to parse test server URL")

	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL:    u,
		HTTPClient: server.Client(),
		Token:      "JcmErkJjju-KSrztst0IJX7xGJhKQPtfv",
	})
	require.NoError(t, err, "failed to create coder.Client")

	err = client.UpdateUser(context.Background(), "me", coder.UpdateUserReq{
		UserPasswordSettings: &coder.UserPasswordSettings{
			OldPassword: oldPassword,
			Password:    newPassword,
			Temporary:   false,
		},
	})
	require.NoError(t, err, "error when updating password")
}
