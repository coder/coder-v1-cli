package loginsrv_test

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cdr.dev/slog/sloggers/slogtest/assert"

	"cdr.dev/coder-cli/internal/loginsrv"
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
		assert.Success(t, "Error creating the http request", err)

		errChan := make(chan error)
		go func() {
			defer close(errChan)
			resp, err := http.DefaultClient.Do(req)

			_, _ = io.Copy(ioutil.Discard, resp.Body) // Ignore the body, worry about the response code.
			_ = resp.Body.Close()                     // Best effort.

			assert.Equal(t, "Unexpected status code", http.StatusOK, resp.StatusCode)

			errChan <- err
		}()

		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for the session token.")
		case err := <-errChan:
			t.Fatalf("The HTTP client returned before we got the token (%+v).", err)
		case actualToken := <-tokenChan:
			assert.Equal(t, "Unexpected token received from the local server.", testToken, actualToken)
		}

		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for the handler to finish.")
		case err := <-errChan:
			assert.Success(t, "Error calling test server", err)
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

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil) // Can't fail.
		assert.Success(t, "Error creating the http request", err)

		resp, err := http.DefaultClient.Do(req)
		assert.Success(t, "Error calling test server", err)

		_, _ = io.Copy(ioutil.Discard, resp.Body) // Ignore the body, worry about the response code.
		_ = resp.Body.Close()                     // Best effort.

		assert.Equal(t, "Unexpected status code", http.StatusBadRequest, resp.StatusCode)
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
