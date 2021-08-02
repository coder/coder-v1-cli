package coder

import (
	"context"
	"net/http"

	"github.com/pion/webrtc/v3"
)

type getICEServersRes struct {
	Data []webrtc.ICEServer `json:"data"`
}

// ICEServers fetches the list of ICE servers advertised by the deployment.
func (c *DefaultClient) ICEServers(ctx context.Context) ([]webrtc.ICEServer, error) {
	var res getICEServersRes
	err := c.requestBody(ctx, http.MethodGet, "/api/private/webrtc/ice", nil, &res)
	if err != nil {
		return nil, err
	}

	return res.Data, nil
}
