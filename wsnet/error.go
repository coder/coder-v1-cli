package wsnet

import (
	"fmt"
	"strings"

	"github.com/pion/webrtc/v3"
)

// errWrap wraps the error with some extra details about the state of the
// connection.
type errWrap struct {
	err error

	iceServers []webrtc.ICEServer
	rtc        webrtc.PeerConnectionState
}

var _ error = errWrap{}
var _ interface{ Unwrap() error } = errWrap{}

// Error implements error.
func (e errWrap) Error() string {
	return fmt.Sprintf("%v (ice: [%v], rtc: %v)", e.err.Error(), e.ice(), e.rtc.String())
}

func (e errWrap) ice() string {
	msgs := []string{}
	for _, s := range e.iceServers {
		msgs = append(msgs, strings.Join(s.URLs, ", "))
	}

	return strings.Join(msgs, ", ")
}

// Unwrap implements Unwrapper.
func (e errWrap) Unwrap() error {
	return e.err
}
