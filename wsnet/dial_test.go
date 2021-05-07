package wsnet

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/pion/webrtc/v3"
)

func ExampleDial_basic() {
	servers := []webrtc.ICEServer{{
		URLs:           []string{"turns:master.cdr.dev"},
		Username:       "kyle",
		Credential:     "pass",
		CredentialType: webrtc.ICECredentialTypePassword,
	}}

	for _, server := range servers {
		err := DialICE(server, nil)
		if errors.Is(err, ErrInvalidCredentials) {
			// You could do something...
		}
		if errors.Is(err, ErrMismatchedProtocol) {
			// Likely they used TURNS when they should have used TURN.
			// Or they could have used TURN instead of TURNS.
		}
	}

	dialer, err := DialWebsocket(context.Background(), "wss://master.cdr.dev/agent/workspace/connect", servers)
	if err != nil {
		// Do something...
	}
	conn, err := dialer.DialContext(context.Background(), "tcp", "localhost:13337")
	if err != nil {
		// Something...
	}
	defer conn.Close()
	// You now have access to the proxied remote port in `conn`.
}

func TestDial(t *testing.T) {
	t.Run("Ping", func(t *testing.T) {
		connectAddr, listenAddr := createDumbBroker(t)
		_, err := Listen(context.Background(), listenAddr)
		if err != nil {
			t.Error(err)
		}
		dialer, err := DialWebsocket(context.Background(), connectAddr, nil)
		if err != nil {
			t.Error(err)
		}
		err = dialer.Ping(context.Background())
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("OPError", func(t *testing.T) {
		connectAddr, listenAddr := createDumbBroker(t)
		_, err := Listen(context.Background(), listenAddr)
		if err != nil {
			t.Error(err)
		}
		dialer, err := DialWebsocket(context.Background(), connectAddr, nil)
		if err != nil {
			t.Error(err)
		}
		_, err = dialer.DialContext(context.Background(), "tcp", "localhost:100")
		if err == nil {
			t.Error("should have gotten err")
			return
		}
		_, ok := err.(*net.OpError)
		if !ok {
			t.Error("invalid error type returned")
			return
		}
	})

	t.Run("Proxy", func(t *testing.T) {
		listener, err := net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			t.Error(err)
			return
		}
		msg := []byte("Hello!")
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				t.Error(err)
			}
			_, _ = conn.Write(msg)
		}()

		connectAddr, listenAddr := createDumbBroker(t)
		_, err = Listen(context.Background(), listenAddr)
		if err != nil {
			t.Error(err)
		}
		dialer, err := DialWebsocket(context.Background(), connectAddr, nil)
		if err != nil {
			t.Error(err)
		}
		conn, err := dialer.DialContext(context.Background(), listener.Addr().Network(), listener.Addr().String())
		if err != nil {
			t.Error(err)
			return
		}
		rec := make([]byte, len(msg))
		_, err = conn.Read(rec)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(msg, rec) {
			t.Error("bytes were different", string(msg), string(rec))
		}
	})

	// Expect that we'd get an EOF on the server closing.
	t.Run("EOF on Close", func(t *testing.T) {
		listener, err := net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			t.Error(err)
			return
		}
		connectAddr, listenAddr := createDumbBroker(t)
		srv, err := Listen(context.Background(), listenAddr)
		if err != nil {
			t.Error(err)
		}
		dialer, err := DialWebsocket(context.Background(), connectAddr, nil)
		if err != nil {
			t.Error(err)
		}
		conn, err := dialer.DialContext(context.Background(), listener.Addr().Network(), listener.Addr().String())
		if err != nil {
			t.Error(err)
			return
		}
		go srv.Close()
		rec := make([]byte, 16)
		_, err = conn.Read(rec)
		if !errors.Is(err, io.EOF) {
			t.Error(err)
			return
		}
	})
}
