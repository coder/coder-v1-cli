package wsnet

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/hashicorp/yamux"
	"nhooyr.io/websocket"
)

func listenBroker(t *testing.T) (connectAddr string, listenAddr string) {
	listener, err := net.Listen("tcp4", "0.0.0.0:0")
	if err != nil {
		t.Error(err)
	}
	t.Cleanup(func() {
		listener.Close()
	})
	mux := http.NewServeMux()
	var sess *yamux.Session
	mux.HandleFunc("/listen", func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Error(err)
		}
		nc := websocket.NetConn(context.Background(), c, websocket.MessageBinary)
		oc, err := sess.Open()
		if err != nil {
			t.Error(err)
		}
		go io.Copy(nc, oc)
		io.Copy(oc, nc)
	})
	mux.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Error(err)
		}
		nc := websocket.NetConn(context.Background(), c, websocket.MessageBinary)
		sess, err = yamux.Client(nc, nil)
		if err != nil {
			t.Error(err)
		}
	})

	s := http.Server{
		Handler: mux,
	}
	go s.Serve(listener)
	return fmt.Sprintf("ws://%s/connect", listener.Addr()), fmt.Sprintf("ws://%s/listen", listener.Addr())
}
