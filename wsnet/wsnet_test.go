package wsnet

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"cdr.dev/slog/sloggers/slogtest/assert"
	"github.com/hashicorp/yamux"
	"github.com/pion/ice/v2"
	"github.com/pion/turn/v2"
	"nhooyr.io/websocket"
)

// createDumbBroker proxies sockets between /listen and /connect
// to emulate an authenticated WebSocket pair.
func createDumbBroker(t *testing.T) (connectAddr string, listenAddr string) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Error(err)
	}
	t.Cleanup(func() {
		listener.Close()
	})
	var (
		mux  = http.NewServeMux()
		sess *yamux.Session
		mut  sync.Mutex
	)
	mux.HandleFunc("/listen", func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Error(err)
		}
		nc := websocket.NetConn(context.Background(), c, websocket.MessageBinary)
		mut.Lock()
		defer mut.Unlock()
		sess, err = yamux.Client(nc, nil)
		if err != nil {
			t.Error(err)
		}
	})
	mux.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		if sess == nil {
			t.Error("listen not called")
			return
		}
		nc := websocket.NetConn(context.Background(), c, websocket.MessageBinary)
		mut.Lock()
		defer mut.Unlock()
		oc, err := sess.Open()
		if err != nil {
			t.Error(err)
		}
		go func() {
			_, _ = io.Copy(nc, oc)
		}()
		_, _ = io.Copy(oc, nc)
	})

	s := http.Server{
		Handler: mux,
	}
	go func() {
		_ = s.Serve(listener)
	}()
	return fmt.Sprintf("ws://%s/connect", listener.Addr()), fmt.Sprintf("ws://%s/listen", listener.Addr())
}

// createTURNServer allocates a TURN server and returns the address.
func createTURNServer(t *testing.T, server ice.SchemeType, pass string) string {
	var (
		listeners   []turn.ListenerConfig
		pcListeners []turn.PacketConnConfig
		relay       = &turn.RelayAddressGeneratorStatic{
			RelayAddress: net.ParseIP("127.0.0.1"),
			Address:      "127.0.0.1",
		}
		listenAddr net.Addr
	)
	url, _ := ice.ParseURL(fmt.Sprintf("%s:localhost", server))

	switch url.Proto {
	case ice.ProtoTypeTCP:
		var (
			tcpListener net.Listener
			err         error
		)
		if url.IsSecure() {
			tcpListener, err = tls.Listen("tcp4", "127.0.0.1:0", generateTLSConfig(t))
		} else {
			tcpListener, err = net.Listen("tcp4", "127.0.0.1:0")
		}
		if err != nil {
			t.Error(err)
		}
		listenAddr = tcpListener.Addr()
		listeners = []turn.ListenerConfig{{
			Listener:              tcpListener,
			RelayAddressGenerator: relay,
		}}
	case ice.ProtoTypeUDP:
		udpListener, err := net.ListenPacket("udp4", "127.0.0.1:0")
		if err != nil {
			t.Error(err)
		}
		listenAddr = udpListener.LocalAddr()
		pcListeners = []turn.PacketConnConfig{{
			PacketConn:            udpListener,
			RelayAddressGenerator: relay,
		}}
	}

	srv, err := turn.NewServer(turn.ServerConfig{
		PacketConnConfigs: pcListeners,
		ListenerConfigs:   listeners,
		Realm:             "coder",
		AuthHandler: func(username, realm string, srcAddr net.Addr) (key []byte, ok bool) {
			return turn.GenerateAuthKey(username, realm, pass), true
		},
	})
	if err != nil {
		t.Error(err)
	}
	t.Cleanup(func() {
		for _, l := range listeners {
			l.Listener.Close()
		}
		for _, l := range pcListeners {
			l.PacketConn.Close()
		}
		srv.Close()
	})

	return listenAddr.String()
}

func generateTLSConfig(t testing.TB) *tls.Config {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Success(t, "generate key", err)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	assert.Success(t, "create certificate", err)
	certBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	assert.Success(t, "marshal private key", err)
	keyBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes})
	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	assert.Success(t, "convert to key pair", err)
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
}
