package wsnet

import (
	"context"
	"fmt"
	"testing"

	"github.com/pion/ice/v2"
	"github.com/pion/turn/v2"
	"github.com/pion/webrtc/v3"
)

func TestDial(t *testing.T) {
	t.Run("Example", func(t *testing.T) {
		connectAddr, _ := listenBroker(t)
		turnAddr := listenTURN(t, ice.ProtoTypeTCP, "wowie", true)

		dialer, err := Dial(context.Background(), connectAddr, &DialConfig{
			[]ICEServer{{
				URLs:           []string{turnAddr},
				Username:       "insecure",
				Credential:     "pass",
				CredentialType: webrtc.ICECredentialTypePassword,
			}},
		})
		if err != nil {
			t.Error(err)
		}
		fmt.Printf("Dialer: %+v\n", dialer)
	})
}

func testTURN() {

	turn.NewServer(turn.ServerConfig{})
}
