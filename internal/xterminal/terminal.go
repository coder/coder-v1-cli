// +build !windows

package xterminal

import (
	"context"
	"os"
	"os/signal"

	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
)

// State differs per-platform.
type State struct {
	s *terminal.State
}

// MakeRaw sets the terminal to raw.
func MakeRaw(fd uintptr) (*State, error) {
	s, err := terminal.MakeRaw(int(fd))
	return &State{s}, err
}

// MakeOutputRaw does nothing on non-Windows platforms.
func MakeOutputRaw(fd uintptr) (*State, error) {
	return nil, nil
}

// Restore terminal back to original state.
func Restore(fd uintptr, state *State) error {
	if state == nil {
		return nil
	}

	return terminal.Restore(int(fd), state.s)
}

// ColorEnabled returns true on Linux if handle is a terminal.
func ColorEnabled(fd uintptr) (bool, error) {
	return terminal.IsTerminal(int(fd)), nil
}

// ResizeEvent describes the new terminal dimensions following a resize
type ResizeEvent struct {
	Height, Width uint16
}

// ResizeEvents sends terminal resize events
func ResizeEvents(ctx context.Context, termfd uintptr) chan ResizeEvent {
	sigs := make(chan os.Signal, 16)
	signal.Notify(sigs, unix.SIGWINCH)

	events := make(chan ResizeEvent)

	go func() {
		for ctx.Err() == nil {
			width, height, err := terminal.GetSize(int(termfd))
			if err != nil {
				return
			}
			events <- ResizeEvent{
				Height: uint16(height),
				Width:  uint16(width),
			}

			<-sigs
		}
	}()

	return events
}
