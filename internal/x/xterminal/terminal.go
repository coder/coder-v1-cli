// +build !windows

package xterminal

import (
	"golang.org/x/term"
)

// State differs per-platform.
type State struct {
	s *term.State
}

// MakeOutputRaw does nothing on non-Windows platforms.
func MakeOutputRaw(fd uintptr) (*State, error) { return nil, nil }

// Restore terminal back to original state.
func Restore(fd uintptr, state *State) error {
	if state == nil {
		return nil
	}

	return term.Restore(int(fd), state.s)
}
