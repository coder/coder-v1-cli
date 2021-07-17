package wsnet

import (
	"fmt"
	"math/bits"
	"net"
	"strconv"
	"strings"

	"github.com/pion/webrtc/v3"
)

// DialPolicy a single network + address + port combinations that a connection
// is permitted to use.
type DialPolicy struct {
	// If network is empty, it applies to all networks.
	Network string `json:"network"`
	// Host is the IP or hostname of the address. It should not contain the
	// port.If empty, it applies to all hosts. "localhost", [::1], and any IPv4
	// address under "127.0.0.0/8" can be used interchangeably.
	Host string `json:"address"`
	// If port is 0, it applies to all ports.
	Port uint16 `json:"port"`
}

// permits checks if a DialPolicy permits a specific network + host + port
// combination. The host must be put through normalizeHost first.
func (p DialPolicy) permits(network, host string, port uint16) bool {
	if p.Network != "" && p.Network != network {
		return false
	}
	if p.Host != "" && canonicalizeHost(p.Host) != host {
		return false
	}
	if p.Port != 0 && p.Port != port {
		return false
	}

	return true
}

// BrokerMessage is used for brokering a dialer and listener.
//
// Dialers initiate an exchange by providing an Offer,
// along with a list of ICE servers for the listener to
// peer with.
//
// The listener should respond with an offer, then both
// sides can begin exchanging candidates.
type BrokerMessage struct {
	// Dialer -> Listener
	Offer   *webrtc.SessionDescription `json:"offer"`
	Servers []webrtc.ICEServer         `json:"servers"`

	// Policies denote which addresses the client can dial. If empty or nil, all
	// addresses are permitted.
	Policies []DialPolicy `json:"ports"`

	// Listener -> Dialer
	Error  string                     `json:"error"`
	Answer *webrtc.SessionDescription `json:"answer"`

	// Bidirectional
	Candidate string `json:"candidate"`
}

// getAddress parses the data channel's protocol into an address suitable for
// net.Dial. It also verifies that the BrokerMessage permits connecting to said
// address.
func (msg BrokerMessage) getAddress(protocol string) (netwk, addr string, err error) {
	parts := strings.SplitN(protocol, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid dial address: %v", protocol)
	}
	host, port, err := net.SplitHostPort(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("invalid dial address: %v", protocol)
	}

	var (
		network    = parts[0]
		normalHost = canonicalizeHost(host)
		// Still return the original host value, not the canonical value.
		fullAddr = net.JoinHostPort(host, port)
	)
	if network == "" {
		return "", "", fmt.Errorf("invalid dial address %q network: %v", protocol, network)
	}
	if host == "" {
		return "", "", fmt.Errorf("invalid dial address %q host: %v", protocol, host)
	}

	portParsed, err := strconv.Atoi(port)
	if err != nil || portParsed < 0 || bits.Len(uint(portParsed)) > 16 {
		return "", "", fmt.Errorf("invalid dial address %q port: %v", protocol, port)
	}
	if len(msg.Policies) == 0 {
		return network, fullAddr, nil
	}

	portParsedU16 := uint16(portParsed)
	for _, p := range msg.Policies {
		if p.permits(network, normalHost, portParsedU16) {
			return network, fullAddr, nil
		}
	}

	return "", "", fmt.Errorf("connections are not permitted to %q by policy", protocol)
}

// canonicalizeHost converts all representations of "localhost" to "localhost".
func canonicalizeHost(addr string) string {
	addr = strings.TrimPrefix(addr, "[")
	addr = strings.TrimSuffix(addr, "]")

	ip := net.ParseIP(addr)
	if ip == nil {
		return addr
	}

	if ip.IsLoopback() {
		return "localhost"
	}
	return addr
}

type notPermittedByPolicyErr struct {
	protocol string
}

var _ error = notPermittedByPolicyErr{}

// Error implements error.
func (e notPermittedByPolicyErr) Error() string {
	return fmt.Sprintf("connections are not permitted to %q by policy", e.protocol)
}
