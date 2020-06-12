package activity

import "io"

type activityWriter struct {
	p  *Pusher
	wr io.Writer
}

func (w *activityWriter) Write(p []byte) (n int, err error) {
	w.p.Push()
	return w.wr.Write(p)
}

func (p *Pusher) Writer(wr io.Writer) io.Writer {
	return &activityWriter{p: p, wr: wr}
}
