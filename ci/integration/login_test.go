package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

	"cdr.dev/slog/sloggers/slogtest/assert"
)

type credentials struct {
	url, token string
}

func login(ctx context.Context, t *testing.T) credentials {
	var (
		email    = requireEnv(t, "CODER_EMAIL")
		password = requireEnv(t, "CODER_PASSWORD")
		rawURL   = requireEnv(t, "CODER_URL")
	)
	sessionToken := getSessionToken(ctx, t, email, password, rawURL)

	return credentials{
		url:   rawURL,
		token: sessionToken,
	}
}

func requireEnv(t *testing.T, key string) string {
	value := os.Getenv(key)
	assert.True(t, fmt.Sprintf("%q is nonempty", key), value != "")
	return value
}

type loginBuiltInAuthReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginBuiltInAuthResp struct {
	SessionToken string `json:"session_token"`
}

func getSessionToken(ctx context.Context, t *testing.T, email, password, rawURL string) string {
	reqbody := loginBuiltInAuthReq{
		Email:    email,
		Password: password,
	}
	body, err := json.Marshal(reqbody)
	assert.Success(t, "marshal login req body", err)

	u, err := url.Parse(rawURL)
	assert.Success(t, "parse raw url", err)
	u.Path = "/auth/basic/login"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	assert.Success(t, "new request", err)

	resp, err := http.DefaultClient.Do(req)
	assert.Success(t, "do request", err)
	assert.Equal(t, "request status 201", http.StatusCreated, resp.StatusCode)

	var tokenResp loginBuiltInAuthResp
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	assert.Success(t, "decode response", err)

	defer resp.Body.Close()

	return tokenResp.SessionToken
}
