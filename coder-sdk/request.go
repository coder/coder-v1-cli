package coder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/xerrors"
)

// request is a helper to set the cookie, marshal the payload and execute the request.
func (c Client) request(ctx context.Context, method, path string, in interface{}) (*http.Response, error) {
	// Create a default http client with the auth in the cookie.
	client, err := c.newHTTPClient()
	if err != nil {
		return nil, xerrors.Errorf("new http client: %w", err)
	}

	// If we have incoming data, encode it as json.
	var payload io.Reader
	if in != nil {
		body, err := json.Marshal(in)
		if err != nil {
			return nil, xerrors.Errorf("marshal request: %w", err)
		}
		payload = bytes.NewReader(body)
	}

	// Create the http request.
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL.String()+path, payload)
	if err != nil {
		return nil, xerrors.Errorf("create request: %w", err)
	}

	// Execute the request.
	return client.Do(req)
}

// requestBody is a helper extending the Client.request helper, checking the response code
// and decoding the response payload.
func (c Client) requestBody(ctx context.Context, method, path string, in, out interface{}) error {
	resp, err := c.request(ctx, method, path, in)
	if err != nil {
		return xerrors.Errorf("Execute request: %q", err)
	}
	defer func() { _ = resp.Body.Close() }() // Best effort, likely connection dropped.

	// Responses in the 100 are handled by the http lib, in the 200 range, we have a success.
	// Consider anything at or above 300 to be an error.
	if resp.StatusCode > 299 {
		return fmt.Errorf("unexpected status code: %w", bodyError(resp))
	}

	// If we expect a payload, process it as json.
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return xerrors.Errorf("decode response body: %w", err)
		}
	}
	return nil
}