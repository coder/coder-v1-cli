// +build !windows

package xterminal

import (
	"golang.org/x/crypto/ssh/terminal"
)

// State differs per-platform.
type State struct {
	s *terminal.State
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
