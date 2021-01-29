package coder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/xerrors"
)

type requestOptions struct {
	BaseURLOverride *url.URL
	Query           url.Values
	Headers         http.Header
	Reader          io.Reader
}

type requestOption func(*requestOptions)

// withQueryParams sets the provided query parameters on the request.
func withQueryParams(q url.Values) func(o *requestOptions) {
	return func(o *requestOptions) {
		o.Query = q
	}
}

func withHeaders(h http.Header) func(o *requestOptions) {
	return func(o *requestOptions) {
		o.Headers = h
	}
}

func withBaseURL(base *url.URL) func(o *requestOptions) {
	return func(o *requestOptions) {
		o.BaseURLOverride = base
	}
}

func withBody(w io.Reader) func(o *requestOptions) {
	return func(o *requestOptions) {
		o.Reader = w
	}
}

// request is a helper to set the cookie, marshal the payload and execute the request.
func (c Client) request(ctx context.Context, method, path string, in interface{}, options ...requestOption) (*http.Response, error) {
	// Create a default http client with the auth in the cookie.
	client, err := c.newHTTPClient()
	if err != nil {
		return nil, xerrors.Errorf("new http client: %w", err)
	}
	url := *c.BaseURL

	var config requestOptions
	for _, o := range options {
		o(&config)
	}
	if config.BaseURLOverride != nil {
		url = *config.BaseURLOverride
	}
	if config.Query != nil {
		url.RawQuery = config.Query.Encode()
	}
	url.Path = path

	// If we have incoming data, encode it as json.
	var payload io.Reader
	if in != nil {
		body, err := json.Marshal(in)
		if err != nil {
			return nil, xerrors.Errorf("marshal request: %w", err)
		}
		payload = bytes.NewReader(body)
	}

	if config.Reader != nil {
		payload = config.Reader
	}

	// Create the http request.
	req, err := http.NewRequestWithContext(ctx, method, url.String(), payload)
	if err != nil {
		return nil, xerrors.Errorf("create request: %w", err)
	}

	if config.Headers != nil {
		req.Header = config.Headers
	}

	// Execute the request.
	return client.Do(req)
}

// requestBody is a helper extending the Client.request helper, checking the response code
// and decoding the response payload.
func (c Client) requestBody(ctx context.Context, method, path string, in, out interface{}, opts ...requestOption) error {
	resp, err := c.request(ctx, method, path, in, opts...)
	if err != nil {
		return xerrors.Errorf("Execute request: %q", err)
	}
	defer func() { _ = resp.Body.Close() }() // Best effort, likely connection dropped.

	// Responses in the 100 are handled by the http lib, in the 200 range, we have a success.
	// Consider anything at or above 300 to be an error.
	if resp.StatusCode > 299 {
		return fmt.Errorf("unexpected status code %d: %w", resp.StatusCode, bodyError(resp))
	}

	// If we expect a payload, process it as json.
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return xerrors.Errorf("decode response body: %w", err)
		}
	}
	return nil
}
