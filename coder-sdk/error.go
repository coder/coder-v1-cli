package coder

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"

	"golang.org/x/xerrors"
)

// ErrNotFound describes an error case in which the requested resource could not be found
var ErrNotFound = xerrors.Errorf("resource not found")

// apiError is the expected payload format for our errors.
type apiError struct {
	Err struct {
		Msg string `json:"msg"`
	} `json:"error"`
}

func bodyError(resp *http.Response) error {
	byt, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return xerrors.Errorf("dump response: %w", err)
	}

	var msg apiError
	// Try to decode the payload as an error, if it fails or if there is no error message,
	// return the response URL with the dump.
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil || msg.Err.Msg == "" {
		return xerrors.Errorf("%s\n%s", resp.Request.URL, byt)
	}

	// If the payload was a in the expected error format with a message, include it.
	return xerrors.Errorf("%s\n%s%s", resp.Request.URL, byt, msg.Err.Msg)
}
