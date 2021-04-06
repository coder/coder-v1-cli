package proto

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/pion/webrtc/v3"
)

// Message is a common format for agent and client to use in handshake.
type Message struct {
	Error     string                     `json:"error"`
	Candidate string                     `json:"candidate"`
	Offer     *webrtc.SessionDescription `json:"offer"`
	Answer    *webrtc.SessionDescription `json:"answer"`
}

// WriteError responds with an error and status code.
func WriteError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(err.Error()))
}

// ProxyICECandidates sends all ICE candidates using the message protocol
// to the writer provided.
func ProxyICECandidates(conn *webrtc.PeerConnection, w io.Writer) func() {
	queue := make([]*webrtc.ICECandidate, 0)
	flushed := false
	write := func(i *webrtc.ICECandidate) {
		b, _ := json.Marshal(&Message{
			Candidate: i.ToJSON().Candidate,
		})
		_, _ = w.Write(b)
	}

	conn.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}
		if !flushed {
			queue = append(queue, i)
			return
		}

		write(i)
	})
	return func() {
		for _, i := range queue {
			write(i)
		}
		flushed = true
	}
}

// AcceptICECandidate adds the candidate to the connection.
func AcceptICECandidate(conn *webrtc.PeerConnection, m *Message) error {
	return conn.AddICECandidate(webrtc.ICECandidateInit{
		Candidate: m.Candidate,
	})
}
