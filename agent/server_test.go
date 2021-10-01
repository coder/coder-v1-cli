package agent_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"cdr.dev/coder-cli/agent"
	"cdr.dev/slog"
)

func Test_TrustCert(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	srv := testServer()
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	s, err := agent.NewServer(agent.ServerArgs{
		Log:      slog.Make(),
		CoderURL: u,
		Token:    "random",
	})
	require.NoError(t, err, "NewServer")

	s.TrustCertificate(ctx)
}

func testServer() *httptest.Server {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	return srv
}
