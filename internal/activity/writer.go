package activity

import (
	"context"
	"io"
)

// writer wraps a standard io.Writer with the activity pusher.
type writer struct {
	p  *Pusher
	wr io.Writer
}

// Write writes to the underlying writer and tracks activity.
func (w *writer) Write(buf []byte) (int, error) {
	w.p.Push(context.Background())
	return w.wr.Write(buf)
}

// Writer wraps the given writer such that all writes trigger an activity push
func (p *Pusher) Writer(wr io.Writer) io.Writer {
	return &writer{p: p, wr: wr}
}
