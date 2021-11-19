package wsnet

import (
	"fmt"
	"strings"

	"github.com/pion/webrtc/v3"
)

// wrapError wraps the error with some extra details about the state of the
// connection.
type wrapError struct {
	err error

	iceServers []webrtc.ICEServer
	rtc        webrtc.PeerConnectionState
}

var _ error = wrapError{}
var _ interface{ Unwrap() error } = wrapError{}

// Error implements error.
func (e wrapError) Error() string {
	return fmt.Sprintf("%v (ice: [%v], rtc: %v)", e.err.Error(), e.ice(), e.rtc.String())
}

func (e wrapError) ice() string {
	msgs := []string{}
	for _, s := range e.iceServers {
		msgs = append(msgs, strings.Join(s.URLs, ", "))
	}

	return strings.Join(msgs, ", ")
}

// Unwrap implements Unwrapper.
func (e wrapError) Unwrap() error {
	return e.err
}
