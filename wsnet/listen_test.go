package wsnet

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

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
			srv    = http.Server{
				Handler: mux,
			}
		)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			ws, err := websocket.Accept(w, r, nil)
			if err != nil {
				t.Error(err)
				return
			}
			connCh <- ws
		})

		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			t.Error(err)
			return
		}
		go func() {
			_ = srv.Serve(listener)
		}()
		addr := listener.Addr()
		broker := fmt.Sprintf("http://%s/", addr.String())

		_, err = Listen(context.Background(), broker, nil)
		if err != nil {
			t.Error(err)
			return
		}
		conn := <-connCh
		_ = listener.Close()
		// We need to close the connection too... closing a TCP
		// listener does not close active local connections.
		_ = conn.Close(websocket.StatusGoingAway, "")

		// At least a few retry attempts should be had...
		time.Sleep(connectionRetryInterval * 5)

		listener, err = net.Listen("tcp4", addr.String())
		if err != nil {
			t.Error(err)
			return
		}
		go func() {
			_ = srv.Serve(listener)
		}()
		<-connCh
	})
}
