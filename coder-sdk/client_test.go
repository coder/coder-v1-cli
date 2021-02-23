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
	require.NoError(t, err, "failed to parse test server URL")

	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL: u,
		Token:   "SwdcSoq5Jc-0C1r8wfwm7h6h9i0RDk7JT",
	})
	require.NoError(t, err, "failed to create coder.Client")

	err = client.PushActivity(context.Background(), source, envID)
	require.NoError(t, err)
}

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

func TestAuthentication(t *testing.T) {
	t.Parallel()

	const token = "g4mtIPUaKt-pPl9Q0xmgKs7acSypHt4Jf"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken := r.Header.Get("Session-Token")
		require.Equal(t, token, gotToken, "token does not match")

		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(func() {
		server.Close()
	})

	u, err := url.Parse(server.URL)
	require.NoError(t, err, "failed to parse test server URL")

	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL: u,
		Token:   token,
	})
	require.NoError(t, err, "failed to create coder.Client")

	_, err = client.APIVersion(context.Background())
	require.NoError(t, err, "failed to get API version information")
}

func TestPasswordAuthentication(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/basic/login", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, r.Method, http.MethodPost, "login is a POST")

		expected := map[string]interface{}{
			"email":    "user@coder.com",
			"password": "coder4all",
		}
		var request map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err, "error decoding JSON")
		require.EqualValues(t, expected, request, "unexpected request data")

		response := map[string]interface{}{
			"session_token": "g4mtIPUaKt-pPl9Q0xmgKs7acSypHt4Jf",
		}

		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(response)
		require.NoError(t, err, "error encoding JSON")
	})
	mux.HandleFunc("/api/v0/users/me", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method, "Users is a GET")

		require.Equal(t, "g4mtIPUaKt-pPl9Q0xmgKs7acSypHt4Jf", r.Header.Get("Session-Token"), "expected session token to match return of login")

		user := map[string]interface{}{
			"id":                 "default",
			"email":              "user@coder.com",
			"username":           "charlie",
			"name":               "Charlie Root",
			"roles":              []coder.Role{coder.SiteAdmin},
			"temporary_password": false,
			"login_type":         coder.LoginTypeBuiltIn,
			"key_regenerated_at": time.Now(),
			"created_at":         time.Now(),
			"updated_at":         time.Now(),
		}

		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(user)
		require.NoError(t, err, "error encoding JSON")
	})
	server := httptest.NewTLSServer(mux)
	t.Cleanup(func() {
		server.Close()
	})

	u, err := url.Parse(server.URL)
	require.NoError(t, err, "failed to parse test server URL")
	require.Equal(t, "https", u.Scheme, "expected HTTPS base URL")

	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL:    u,
		HTTPClient: server.Client(),
		Email:      "user@coder.com",
		Password:   "coder4all",
	})
	require.NoError(t, err, "failed to create Client")

	user, err := client.Me(context.Background())
	require.NoError(t, err, "failed to get information about current user")
	require.Equal(t, "user@coder.com", user.Email, "expected test user")
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
	require.NoError(t, err, "failed to parse test server URL")

	for _, prefix := range contextRoots {
		u.Path = prefix

		client, err := coder.NewClient(coder.ClientOptions{
			BaseURL: u,
			Token:   "FrOgA6xhpM-p5nTfsupmvzYJA6DJSOUoE",
		})
		require.NoError(t, err, "failed to create coder.Client")

		_, err = client.Users(context.Background())
		require.Error(t, err, "expected 503 error")
	}
}
