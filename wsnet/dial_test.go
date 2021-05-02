package wsnet

import (
	"context"
	"errors"
	"fmt"
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
			// Likely they used TURN when they should have used TURN.
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

	t.Run("Pipe", func(t *testing.T) {
		connectAddr, listenAddr := createDumbBroker(t)
		listener, err := Listen(context.Background(), listenAddr)
		if err != nil {
			t.Error(err)
		}
		dialer, err := Dial(context.Background(), connectAddr, nil)
		if err != nil {
			t.Error(err)
		}
		go func() {
			conn, err := dialer.DialContext(context.Background(), "tcp", "localhost:40000")
			if err != nil {
				t.Error(err)
			}
			conn.Write([]byte("hello"))
		}()
		conn, err := listener.Accept()
		if err != nil {
			t.Error(err)
		}
		b := make([]byte, 5)
		_, _ = conn.Read(b)
		fmt.Printf("WE LEGIT GOT IT! %s\n", b)
	})
}
