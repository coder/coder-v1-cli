package wsnet

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/pion/ice/v2"
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

	dialer, err := DialWebsocket(context.Background(), "wss://master.cdr.dev/agent/workspace/connect", &DialOptions{
		ICEServers: servers,
	})
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

// nolint:gocognit,gocyclo
func TestDial(t *testing.T) {
	t.Run("Ping", func(t *testing.T) {
		connectAddr, listenAddr := createDumbBroker(t)
		_, err := Listen(context.Background(), listenAddr, nil)
		if err != nil {
			t.Error(err)
			return
		}
		dialer, err := DialWebsocket(context.Background(), connectAddr, nil)
		if err != nil {
			t.Error(err)
			return
		}
		err = dialer.Ping(context.Background())
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("OPError", func(t *testing.T) {
		connectAddr, listenAddr := createDumbBroker(t)
		_, err := Listen(context.Background(), listenAddr, nil)
		if err != nil {
			t.Error(err)
			return
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
		_, err = Listen(context.Background(), listenAddr, nil)
		if err != nil {
			t.Error(err)
			return
		}
		dialer, err := DialWebsocket(context.Background(), connectAddr, nil)
		if err != nil {
			t.Error(err)
			return
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
		go func() {
			_, _ = listener.Accept()
		}()
		connectAddr, listenAddr := createDumbBroker(t)
		srv, err := Listen(context.Background(), listenAddr, nil)
		if err != nil {
			t.Error(err)
			return
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

	t.Run("Disconnect", func(t *testing.T) {
		connectAddr, listenAddr := createDumbBroker(t)
		_, err := Listen(context.Background(), listenAddr, nil)
		if err != nil {
			t.Error(err)
			return
		}
		dialer, err := DialWebsocket(context.Background(), connectAddr, nil)
		if err != nil {
			t.Error(err)
			return
		}
		err = dialer.Close()
		if err != nil {
			t.Error(err)
			return
		}
		err = dialer.Ping(context.Background())
		if err != webrtc.ErrConnectionClosed {
			t.Error(err)
		}
	})

	t.Run("Disconnect DialContext", func(t *testing.T) {
		tcpListener, err := net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			t.Error(err)
			return
		}
		go func() {
			_, _ = tcpListener.Accept()
		}()

		connectAddr, listenAddr := createDumbBroker(t)
		_, err = Listen(context.Background(), listenAddr, nil)
		if err != nil {
			t.Error(err)
			return
		}
		turnAddr, closeTurn := createTURNServer(t, ice.SchemeTypeTURN)
		dialer, err := DialWebsocket(context.Background(), connectAddr, &DialOptions{
			ICEServers: []webrtc.ICEServer{{
				URLs:           []string{fmt.Sprintf("turn:%s", turnAddr)},
				Username:       "example",
				Credential:     testPass,
				CredentialType: webrtc.ICECredentialTypePassword,
			}},
		})
		if err != nil {
			t.Error(err)
			return
		}
		conn, err := dialer.DialContext(context.Background(), "tcp", tcpListener.Addr().String())
		if err != nil {
			t.Error(err)
			return
		}
		// Close the TURN server before reading...
		// WebRTC connections take a few seconds to timeout.
		closeTurn()
		_, err = conn.Read(make([]byte, 16))
		if err != io.EOF {
			t.Error(err)
			return
		}
	})

	t.Run("Closed", func(t *testing.T) {
		connectAddr, listenAddr := createDumbBroker(t)
		_, err := Listen(context.Background(), listenAddr, nil)
		if err != nil {
			t.Error(err)
			return
		}
		dialer, err := DialWebsocket(context.Background(), connectAddr, nil)
		if err != nil {
			t.Error(err)
			return
		}
		go func() {
			_ = dialer.Close()
		}()
		select {
		case <-dialer.Closed():
		case <-time.NewTimer(time.Second).C:
			t.Error("didn't close in time")
		}
	})
}

func BenchmarkThroughput(b *testing.B) {
	sizes := []int64{
		4,
		16,
		128,
		256,
		1024,
		4096,
		16384,
		32768,
	}

	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		b.Error(err)
		return
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				b.Error(err)
				return
			}
			go func() {
				_, _ = io.Copy(io.Discard, conn)
			}()
		}
	}()
	connectAddr, listenAddr := createDumbBroker(b)
	_, err = Listen(context.Background(), listenAddr, nil)
	if err != nil {
		b.Error(err)
		return
	}

	dialer, err := DialWebsocket(context.Background(), connectAddr, nil)
	if err != nil {
		b.Error(err)
		return
	}
	for _, size := range sizes {
		size := size
		bytes := make([]byte, size)
		_, _ = rand.Read(bytes)
		b.Run("Rand"+strconv.Itoa(int(size)), func(b *testing.B) {
			b.SetBytes(size)
			b.ReportAllocs()

			conn, err := dialer.DialContext(context.Background(), listener.Addr().Network(), listener.Addr().String())
			if err != nil {
				b.Error(err)
				return
			}
			defer conn.Close()

			for i := 0; i < b.N; i++ {
				_, err := conn.Write(bytes)
				if err != nil {
					b.Error(err)
					break
				}
			}
		})
	}
}
