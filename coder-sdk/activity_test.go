package coder_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/slog/sloggers/slogtest/assert"
)

func TestPushActivity(t *testing.T) {
	t.Parallel()

	const source = "test"
	const envID = "602d377a-e6b8d763cae7561885c5f1b2"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PushActivity is a POST", http.MethodPost, r.Method)
		assert.Equal(t, "URL matches", "/api/private/metrics/usage/push", r.URL.Path)

		expected := map[string]interface{}{
			"source":         source,
			"environment_id": envID,
		}
		var request map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&request)
		assert.Success(t, "error decoding JSON", err)
		assert.Equal(t, "unexpected request data", expected, request)

		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		server.Close()
	})

	u, err := url.Parse(server.URL)
	assert.Success(t, "failed to parse test server URL", err)

	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL: u,
		Token:   "SwdcSoq5Jc-0C1r8wfwm7h6h9i0RDk7JT",
	})
	assert.Success(t, "failed to create coder.Client", err)

	err = client.PushActivity(context.Background(), source, envID)
	assert.Success(t, "expected successful response from PushActivity", err)
}
