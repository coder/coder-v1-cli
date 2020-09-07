package loginsrv_test

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cdr.dev/coder-cli/internal/loginsrv"
	"github.com/stretchr/testify/require"
)

// 500ms should be plenty enough, even on slow machine to perform the request/response cycle.
const httpTimeout = 500 * time.Millisecond

func TestLocalLoginHTTPServer(t *testing.T) {
	t.Parallel()

	t.Run("happy_path", func(t *testing.T) {
		t.Parallel()

		tokenChan := make(chan string)
		defer close(tokenChan)

		ts := httptest.NewServer(&loginsrv.Server{TokenChan: tokenChan})
		defer ts.Close()

		ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
		defer cancel()

		const testToken = "hellosession"

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"?session_token="+testToken, nil) // Can't fail.
		require.NoError(t, err, "Error creating the http request.")

		errChan := make(chan error)
		go func() {
			defer close(errChan)
			resp, err := http.DefaultClient.Do(req)

			_, _ = io.Copy(ioutil.Discard, resp.Body) // Ignore the body, worry about the response code.
			_ = resp.Body.Close()                     // Best effort.

			require.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code.")

			errChan <- err
		}()

		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for the session token.")
		case err := <-errChan:
			t.Fatalf("The HTTP client returned before we got the token (%+v).", err)
		case actualToken := <-tokenChan:
			require.Equal(t, actualToken, actualToken, "Unexpected token received from the local server.")
		}

		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for the handler to finish.")
		case err := <-errChan:
			require.NoError(t, err, "Error calling test server.")
			if t.Failed() { // Case where the assert within the goroutine failed.
				return
			}
		}
	})

	t.Run("missing_token", func(t *testing.T) {
		t.Parallel()

		tokenChan := make(chan string)
		defer close(tokenChan)

		ts := httptest.NewServer(&loginsrv.Server{TokenChan: tokenChan})
		defer ts.Close()

		ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil) // Can't fail.

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Error calling test server.")

		_, _ = io.Copy(ioutil.Discard, resp.Body) // Ignore the body, worry about the response code.
		_ = resp.Body.Close()                     // Best effort.

		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Unexpected status code.")
		select {
		case err := <-ctx.Done():
			t.Fatalf("Unexpected context termination: %s.", err)
		case token, ok := <-tokenChan:
			t.Fatalf("Token channel unexpectedly unblocked. Data: %q, state: %t.", token, ok)
		default:
			// Expected case: valid and live context.
		}
	})
}
