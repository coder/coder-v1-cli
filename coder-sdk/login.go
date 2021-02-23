package coder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/xerrors"
)

// LoginRequest is a request to authenticate using email
// and password credentials.
//
// This is provided for use in tests, and we recommend users authenticate
// using an API Token.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse contains successful response data for an authentication
// request, including an API Token to be used for subsequent requests.
//
// This is provided for use in tests, and we recommend users authenticate
// using an API Token.
type LoginResponse struct {
	SessionToken string `json:"session_token"`
}

// LoginWithPassword exchanges the email/password pair for
// a Session Token.
//
// If client is nil, the http.DefaultClient will be used.
func LoginWithPassword(ctx context.Context, client *http.Client, baseURL *url.URL, req *LoginRequest) (resp *LoginResponse, err error) {
	if client == nil {
		client = http.DefaultClient
	}

	url := *baseURL
	url.Path = fmt.Sprint(strings.TrimSuffix(url.Path, "/"), "/auth/basic/login")

	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(req)
	if err != nil {
		return nil, xerrors.Errorf("failed to marshal JSON: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url.String(), buf)
	if err != nil {
		return nil, xerrors.Errorf("failed to create request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, xerrors.Errorf("error processing login request: %w", err)
	}
	defer response.Body.Close()

	err = json.NewDecoder(response.Body).Decode(&resp)
	if err != nil {
		return nil, xerrors.Errorf("failed to decode response: %w", err)
	}

	return resp, nil
}
