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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method, "Users is a GET")
		require.Equal(t, "/api/v0/users", r.URL.Path)

		users := []map[string]interface{}{
			{
				"id":                 "default",
				"email":              "root@user.com",
				"username":           "root",
				"name":               "Charlie Root",
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
	require.Equal(t, "Charlie Root", users[0].Name)
	require.Equal(t, "root", users[0].Username)
}

func TestUserUpdatePassword(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPatch, r.Method, "Users is a PATCH")
		require.Equal(t, "/api/v0/users/me", r.URL.Path)

		expected := map[string]interface{}{
			"old_password":       "vt9g9rxsptrq",
			"password":           "wmf39jw2f7pk",
			"temporary_password": true,
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
			OldPassword: "vt9g9rxsptrq",
			Password:    "wmf39jw2f7pk",
			Temporary:   true,
		},
	})
	require.NoError(t, err, "error when updating password")
}
