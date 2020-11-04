// Package loginsrv defines the login server in use by coder-cli
// for performing the browser authentication flow.
package loginsrv

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/xerrors"
)

// ReadLine waits for the manual login input to send the session token.
// NOTE: As we are dealing with a Read, cancelling the context will not unblock.
//       The caller is expected to close the reader.
func ReadLine(ctx context.Context, r io.Reader, w io.Writer, tokenChan chan<- string) error {
	// Wrap the reader with bufio to simplify the readline.
	buf := bufio.NewReader(r)

retry:
	_, _ = fmt.Fprintf(w, "or enter token manually:\n") // Best effort. Can only fail on custom writers.
	line, err := buf.ReadString('\n')
	if err != nil {
		// If we get an expected error, discard it and stop the routine.
		// NOTE: UnexpectedEOF is indeed an expected error as we can get it if we receive the token via the http server.
		if err == io.EOF || err == io.ErrClosedPipe || err == io.ErrUnexpectedEOF {
			return nil
		}
		// In the of error, we don't try again. Error out right away.
		return xerrors.Errorf("read input: %w", err)
	}

	// If we don't have any data, try again to read.
	line = strings.TrimSpace(line)
	if line == "" {
		goto retry
	}

	// Handle the case where we copy/paste the full URL instead of just the token.
	// Useful as most browser will auto-select the full URL.
	if u, err := url.Parse(line); err == nil {
		// Check the query string only in case of success, ignore the error otherwise
		// as we consider the input to be the token itself.
		if token := u.Query().Get("session_token"); token != "" {
			line = token
		}
		// If the session_token is missing, we also consider the input the be the token, don't error out.
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case tokenChan <- line:
	}

	return nil
}
