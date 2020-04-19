package wush

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Client converts a Wush connection into streams.
type Client struct {
	done     <-chan struct{}
	statusMu sync.Mutex
	exitCode uint8
	err      error

	Stdin  io.WriteCloser
	Stdout io.Reader
	Stderr io.Reader
}

type stdinWriter struct {
	conn *websocket.Conn
	*Client
	ctx context.Context
}

func (w *stdinWriter) writeChunk(p []byte) (int, error) {
	err := wsjson.Write(w.ctx, w.conn, &ClientMessage{
		Type:  Stdin,
		Input: base64.StdEncoding.EncodeToString(p),
	})
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (w *stdinWriter) Write(p []byte) (int, error) {
	// The real size is ~64,000, but we have to account for the
	// shitty base64ing.
	const maxSize = 8192

	var nn int
	for len(p) > maxSize {
		n, err := w.writeChunk(p[:maxSize])
		nn += n
		if err != nil {
			return nn, err
		}
		p = p[maxSize:]
	}

	n, err := w.writeChunk(p)
	nn += n
	return nn, err
}

func (w *stdinWriter) Close() error {
	return wsjson.Write(w.ctx, w.conn, &ClientMessage{
		Type: CloseStdin,
	})
}

// Dial begins multiplexing the Wush connection
// into independent streams.
// It will cancel all goroutines when the provided context cancels.
func Dial(ctx context.Context, conn *websocket.Conn) *Client {
	var (
		stdoutReader, stdoutWriter = io.Pipe()
		stderrReader, stderrWriter = io.Pipe()
	)
	done := make(chan struct{})
	c := &Client{
		Stdout: stdoutReader,
		Stderr: stderrReader,
		done:   done,
	}
	eg, ctx := errgroup.WithContext(ctx)
	c.Stdin = &stdinWriter{
		Client: c,
		conn:   conn,
		ctx:    ctx,
	}

	// We expect massive reads from some commands. Because we're streaming it's no big deal.
	conn.SetReadLimit(1 << 40)

	// This channel must be buffered because all goroutines exit before the cleanup routine.
	exitCode := make(chan uint8, 1)
	// Start read side
	eg.Go(func() error {
		defer stdoutWriter.Close()
		defer stderrWriter.Close()

		buf := make([]byte, 32<<10)
		for {
			_, rdr, err := conn.Reader(ctx)
			if err != nil {
				return nil
			}
			streamID := make([]byte, 1)
			_, err = io.ReadFull(rdr, streamID)
			if err != nil {
				return xerrors.Errorf("read stream ID: %w", err)
			}
			switch StreamID(streamID[0]) {
			case Stdout:
				_, err = io.CopyBuffer(stdoutWriter, rdr, buf)
				if err != nil {
					return err
				}
			case Stderr:
				_, err = io.CopyBuffer(stderrWriter, rdr, buf)
				if err != nil {
					return err
				}
			case ExitCode:
				exitCodeBuf := make([]byte, 1)
				_, err = io.ReadFull(rdr, exitCodeBuf)
				if err != nil {
					return xerrors.Errorf("read exit code: %w", err)
				}
				exitCode <- uint8(exitCodeBuf[0])
				return nil
			default:
				return fmt.Errorf("unexpected id %x", streamID[0])
			}
		}
	})
	// Cleanup routine
	go func() {
		err := eg.Wait()
		c.statusMu.Lock()
		defer c.statusMu.Unlock()
		// If the command failed before exit code, don't block.
		select {
		case c.exitCode = <-exitCode:
		default:
		}
		c.err = err
		close(done)
	}()

	return c
}

// Wait returns the status code of the command, along
// with any error.
func (c *Client) Wait() (uint8, error) {
	<-c.done
	// There is guaranteed to be no writers after the channel is closed.
	return c.exitCode, c.err
}
