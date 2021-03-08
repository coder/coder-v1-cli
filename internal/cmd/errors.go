package cmd

import (
	"encoding/json"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/clog"
	"golang.org/x/xerrors"
)

// handleAPIError attempts to convert an api error into a more detailed clog error.
// If it cannot, it will return the original error
func handleAPIError(origError error) error {
	var httpError *coder.HTTPError
	if !xerrors.As(origError, &httpError) {
		return origError // Return the original
	}

	ae, err := httpError.Payload()
	if err != nil {
		return origError // Return the original
	}

	switch ae.Err.Code {
	case "wac_template": // template parse errors
		type templatePayload struct {
			ErrorType string   `json:"error_type"`
			Msgs      []string `json:"messages"`
		}

		var p templatePayload
		err := json.Unmarshal(ae.Err.Details, &p)
		if err != nil {
			return origError
		}

		return clog.Error(p.ErrorType, p.Msgs...)
	case "verbose":
		type verbosePayload struct {
			Verbose string `json:"verbose"`
		}
		var p verbosePayload
		err := json.Unmarshal(ae.Err.Details, &p)
		if err != nil {
			return origError
		}

		return clog.Error(origError.Error(), p.Verbose)
	}

	return origError // Return the original
}
