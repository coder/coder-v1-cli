package client

import (
	"net/http"
	"net/http/httputil"

	"golang.org/x/xerrors"
)

func bodyError(resp *http.Response) error {
	byt, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return xerrors.Errorf("dump response: %w", err)
	}
	return xerrors.Errorf("%s", byt)
}
