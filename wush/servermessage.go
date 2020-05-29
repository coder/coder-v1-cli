package wush

import "io"

// StreamID specifies the kind of data being transmitted.
type StreamID byte

// Wush lite output stream IDs.
const (
	Stdout StreamID = 0
	Stderr StreamID = 1

	// StreamID is a special stream ID, where the data is just a
	// single byte representing the exit code of the process.
	ExitCode StreamID = 255
)

// ServerMessage is sent over websocket type binary.
type ServerMessage struct {
	StreamID byte
	Body     io.Reader
}
