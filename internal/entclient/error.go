package entclient

import (
	"net/http"
	"net/http/httputil"

	"golang.org/x/xerrors"
)

func bodyError(resp *http.Response) error {
	byt, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return xerrors.Errorf("dump response: %w", err)
	}
	return xerrors.Errorf("%s\n%s", resp.Request.URL, byt)
}
