package cmd

import (
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"cdr.dev/slog/sloggers/slogtest/assert"

	"cdr.dev/coder-cli/coder-sdk"
)

// TestCLIX509Certs tests if setting an env var allows trusting
// custom certs
func TestCLIX509Certs(t *testing.T) {
	ctx := context.Background()

	// Setup a TLS server
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srvUrl, err := url.Parse(srv.URL)
	assert.Success(t, "test srv url", err)

	// Create a client using default http.Client WITHOUT certs
	// loaded.
	cli, err := coder.NewClient(coder.ClientOptions{
		BaseURL: srvUrl,
		Token:   "Random",
	})
	assert.Success(t, "new client", err)

	// Expect an unknown cert error
	_, err = cli.Me(ctx)
	var certErr x509.UnknownAuthorityError
	assert.True(t, "unknown cert", errors.As(err, &certErr))
}
