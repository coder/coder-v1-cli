// +build windows

package xterminal

import (
	"context"
	"time"

	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/windows"
)

// State differs per-platform.
type State struct {
	mode uint32
}

// makeRaw sets the terminal in raw mode and returns the previous state so it can be restored.
func makeRaw(handle windows.Handle, input bool) (uint32, error) {
	var prevState uint32
	if err := windows.GetConsoleMode(handle, &prevState); err != nil {
		return 0, err
	}

	var raw uint32
	if input {
		raw = prevState &^ (windows.ENABLE_ECHO_INPUT | windows.ENABLE_PROCESSED_INPUT | windows.ENABLE_LINE_INPUT | windows.ENABLE_PROCESSED_OUTPUT)
		raw |= windows.ENABLE_VIRTUAL_TERMINAL_INPUT
	} else {
		raw = prevState | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	}

	if err := windows.SetConsoleMode(handle, raw); err != nil {
		return 0, err
	}
	return prevState, nil
}

// MakeRaw sets an input terminal to raw and enables VT100 processing.
func MakeRaw(handle uintptr) (*State, error) {
	prevState, err := makeRaw(windows.Handle(handle), true)
	if err != nil {
		return nil, err
	}

	return &State{mode: prevState}, nil
}

// MakeOutputRaw sets an output terminal to raw and enables VT100 processing.
func MakeOutputRaw(handle uintptr) (*State, error) {
	prevState, err := makeRaw(windows.Handle(handle), false)
	if err != nil {
		return nil, err
	}

	return &State{mode: prevState}, nil
}

// Restore terminal back to original state.
func Restore(handle uintptr, state *State) error {
	return windows.SetConsoleMode(windows.Handle(handle), state.mode)
}

// ColorEnabled returns true if VT100 processing is enabled on the output
// console.
func ColorEnabled(handle uintptr) (bool, error) {
	var state uint32
	if err := windows.GetConsoleMode(windows.Handle(handle), &state); err != nil {
		return false, err
	}

	return state&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0, nil
}

// ResizeEvent represent the new size of the terminal.
type ResizeEvent struct {
	Height uint16
	Width  uint16
}

func (s ResizeEvent) equal(s2 *ResizeEvent) bool {
	if s2 == nil {
		return false
	}
	return s.Height == s2.Height && s.Width == s2.Width
}

// ResizeEvents sends terminal resize events when the dimensions change.
// Windows does not have a unix.SIGWINCH equivalent, so we poll the terminal size
// at a fixed interval
func ResizeEvents(ctx context.Context, termFD uintptr) chan ResizeEvent {
	// Use a buffered chan to avoid blocking if the main is not ready yet when we send the initial resize event.
	events := make(chan ResizeEvent, 1)

	go func() {
		defer close(events)

		// On windows, as we don't have a signal to know the size changed, we
		// use a ticker and emit then event if the current size differs from last time we checked.
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		var lastEvent *ResizeEvent
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				width, height, err := terminal.GetSize(int(windows.Handle(termFD)))
				if err != nil {
					return
				}
				event := ResizeEvent{
					Height: uint16(height),
					Width:  uint16(width),
				}
				if !event.equal(lastEvent) {
					select {
					case <-ctx.Done():
						return
					case events <- event:
					}
				}
				lastEvent = &event
			}
		}
	}()

	return events
}
