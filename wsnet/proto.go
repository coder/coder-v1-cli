package wsnet

import (
	"github.com/pion/webrtc/v3"
)

// DialPolicy a single network + address + port combinations that a connection
// is permitted to use.
type DialPolicy struct {
	// If network is empty, it applies to all networks.
	Network string `json:"network"`
	// If host is empty, it applies to all hosts. "localhost" and any IP address
	// under "127.0.0.0/8" can be used interchangeably.
	Host string `json:"address"`
	// If port is 0, it applies to all ports.
	Port uint16 `json:"port"`
}

// ProtoMessage is used for brokering a dialer and listener.
//
// Dialers initiate an exchange by providing an Offer,
// along with a list of ICE servers for the listener to
// peer with.
//
// The listener should respond with an offer, then both
// sides can begin exchanging candidates.
type ProtoMessage struct {
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

// dialChannelMessage is used to notify a dial channel of a
// listening state. Modeled after net.OpError, and marshalled
// to that if Net is not "".
type dialChannelMessage struct {
	Err string
	Net string
	Op  string
}
