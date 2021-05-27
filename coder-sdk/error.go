package coder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/xerrors"
)

// ErrNotFound describes an error case in which the requested resource could not be found.
var ErrNotFound = xerrors.New("resource not found")

// ErrPermissions describes an error case in which the requester has insufficient permissions to access the requested resource.
var ErrPermissions = xerrors.New("insufficient permissions")

// ErrAuthentication describes the error case in which the requester has invalid authentication.
var ErrAuthentication = xerrors.New("invalid authentication")

// APIError is the expected payload format for API errors.
type APIError struct {
	Err APIErrorMsg `json:"error"`
}

// APIErrorMsg contains the rich error information returned by API errors.
type APIErrorMsg struct {
	Msg     string          `json:"msg"`
	Code    string          `json:"code"`
	Details json.RawMessage `json:"details"`
}

// NewHTTPError reads the response body and stores metadata
// about the response in order to be unpacked into
// an *APIError.
func NewHTTPError(resp *http.Response) *HTTPError {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, resp.Body)
	if err != nil {
		return &HTTPError{
			cachedErr: err,
		}
	}
	return &HTTPError{
		url:        resp.Request.URL.String(),
		statusCode: resp.StatusCode,
		body:       buf.Bytes(),
	}
}

// HTTPError represents an error from the Coder API.
type HTTPError struct {
	url        string
	statusCode int
	body       []byte
	cached     *APIError
	cachedErr  error
}

// Payload decode the response body into the standard error structure. The `details`
// section is stored as a raw json, and type depends on the `code` field.
func (e *HTTPError) Payload() (*APIError, error) {
	var msg APIError
	if e.cached != nil || e.cachedErr != nil {
		return e.cached, e.cachedErr
	}

	// Try to decode the payload as an error, if it fails or if there is no error message,
	// return the response URL with the status.
	if err := json.Unmarshal(e.body, &msg); err != nil {
		e.cachedErr = err
		return nil, err
	}

	e.cached = &msg
	return &msg, nil
}

func (e *HTTPError) StatusCode() int {
	return e.statusCode
}

func (e *HTTPError) Error() string {
	apiErr, err := e.Payload()
	if err != nil || apiErr.Err.Msg == "" {
		return fmt.Sprintf("%s: %d %s", e.url, e.statusCode, http.StatusText(e.statusCode))
	}

	// If the payload was a in the expected error format with a message, include it.
	return apiErr.Err.Msg
}
