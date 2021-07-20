package wsnet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cdr.dev/slog/sloggers/slogtest"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

func init() {
	// We override this value to make tests faster.
	connectionRetryInterval = 10 * time.Millisecond
}

func TestListen(t *testing.T) {
	t.Run("Reconnect", func(t *testing.T) {
		var (
			connCh = make(chan *websocket.Conn)
			mux    = http.NewServeMux()
		)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			ws, err := websocket.Accept(w, r, nil)
			if err != nil {
				t.Error(err)
				return
			}
			connCh <- ws
		})

		s := httptest.NewServer(mux)
		defer s.Close()

		l, err := Listen(context.Background(), slogtest.Make(t, nil), s.URL, "")
		require.NoError(t, err)
		defer l.Close()
		conn := <-connCh

		// Kill the server connection.
		err = conn.Close(websocket.StatusGoingAway, "")
		require.NoError(t, err)

		// At least a few retry attempts should be had...
		time.Sleep(connectionRetryInterval * 5)
		<-connCh
	})
}
