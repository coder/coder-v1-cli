package activity

import "io"

type activityWriter struct {
	p  *Pusher
	wr io.Writer
}

// Write writes to the underlying writer and tracks activity
func (w *activityWriter) Write(p []byte) (n int, err error) {
	w.p.Push()
	return w.wr.Write(p)
}

// Writer wraps the given writer such that all writes trigger an activity push
func (p *Pusher) Writer(wr io.Writer) io.Writer {
	return &activityWriter{p: p, wr: wr}
}
