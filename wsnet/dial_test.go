package wsnet

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"cdr.dev/slog/sloggers/slogtest"
	"github.com/pion/ice/v2"
	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	}, nil)
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
	t.Run("Timeout", func(t *testing.T) {
		t.Parallel()

		connectAddr, _ := createDumbBroker(t)

		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancelFunc()
		dialer, err := DialWebsocket(ctx, connectAddr, nil, nil)
		require.True(t, errors.Is(err, context.DeadlineExceeded))
		require.Error(t, dialer.conn.Close(), "already wrote close")
	})

	t.Run("Ping", func(t *testing.T) {
		t.Parallel()

		connectAddr, listenAddr := createDumbBroker(t)
		l, err := Listen(context.Background(), slogtest.Make(t, nil), listenAddr, "")
		require.NoError(t, err)
		defer l.Close()

		dialer, err := DialWebsocket(context.Background(), connectAddr, nil, nil)
		require.NoError(t, err)

		err = dialer.Ping(context.Background())
		require.NoError(t, err)
	})

	t.Run("Ping Close", func(t *testing.T) {
		t.Parallel()

		connectAddr, listenAddr := createDumbBroker(t)
		l, err := Listen(context.Background(), slogtest.Make(t, nil), listenAddr, "")
		require.NoError(t, err)
		defer l.Close()

		turnAddr, closeTurn := createTURNServer(t, ice.SchemeTypeTURN)
		dialer, err := DialWebsocket(context.Background(), connectAddr, &DialOptions{
			ICEServers: []webrtc.ICEServer{{
				URLs:           []string{fmt.Sprintf("turn:%s", turnAddr)},
				Username:       "example",
				Credential:     testPass,
				CredentialType: webrtc.ICECredentialTypePassword,
			}},
		}, nil)
		require.NoError(t, err)

		_ = dialer.Ping(context.Background())
		closeTurn()
		err = dialer.Ping(context.Background())
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("OPError", func(t *testing.T) {
		t.Parallel()

		connectAddr, listenAddr := createDumbBroker(t)
		l, err := Listen(context.Background(), slogtest.Make(t, nil), listenAddr, "")
		require.NoError(t, err)
		defer l.Close()

		dialer, err := DialWebsocket(context.Background(), connectAddr, nil, nil)
		require.NoError(t, err)

		_, err = dialer.DialContext(context.Background(), "tcp", "localhost:100")
		assert.Error(t, err)

		// Double pointer intended.
		netErr := &net.OpError{}
		assert.ErrorAs(t, err, &netErr)
	})

	t.Run("Proxy", func(t *testing.T) {
		t.Parallel()

		listener, err := net.Listen("tcp", "0.0.0.0:0")
		require.NoError(t, err)

		msg := []byte("Hello!")
		go func() {
			conn, err := listener.Accept()
			require.NoError(t, err)

			_, _ = conn.Write(msg)
		}()

		connectAddr, listenAddr := createDumbBroker(t)
		l, err := Listen(context.Background(), slogtest.Make(t, nil), listenAddr, "")
		require.NoError(t, err)
		defer l.Close()

		dialer, err := DialWebsocket(context.Background(), connectAddr, nil, nil)
		require.NoError(t, err)

		conn, err := dialer.DialContext(context.Background(), listener.Addr().Network(), listener.Addr().String())
		require.NoError(t, err)

		rec := make([]byte, len(msg))
		_, err = conn.Read(rec)
		require.NoError(t, err)

		assert.Equal(t, msg, rec)
	})

	// Expect that we'd get an EOF on the server closing.
	t.Run("EOF on Close", func(t *testing.T) {
		t.Parallel()

		listener, err := net.Listen("tcp", "0.0.0.0:0")
		require.NoError(t, err)
		go func() {
			_, _ = listener.Accept()
		}()

		connectAddr, listenAddr := createDumbBroker(t)
		l, err := Listen(context.Background(), slogtest.Make(t, nil), listenAddr, "")
		require.NoError(t, err)
		defer l.Close()

		dialer, err := DialWebsocket(context.Background(), connectAddr, nil, nil)
		require.NoError(t, err)

		conn, err := dialer.DialContext(context.Background(), listener.Addr().Network(), listener.Addr().String())
		require.NoError(t, err)

		go l.Close()
		rec := make([]byte, 16)
		_, err = conn.Read(rec)
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("Disconnect", func(t *testing.T) {
		t.Parallel()

		connectAddr, listenAddr := createDumbBroker(t)
		l, err := Listen(context.Background(), slogtest.Make(t, nil), listenAddr, "")
		require.NoError(t, err)
		defer l.Close()

		dialer, err := DialWebsocket(context.Background(), connectAddr, nil, nil)
		require.NoError(t, err)

		err = dialer.Close()
		require.NoError(t, err)

		err = dialer.Ping(context.Background())
		assert.ErrorIs(t, err, webrtc.ErrConnectionClosed)
	})

	t.Run("Disconnect DialContext", func(t *testing.T) {
		t.Parallel()

		tcpListener, err := net.Listen("tcp", "0.0.0.0:0")
		require.NoError(t, err)
		go func() {
			_, _ = tcpListener.Accept()
		}()

		connectAddr, listenAddr := createDumbBroker(t)
		l, err := Listen(context.Background(), slogtest.Make(t, nil), listenAddr, "")
		require.NoError(t, err)
		defer l.Close()

		turnAddr, closeTurn := createTURNServer(t, ice.SchemeTypeTURN)
		dialer, err := DialWebsocket(context.Background(), connectAddr, &DialOptions{
			ICEServers: []webrtc.ICEServer{{
				URLs:           []string{fmt.Sprintf("turn:%s", turnAddr)},
				Username:       "example",
				Credential:     testPass,
				CredentialType: webrtc.ICECredentialTypePassword,
			}},
		}, nil)
		require.NoError(t, err)

		conn, err := dialer.DialContext(context.Background(), "tcp", tcpListener.Addr().String())
		require.NoError(t, err)

		// Close the TURN server before reading...
		// WebRTC connections take a few seconds to timeout.
		closeTurn()
		_, err = conn.Read(make([]byte, 16))
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("Active Connections", func(t *testing.T) {
		t.Parallel()

		listener, err := net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			t.Error(err)
			return
		}
		go func() {
			_, _ = listener.Accept()
		}()
		connectAddr, listenAddr := createDumbBroker(t)
		_, err = Listen(context.Background(), slogtest.Make(t, nil), listenAddr, "")
		if err != nil {
			t.Error(err)
			return
		}
		dialer, err := DialWebsocket(context.Background(), connectAddr, nil, nil)
		if err != nil {
			t.Error(err)
		}
		conn, _ := dialer.DialContext(context.Background(), listener.Addr().Network(), listener.Addr().String())
		assert.Equal(t, 1, dialer.activeConnections())
		_ = conn.Close()
		assert.Equal(t, 0, dialer.activeConnections())
		_, _ = dialer.DialContext(context.Background(), listener.Addr().Network(), listener.Addr().String())
		conn, _ = dialer.DialContext(context.Background(), listener.Addr().Network(), listener.Addr().String())
		assert.Equal(t, 2, dialer.activeConnections())
		_ = conn.Close()
		assert.Equal(t, 1, dialer.activeConnections())
	})

	t.Run("Close Listeners on Disconnect", func(t *testing.T) {
		t.Parallel()

		tcpListener, err := net.Listen("tcp", "0.0.0.0:0")
		require.NoError(t, err)
		go func() {
			_, _ = tcpListener.Accept()
		}()

		connectAddr, listenAddr := createDumbBroker(t)
		l, err := Listen(context.Background(), slogtest.Make(t, nil), listenAddr, "")
		require.NoError(t, err)

		turnAddr, closeTurn := createTURNServer(t, ice.SchemeTypeTURN)
		dialer, err := DialWebsocket(context.Background(), connectAddr, &DialOptions{
			ICEServers: []webrtc.ICEServer{{
				URLs:           []string{fmt.Sprintf("turn:%s", turnAddr)},
				Username:       "example",
				Credential:     testPass,
				CredentialType: webrtc.ICECredentialTypePassword,
			}},
		}, nil)
		require.NoError(t, err)

		_, err = dialer.DialContext(context.Background(), "tcp", tcpListener.Addr().String())
		require.NoError(t, err)

		closeTurn()

		for i := 0; i < 15; i++ {
			time.Sleep(time.Second)

			if len(l.(*listener).connClosers) == 0 {
				break
			}
		}

		assert.Equal(t, 0, len(l.(*listener).connClosers))
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
	l, err := Listen(context.Background(), slogtest.Make(b, nil), listenAddr, "")
	if err != nil {
		b.Error(err)
		return
	}
	defer l.Close()

	dialer, err := DialWebsocket(context.Background(), connectAddr, nil, nil)
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
