package coder_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"cdr.dev/coder-cli/coder-sdk"
)

func TestPushActivity(t *testing.T) {
	t.Parallel()

	const source = "test"
	const envID = "602d377a-e6b8d763cae7561885c5f1b2"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method, "PushActivity is a POST")
		require.Equal(t, "/api/private/metrics/usage/push", r.URL.Path)

		expected := map[string]interface{}{
			"source":         source,
			"environment_id": envID,
		}
		var request map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err, "error decoding JSON")
		require.EqualValues(t, expected, request, "unexpected request data")

		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		server.Close()
	})

	u, err := url.Parse(server.URL)
	require.NoError(t, err, "failed to parse test server URL")

	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL: u,
		Token:   "SwdcSoq5Jc-0C1r8wfwm7h6h9i0RDk7JT",
	})
	require.NoError(t, err, "failed to create coder.Client")

	err = client.PushActivity(context.Background(), source, envID)
	require.NoError(t, err)
}
