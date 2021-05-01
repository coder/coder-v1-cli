package wsnet

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	"cdr.dev/slog/sloggers/slogtest/assert"
	"github.com/pion/dtls/v2"
	"github.com/pion/ice/v2"
	"github.com/pion/turn/v2"
)

func listenTURN(t *testing.T, protocol ice.ProtoType, pass string, useTLS bool) string {
	var (
		listeners   = []turn.ListenerConfig{}
		pcListeners = []turn.PacketConnConfig{}
		tlsConfig   *tls.Config
		relay       = &turn.RelayAddressGeneratorStatic{
			RelayAddress: net.ParseIP("127.0.0.1"),
			Address:      "127.0.0.1",
		}
		listenAddr net.Addr
	)
	if useTLS {
		tlsConfig = generateTLSConfig(t)
	}

	switch protocol {
	case ice.ProtoTypeTCP:
		var (
			tcpListener net.Listener
			err         error
		)
		if useTLS {
			tcpListener, err = tls.Listen("tcp4", "0.0.0.0:0", tlsConfig)
		} else {
			tcpListener, err = net.Listen("tcp4", "0.0.0.0:0")
		}
		if err != nil {
			t.Error(err)
		}
		listenAddr = tcpListener.Addr()
		listeners = append(listeners, turn.ListenerConfig{
			Listener:              tcpListener,
			RelayAddressGenerator: relay,
		})
	case ice.ProtoTypeUDP:
		if useTLS {
			addr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:0")
			if err != nil {
				t.Error(err)
			}
			udpListener, err := dtls.Listen("udp4", addr, &dtls.Config{
				Certificates: tlsConfig.Certificates,
			})
			if err != nil {
				t.Error(err)
			}
			listenAddr = udpListener.Addr()
			listeners = append(listeners, turn.ListenerConfig{
				Listener:              udpListener,
				RelayAddressGenerator: relay,
			})
		} else {
			udpListener, err := net.ListenPacket("udp4", "0.0.0.0:0")
			if err != nil {
				t.Error(err)
			}
			listenAddr = udpListener.LocalAddr()
			pcListeners = append(pcListeners, turn.PacketConnConfig{
				PacketConn:            udpListener,
				RelayAddressGenerator: relay,
			})
		}
	}

	t.Cleanup(func() {
		for _, l := range listeners {
			l.Listener.Close()
		}
		for _, l := range pcListeners {
			l.PacketConn.Close()
		}
	})

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
		srv.Close()
	})

	scheme := "turn"
	if useTLS {
		scheme = "turns"
	}
	return fmt.Sprintf("%s:%s", scheme, listenAddr.String())
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
