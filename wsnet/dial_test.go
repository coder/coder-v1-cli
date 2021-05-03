package wsnet

import (
	"context"
	"errors"
	"io"
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
		err := DialICE(server, DefaultICETimeout)
		if errors.Is(err, ErrInvalidCredentials) {
			// You could do something...
		}
		if errors.Is(err, ErrMismatchedProtocol) {
			// Likely they used TURNS when they should have used TURN.
			// Or they could have used TURN instead of TURNS.
		}
	}

	dialer, err := Dial(context.Background(), "wss://master.cdr.dev/agent/workspace/connect", &DialConfig{
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

func TestDial(t *testing.T) {
	t.Run("Ping", func(t *testing.T) {
		connectAddr, listenAddr := createDumbBroker(t)
		_, err := Listen(context.Background(), listenAddr)
		if err != nil {
			t.Error(err)
		}
		dialer, err := Dial(context.Background(), connectAddr, nil)
		if err != nil {
			t.Error(err)
		}
		err = dialer.Ping(context.Background())
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Disconnect", func(t *testing.T) {
		connectAddr, listenAddr := createDumbBroker(t)
		listener, err := Listen(context.Background(), listenAddr)
		if err != nil {
			t.Error(err)
		}
		go func() {
			c, _ := listener.Accept()
			c.Close()
		}()
		dialer, err := Dial(context.Background(), connectAddr, nil)
		if err != nil {
			t.Error(err)
		}
		conn, err := dialer.DialContext(context.Background(), "tcp", "example")
		if err != nil {
			t.Error(err)
		}
		b := make([]byte, 16)
		_, err = conn.Read(b)
		if err != io.EOF {
			t.Error(err)
		}
	})
}
