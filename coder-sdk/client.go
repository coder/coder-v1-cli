package coder

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/xerrors"
)

// ensure that DefaultClient implements Client.
var _ = Client(&DefaultClient{})

// Me is the user ID of the authenticated user.
const Me = "me"

// ClientOptions contains options for the Coder SDK Client.
type ClientOptions struct {
	// BaseURL is the root URL of the Coder installation (required).
	BaseURL *url.URL

	// Client is the http.Client to use for requests (optional).
	//
	// If omitted, the http.DefaultClient will be used.
	HTTPClient *http.Client

	// Token is the API Token used to authenticate (optional).
	//
	// If Token is provided, the DefaultClient will use it to
	// authenticate. If it is not provided, it will require another
	// type of credential, such as a Email/Password pair.
	Token string

	// Email will be used to authenticate.
	//
	// DefaultClient requires exactly one type of credential
	// (Token, or Email/Password pair).
	//
	// NewClient will exchange the Email and Password for a Token
	// and will not retain them in-memory after NewClient returns.
	Email string

	// Password will be used to authenticate.
	//
	// DefaultClient requires exactly one type of credential
	// (Token, or Email/Password pair).
	//
	// NewClient will exchange the Email and Password for a Token
	// and will not retain them in-memory after NewClient returns.
	Password string
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

	token := opts.Token
	if token == "" {
		if opts.Email == "" || opts.Password == "" {
			return nil, errors.New("either an API Token or email/password pair are required")
		}

		// Exchange the username/password for a token.
		// We apply a default timeout of 5 seconds here.
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		resp, err := LoginWithPassword(ctx, httpClient, opts.BaseURL, &LoginRequest{
			Email:    opts.Email,
			Password: opts.Password,
		})
		if err != nil {
			return nil, xerrors.Errorf("failed to login with email/password: %w", err)
		}

		token = resp.SessionToken
		if token == "" {
			return nil, errors.New("server returned an empty session token")
		}
	}

	// TODO: add basic validation to make sure the token looks OK.

	client := &DefaultClient{
		baseURL:    opts.BaseURL,
		httpClient: httpClient,
		token:      token,
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
