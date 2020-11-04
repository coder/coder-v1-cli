package coder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	"golang.org/x/xerrors"
)

// ErrNotFound describes an error case in which the requested resource could not be found.
var ErrNotFound = xerrors.New("resource not found")

// ErrPermissions describes an error case in which the requester has insufficient permissions to access the requested resource.
var ErrPermissions = xerrors.New("insufficient permissions")

// ErrAuthentication describes the error case in which the requester has invalid authentication.
var ErrAuthentication = xerrors.New("invalid authentication")

// APIError is the expected payload format for our errors.
type APIError struct {
	Err APIErrorMsg `json:"error"`
}

// APIErrorMsg contains the rich error information returned by API errors.
type APIErrorMsg struct {
	Msg string `json:"msg"`
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

	var msg APIError
	// Try to decode the payload as an error, if it fails or if there is no error message,
	// return the response URL with the dump.
	if err := json.NewDecoder(e.Response.Body).Decode(&msg); err != nil || msg.Err.Msg == "" {
		return fmt.Sprintf("%s\n%s", e.Response.Request.URL, dump)
	}

	// If the payload was a in the expected error format with a message, include it.
	return msg.Err.Msg
}

func bodyError(resp *http.Response) error {
	return &HTTPError{resp}
}
