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
	previousState, err := terminal.MakeRaw(int(fd))
	if err != nil {
		return nil, err
	}
	return &State{s: previousState}, nil
}

// MakeOutputRaw does nothing on non-Windows platforms.
func MakeOutputRaw(fd uintptr) (*State, error) { return nil, nil }

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
	Height uint16
	Width  uint16
}

// ResizeEvents sends terminal resize events.
func ResizeEvents(ctx context.Context, termFD uintptr) chan ResizeEvent {
	// Use a buffered chan to avoid blocking when we emit the initial resize event.
	// We send the event right away while the main routine might not be ready just yet.
	events := make(chan ResizeEvent, 1)

	go func() {
		sigChan := make(chan os.Signal, 16) // Arbitrary large buffer size to allow for "continuous" resizing without blocking.
		defer close(sigChan)

		// Terminal resize event are notified using the SIGWINCH signal, start watching for it.
		signal.Notify(sigChan, unix.SIGWINCH)
		defer signal.Stop(sigChan)

		// Emit an initial signal event to make sure the server receives our current window size.
		select {
		case <-ctx.Done():
			return
		case sigChan <- unix.SIGWINCH:
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-sigChan:
				width, height, err := terminal.GetSize(int(termFD))
				if err != nil {
					return
				}
				event := ResizeEvent{
					Height: uint16(height),
					Width:  uint16(width),
				}
				select {
				case <-ctx.Done():
					return
				case events <- event:
				}
			}
		}
	}()

	return events
}
