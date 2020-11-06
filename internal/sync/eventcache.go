package sync

import (
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

type timedEvent struct {
	CreatedAt time.Time
	fsnotify.Event
}

type eventCache map[string]timedEvent

func (cache eventCache) Add(ev timedEvent) {
	lastEvent, ok := cache[ev.Name]
	if ok {
		// If the file was quickly created and then destroyed, pretend nothing ever happened.
		if lastEvent.Op&fsnotify.Create == fsnotify.Create && ev.Op&fsnotify.Remove == fsnotify.Remove {
			delete(cache, ev.Name)
			return
		}
	}
	// Only let the latest event for a path have action.
	cache[ev.Name] = ev
}

// SequentialEvents returns the list of events that pertain to directories.
// The set of returned events is disjoint with ConcurrentEvents.
func (cache eventCache) SequentialEvents() []timedEvent {
	var r []timedEvent
	for _, ev := range cache {
		info, err := os.Stat(ev.Name)
		if err == nil && !info.IsDir() {
			continue
		}
		// Include files that have deleted here.
		// It's unclear whether they're files or folders.
		r = append(r, ev)
	}
	return r
}

// ConcurrentEvents returns the list of events that are safe to process after SequentialEvents.
// The set of returns events is disjoint with SequentialEvents.
func (cache eventCache) ConcurrentEvents() []timedEvent {
	var r []timedEvent
	for _, ev := range cache {
		info, err := os.Stat(ev.Name)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		r = append(r, ev)
	}
	return r
}
