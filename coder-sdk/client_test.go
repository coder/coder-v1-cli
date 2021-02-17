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
	"github.com/stretchr/testify/require"
)

func TestPushActivity(t *testing.T) {
	t.Parallel()

	const source = "test"
	const envID = "602d377a-e6b8d763cae7561885c5f1b2"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method, "PushActivity is a POST")
		require.Equal(t, "/api/private/metrics/usage/push", r.URL.Path)

		expected := map[string]interface{}{
			"source":         source,
			"environment_id": envID,
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
	require.NoError(t, err)

	client := coder.Client{
		BaseURL: u,
	}
	err = client.PushActivity(context.Background(), source, envID)
	require.NoError(t, err)
}

func TestUsers(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method, "Users is a GET")
		require.Equal(t, "/api/v0/users", r.URL.Path)
		r.Cookie("session_token")

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
	require.NoError(t, err)

	client := coder.Client{
		BaseURL: u,
	}
	users, err := client.Users(context.Background())
	require.Len(t, users, 1, "users should return a single user")
	require.Equal(t, "Charlie Root", users[0].Name)
	require.Equal(t, "root", users[0].Username)
}

func TestAuthentication(t *testing.T) {
	t.Parallel()

	const token = "g4mtIPUaKt-pPl9Q0xmgKs7acSypHt4Jf"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		require.NoError(t, err, "error extracting session token")
		require.Equal(t, token, cookie.Value, "token does not match")

		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(func() {
		server.Close()
	})

	u, err := url.Parse(server.URL)
	require.NoError(t, err)

	client := coder.Client{
		BaseURL: u,
		Token:   token,
	}
	_, _ = client.APIVersion(context.Background())
}

func TestContextRoot(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, r.Method, http.MethodGet, "Users is a GET")
		require.Equal(t, r.URL.Path, "/context-root/api/v0/users")

		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(func() {
		server.Close()
	})

	contextRoots := []string{
		"/context-root",
		"/context-root/",
	}

	u, err := url.Parse(server.URL)
	require.NoError(t, err)

	for _, prefix := range contextRoots {
		u.Path = prefix

		client := coder.Client{
			BaseURL: u,
		}
		_, _ = client.Users(context.Background())
	}
}
