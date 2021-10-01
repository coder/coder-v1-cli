package coder

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_TrustEnvironment(t *testing.T) {
	ctx := context.Background()

	const version = "test"
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("coder-version", version)
		_, _ = w.Write([]byte("{}"))
		w.WriteHeader(http.StatusNoContent)
	}))

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	c, err := NewClient(ClientOptions{
		BaseURL: u,
		Token:   "random",
	})
	require.NoError(t, err)

	_, err = c.APIVersion(ctx)
	require.Error(t, err)
	require.Regexp(t, "x509: certificate signed by unknown authority", err.Error())

	// TODO: @emyrk check proper handshake
	c.httpClient = insecureHTTPClient()
	challenge, err := c.TrustEnvironment(ctx, "random")
	require.NoError(t, err)
	require.Len(t, challenge.PemCertificates, 1)

	// Add the cert to the trusted pool, try the api call again
	pool := x509.NewCertPool()
	for i := range challenge.Certificates {
		pool.AddCert(challenge.Certificates[i])
	}
	conf := &tls.Config{RootCAs: pool}
	c.httpClient = &http.Client{
		Timeout: time.Second * 3,
		Transport: &http.Transport{
			TLSClientConfig: conf,
		},
	}

	v, err := c.APIVersion(ctx)
	require.NoError(t, err)
	require.Equal(t, version, v)
}

func insecureHTTPClient() *http.Client {
	conf := &tls.Config{InsecureSkipVerify: true}
	return &http.Client{
		Timeout: time.Second * 3,
		Transport: &http.Transport{
			TLSClientConfig: conf,
		},
	}
}
