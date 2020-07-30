package entclient

import (
	"net/http"
	"net/http/httputil"

	"golang.org/x/xerrors"
)

// ErrNotFound describes an error case in which the request resource could not be found
var ErrNotFound = xerrors.Errorf("resource not found")

func bodyError(resp *http.Response) error {
	byt, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return xerrors.Errorf("dump response: %w", err)
	}
	return xerrors.Errorf("%s\n%s", resp.Request.URL, byt)
}
