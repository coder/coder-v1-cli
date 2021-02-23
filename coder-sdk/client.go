package coder

import (
	"errors"
	"net/http"
	"net/url"
)

// ensure that DefaultClient implements Client.
var _ = Client(&DefaultClient{})

// Me is the user ID of the authenticated user.
const Me = "me"

// ClientOptions contains options for the Coder SDK Client.
type ClientOptions struct {
	// BaseURL is the root URL of the Coder installation.
	BaseURL *url.URL

	// Client is the http.Client to use for requests (optional).
	// If omitted, the http.DefaultClient will be used.
	HTTPClient *http.Client

	// Token is the API Token used to authenticate
	Token string
}

// NewClient creates a new default Coder SDK client.
func NewClient(opts ClientOptions) (*DefaultClient, error) {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	if opts.BaseURL == nil {
		return nil, errors.New("the BaseURL parameter is required")
	}

	if opts.Token == "" {
		return nil, errors.New("an API token is required")
	}

	client := &DefaultClient{
		baseURL:    opts.BaseURL,
		httpClient: httpClient,
		token:      opts.Token,
	}

	return client, nil
}

// DefaultClient is the default implementation of the coder.Client
// interface.
//
// The empty value is meaningless and the fields are unexported;
// use NewClient to create an instance.
type DefaultClient struct {
	// baseURL is the URL (scheme, hostname/IP address, port,
	// path prefix of the Coder installation)
	baseURL *url.URL

	// httpClient is the http.Client used to issue requests.
	httpClient *http.Client

	// token is the API Token credential.
	token string
}
