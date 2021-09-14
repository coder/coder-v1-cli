package wsnet

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"nhooyr.io/websocket"
)

// TestShallowCopyDialOpts ensures the shallow copy for dial options works as expected
func TestShallowCopyDialOpts(t *testing.T) {
	opts := &websocket.DialOptions{}
	cpy := shallowCopyDialOpts(*opts)

	require.Equal(t, opts, cpy)
	cpy.HTTPHeader = http.Header{
		"test": []string{},
	}
	require.NotEqual(t, opts, cpy)

	headerCpy := shallowCopyDialOpts(*cpy)
	cpy.HTTPHeader["x"] = []string{"Random"}
	require.Equal(t, cpy, headerCpy)
}
