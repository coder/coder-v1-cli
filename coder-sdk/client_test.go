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

	require.Equal(t, token, client.Token(), "expected Token to match")
	require.EqualValues(t, *u, client.BaseURL(), "expected BaseURL to match")

	_, err = client.APIVersion(context.Background())
	require.NoError(t, err, "failed to get API version information")
}

func TestPasswordAuthentication(t *testing.T) {
	t.Parallel()

	const email = "user@coder.com"
	const password = "coder4all"
	const token = "g4mtIPUaKt-pPl9Q0xmgKs7acSypHt4Jf"

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/basic/login", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, r.Method, http.MethodPost, "login is a POST")

		expected := map[string]interface{}{
			"email":    email,
			"password": password,
		}
		var request map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err, "error decoding JSON")
		require.EqualValues(t, expected, request, "unexpected request data")

		response := map[string]interface{}{
			"session_token": token,
		}

		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(response)
		require.NoError(t, err, "error encoding JSON")
	})
	mux.HandleFunc("/api/v0/users/me", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method, "Users is a GET")

		require.Equal(t, token, r.Header.Get("Session-Token"), "expected session token to match return of login")

		user := map[string]interface{}{
			"id":                 "default",
			"email":              email,
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
		Email:      email,
		Password:   password,
	})
	require.NoError(t, err, "failed to create Client")
	require.Equal(t, token, client.Token(), "expected token to match")

	user, err := client.Me(context.Background())
	require.NoError(t, err, "failed to get information about current user")
	require.Equal(t, email, user.Email, "expected test user")
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
