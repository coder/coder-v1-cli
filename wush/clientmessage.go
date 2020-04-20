package wush

// ClientMsgType specifies the type of a wush lite message from the client.
type ClientMsgType int

const (
	Stdin ClientMsgType = iota
	Resize
	CloseStdin
)

// ClientMessage is sent from the local client to the wush server.
type ClientMessage struct {
	Type ClientMsgType `json:"type"`

	// Valid on "input" messages.
	// Because this encodes everything in base64, it will case ~30% performance
	// overhead.
	Input string `json:"input"`

	// Valid on "resize" messages.
	Height int `json:"height"`
	Width  int `json:"width"`
}
