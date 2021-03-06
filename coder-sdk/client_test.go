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

func TestAuthentication(t *testing.T) {
	t.Parallel()

	const token = "g4mtIPUaKt-pPl9Q0xmgKs7acSypHt4Jf"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken := r.Header.Get("Session-Token")
		assert.Equal(t, "token does not match", token, gotToken)

		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(func() {
		server.Close()
	})

	u, err := url.Parse(server.URL)
	assert.Success(t, "failed to parse test server URL", err)

	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL: u,
		Token:   token,
	})
	assert.Success(t, "failed to create coder.Client", err)

	assert.Equal(t, "expected Token to match", token, client.Token())
	assert.Equal(t, "expected BaseURL to match", *u, client.BaseURL())

	_, err = client.APIVersion(context.Background())
	assert.Success(t, "failed to get API version information", err)
}

func TestPasswordAuthentication(t *testing.T) {
	t.Parallel()

	const email = "user@coder.com"
	const password = "coder4all"
	const token = "g4mtIPUaKt-pPl9Q0xmgKs7acSypHt4Jf"

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/basic/login", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "login is a POST", http.MethodPost, r.Method)

		expected := map[string]interface{}{
			"email":    email,
			"password": password,
		}
		var request map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&request)
		assert.Success(t, "error decoding JSON", err)
		assert.Equal(t, "unexpected request data", expected, request)

		response := map[string]interface{}{
			"session_token": token,
		}

		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(response)
		assert.Success(t, "error encoding JSON", err)
	})
	mux.HandleFunc("/api/v0/users/me", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Users is a GET", http.MethodGet, r.Method)

		gotToken := r.Header.Get("Session-Token")
		assert.Equal(t, "expected session token to match return of login", token, gotToken)

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
		assert.Success(t, "error encoding JSON", err)
	})
	server := httptest.NewTLSServer(mux)
	t.Cleanup(func() {
		server.Close()
	})

	u, err := url.Parse(server.URL)
	assert.Success(t, "failed to parse test server URL", err)
	assert.Equal(t, "expected HTTPS base URL", "https", u.Scheme)

	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL:    u,
		HTTPClient: server.Client(),
		Email:      email,
		Password:   password,
	})
	assert.Success(t, "failed to create Client", err)
	assert.Equal(t, "expected token to match", token, client.Token())

	user, err := client.Me(context.Background())
	assert.Success(t, "failed to get information about current user", err)
	assert.Equal(t, "expected test user", email, user.Email)
}

func TestContextRoot(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Users is a GET", http.MethodGet, r.Method)
		assert.Equal(t, "expected context root", "/context-root/api/v0/users", r.URL.Path)

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
	assert.Success(t, "failed to parse test server URL", err)

	for _, prefix := range contextRoots {
		u.Path = prefix

		client, err := coder.NewClient(coder.ClientOptions{
			BaseURL: u,
			Token:   "FrOgA6xhpM-p5nTfsupmvzYJA6DJSOUoE",
		})
		assert.Success(t, "failed to create coder.Client", err)

		_, err = client.Users(context.Background())
		assert.Error(t, "expected 503 error", err)
	}
}
