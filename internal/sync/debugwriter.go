package sync

import (
	"fmt"
	"io"
	"os"
)

type debugWriter struct {
	Prefix string
	W io.Writer
}

func (w debugWriter) Write(p []byte) (n int, err error) {
	if os.Getenv("DEBUG") == "" {
		return len(p), nil
	}
	_, err = fmt.Fprintf(w.W, "%v: %q\n", w.Prefix, p)
	return len(p), err
}

