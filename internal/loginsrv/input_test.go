package loginsrv_test

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"cdr.dev/slog/sloggers/slogtest/assert"

	"cdr.dev/coder-cli/internal/loginsrv"
)

// 100ms is plenty of time as we are dealing with simple in-memory pipe.
const readTimeout = 100 * time.Millisecond

func TestReadLine(t *testing.T) {
	t.Parallel()

	const testToken = "hellosession"

	for _, scene := range []struct{ name, format string }{
		{"full_url", "http://localhost:4321?session_token=%s\n"},
		{"direct", "%s\n"},
		{"whitespaces", "\n\n   %s  \n\n"},
	} {
		scene := scene
		t.Run(scene.name, func(t *testing.T) {
			t.Parallel()

			tokenChan := make(chan string)
			defer close(tokenChan)

			ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
			defer cancel()

			r, w := io.Pipe()
			defer func() { _, _ = r.Close(), w.Close() }() // Best effort.

			errChan := make(chan error)
			go func() { defer close(errChan); errChan <- loginsrv.ReadLine(ctx, r, ioutil.Discard, tokenChan) }()

			doneChan := make(chan struct{})
			go func() {
				defer close(doneChan)
				_, _ = fmt.Fprintf(w, scene.format, testToken) // Best effort.
			}()

			select {
			case <-ctx.Done():
				t.Fatal("Timeout sending the input.")
			case err := <-errChan:
				t.Fatalf("ReadLine returned before we got the token (%v).", err)
			case <-doneChan:
			}

			select {
			case <-ctx.Done():
				t.Fatal("Timeout waiting for the input.")
			case err := <-errChan:
				t.Fatalf("ReadLine returned before we got the token (%v).", err)
			case actualToken := <-tokenChan:
				assert.Equal(t, "Unexpected token received from readline.", testToken, actualToken)
			}

			select {
			case <-ctx.Done():
				t.Fatal("Timeout waiting for readline to finish.")
			case err := <-errChan:
				assert.Success(t, "Error reading the line.", err)
			}
		})
	}
}

func TestReadLineMissingToken(t *testing.T) {
	t.Parallel()

	tokenChan := make(chan string)
	defer close(tokenChan)

	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	defer cancel()

	r, w := io.Pipe()
	defer func() { _, _ = r.Close(), w.Close() }() // Best effort.

	errChan := make(chan error)
	go func() { defer close(errChan); errChan <- loginsrv.ReadLine(ctx, r, ioutil.Discard, tokenChan) }()

	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)

		// Send multiple empty lines.
		for i := 0; i < 5; i++ {
			_, _ = fmt.Fprint(w, "\n") // Best effort.
		}
	}()

	// Make sure the write doesn't timeout.
	select {
	case <-ctx.Done():
		t.Fatal("Timeout sending the input.")
	case err := <-errChan:
		t.Fatalf("ReadLine returned before we got the token (%+v).", err)
	case token, ok := <-tokenChan:
		t.Fatalf("Token channel unexpectedly unblocked. Data: %q, state: %t.", token, ok)
	case <-doneChan:
	}

	// Manually close the input.
	_ = r.CloseWithError(io.EOF) // Best effort.

	// Make sure ReadLine properly ended.
	select {
	case <-ctx.Done():
		t.Fatal("Timeout waiting for readline to finish.")
	case err := <-errChan:
		assert.Success(t, "Error reading the line.", err)
	case token, ok := <-tokenChan:
		t.Fatalf("Token channel unexpectedly unblocked. Data: %q, state: %t.", token, ok)
	}
}
