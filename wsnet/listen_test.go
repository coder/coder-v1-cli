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

func TestListen(t *testing.T) {
	t.Run("Reconnect", func(t *testing.T) {
		keepAliveInterval = 50 * time.Millisecond

		var (
			connCh = make(chan interface{})
			mux    = http.NewServeMux()
			srv    = http.Server{
				Handler: mux,
			}
		)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			_, err := websocket.Accept(w, r, nil)
			if err != nil {
				t.Error(err)
				return
			}
			connCh <- struct{}{}
		})

		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			t.Error(err)
			return
		}
		go srv.Serve(listener)

		addr := listener.Addr()
		broker := fmt.Sprintf("http://%s/", addr.String())

		_, err = Listen(context.Background(), broker)
		if err != nil {
			t.Error(err)
			return
		}
		<-connCh
		_ = listener.Close()
		listener, err = net.Listen("tcp4", addr.String())
		if err != nil {
			t.Error(err)
			return
		}
		go srv.Serve(listener)
		<-connCh
	})
}
