package sync

import (
	"os"
	"time"

	"github.com/rjeczalik/notify"
)

type timedEvent struct {
	CreatedAt time.Time
	notify.EventInfo
}

type eventCache map[string]timedEvent

func (cache eventCache) Add(ev timedEvent) {
	lastEvent, ok := cache[ev.Path()]
	if ok {
		switch {
		// If the file was quickly created and then destroyed, pretend nothing ever happened.
		case lastEvent.Event() == notify.Create && ev.Event() == notify.Remove:
			delete(cache, ev.Path())
			return
		}
	}
	// Only let the latest event for a path have action.
	cache[ev.Path()] = ev
}

// SequentialEvents returns the list of events that pertain to directories.
// The set of returned events is disjoint with ConcurrentEvents.
func (cache eventCache) SequentialEvents() []timedEvent {
	var r []timedEvent
	for _, ev := range cache {
		info, err := os.Stat(ev.Path())
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
		info, err := os.Stat(ev.Path())
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
