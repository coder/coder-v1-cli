package entclient

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"

	"golang.org/x/xerrors"
)

// ErrNotFound describes an error case in which the requested resource could not be found
var ErrNotFound = xerrors.Errorf("resource not found")

type apiError struct {
	Err struct {
		Msg string `json:"msg,required"`
	} `json:"error"`
}

func bodyError(resp *http.Response) error {
	byt, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return xerrors.Errorf("dump response: %w", err)
	}

	var msg apiError
	err = json.NewDecoder(resp.Body).Decode(&msg)
	if err != nil || msg.Err.Msg == "" {
		return xerrors.Errorf("%s\n%s", resp.Request.URL, byt)
	}
	return xerrors.Errorf("%s\n%s%s", resp.Request.URL, byt, msg.Err.Msg)
}
