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

func makeRaw(handle windows.Handle, input bool) (uint32, error) {
	var st uint32
	if err := windows.GetConsoleMode(handle, &st); err != nil {
		return 0, err
	}

	var raw uint32
	if input {
		raw = st &^ (windows.ENABLE_ECHO_INPUT | windows.ENABLE_PROCESSED_INPUT | windows.ENABLE_LINE_INPUT | windows.ENABLE_PROCESSED_OUTPUT)
		raw |= windows.ENABLE_VIRTUAL_TERMINAL_INPUT
	} else {
		raw = st | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	}

	if err := windows.SetConsoleMode(handle, raw); err != nil {
		return 0, err
	}
	return st, nil
}

// MakeRaw sets an input terminal to raw and enables VT100 processing.
func MakeRaw(handle uintptr) (*State, error) {
	inSt, err := makeRaw(windows.Handle(handle), true)
	if err != nil {
		return nil, err
	}

	return &State{inSt}, nil
}

// MakeOutputRaw sets an output terminal to raw and enables VT100 processing.
func MakeOutputRaw(handle uintptr) (*State, error) {
	outSt, err := makeRaw(windows.Handle(handle), false)
	if err != nil {
		return nil, err
	}

	return &State{outSt}, nil
}

// Restore terminal back to original state.
func Restore(handle uintptr, state *State) error {
	return windows.SetConsoleMode(windows.Handle(handle), state.mode)
}

// ColorEnabled returns true if VT100 processing is enabled on the output
// console.
func ColorEnabled(handle uintptr) (bool, error) {
	var st uint32
	if err := windows.GetConsoleMode(windows.Handle(handle), &st); err != nil {
		return false, err
	}

	return st&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0, nil
}

type ResizeEvent struct {
	Height, Width uint16
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
func ResizeEvents(ctx context.Context, termfd uintptr) chan ResizeEvent {
	events := make(chan ResizeEvent)
	ticker := time.NewTicker(time.Millisecond * 100)

	go func() {
		defer ticker.Stop()
		var lastEvent *ResizeEvent

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				width, height, err := terminal.GetSize(int(windows.Handle(termfd)))
				if err != nil {
					return
				}
				event := ResizeEvent{
					Height: uint16(height),
					Width:  uint16(width),
				}
				if !event.equal(lastEvent) {
					events <- event
				}
				lastEvent = &event
			}
		}
	}()

	return events
}
