// Package rfsnotify implements recursive folder monitoring by wrapping
// fsnotify.
package rfsnotify

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/xerrors"
)

// RWatcher wraps a fsnotify.Watcher, making it recursive.
type RWatcher struct {
	Events chan fsnotify.Event
	Errors chan error

	done     chan struct{}
	fsnotify *fsnotify.Watcher
	isClosed bool
}

// NewWatcher establishes a new watcher with the underlying OS and begins
// waiting for events.
func NewWatcher() (*RWatcher, error) {
	fsWatch, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, xerrors.Errorf("create underlying fsnotify watcher: %w", err)
	}

	m := &RWatcher{}
	m.fsnotify = fsWatch
	m.Events = make(chan fsnotify.Event)
	m.Errors = make(chan error)
	m.done = make(chan struct{})

	go m.start()

	return m, nil
}

// Add starts watching the named directory and all sub-directories.
func (m *RWatcher) Add(name string) error {
	if m.isClosed {
		return errors.New("rfsnotify instance already closed")
	}

	if err := m.watchRecursive(name, false); err != nil {
		return err
	}

	return nil
}

// Remove stops watching the named directory and all sub-directories.
func (m *RWatcher) Remove(name string) error {
	if err := m.watchRecursive(name, true); err != nil {
		return err
	}

	return nil
}

// Close removes all watches and closes the events channel.
func (m *RWatcher) Close() error {
	if m.isClosed {
		return nil
	}

	close(m.done)
	m.isClosed = true
	return nil
}

func (m *RWatcher) start() {
	for {
		select {
		case e := <-m.fsnotify.Events:
			s, err := os.Stat(e.Name)
			if err == nil && s != nil && s.IsDir() {
				if e.Op&fsnotify.Create != 0 {
					err := m.watchRecursive(e.Name, false)
					if err != nil {
						m.Errors <- xerrors.Errorf("watch newly created dir: %w", err)
					}
				}
			}

			// Since we can't stat a deleted path to see if it's a
			// directory, we try to remove either way.
			if e.Op&fsnotify.Remove == fsnotify.Remove {
				_ = m.fsnotify.Remove(e.Name)
			}
			m.Events <- e

		case e := <-m.fsnotify.Errors:
			m.Errors <- e

		case <-m.done:
			m.fsnotify.Close()
			close(m.Events)
			close(m.Errors)
			return
		}
	}
}

// watchRecursive adds or removes all sub-directories of a given path.
func (m *RWatcher) watchRecursive(path string, unwatch bool) error {
	err := filepath.Walk(path, func(walkPath string, fi os.FileInfo, err error) error {
		if err != nil {
			return xerrors.Errorf("walk path: %w", err)
		}

		if !fi.IsDir() {
			return nil
		}

		if unwatch {
			if err = m.fsnotify.Remove(walkPath); err != nil {
				return xerrors.Errorf("unwatch subdirectory: %w", err)
			}
		} else {
			if err = m.fsnotify.Add(walkPath); err != nil {
				return xerrors.Errorf("watch subdirectory: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return xerrors.Errorf("walk watch directory: %w", err)
	}

	return nil
}
