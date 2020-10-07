package coder

import (
	"encoding/json"
	"fmt"
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

// HTTPError represents an error from the Coder API.
type HTTPError struct {
	*http.Response
}

func (e *HTTPError) Error() string {
	dump, err := httputil.DumpResponse(e.Response, false)
	if err != nil {
		return fmt.Sprintf("dump response: %+v", err)
	}

	var msg apiError
	// Try to decode the payload as an error, if it fails or if there is no error message,
	// return the response URL with the dump.
	if err := json.NewDecoder(e.Response.Body).Decode(&msg); err != nil || msg.Err.Msg == "" {
		return fmt.Sprintf("%s\n%s", e.Response.Request.URL, dump)
	}

	// If the payload was a in the expected error format with a message, include it.
	return fmt.Sprintf("%s\n%s%s", e.Response.Request.URL, dump, msg.Err.Msg)
}

func bodyError(resp *http.Response) error {
	return &HTTPError{resp}
}
