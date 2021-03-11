package xsync

import (
	"io"
	"sync"
)

// Writer synchronizes concurrent writes to an underlying writer.
func Writer(w io.Writer) io.Writer {
	return &writer{
		w: w,
	}
}

type writer struct {
	mu sync.Mutex
	w  io.Writer
}

func (sw *writer) Write(b []byte) (int, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(b)
}
