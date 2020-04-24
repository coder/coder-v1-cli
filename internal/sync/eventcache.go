package sync

import (
	"os"
	"time"

	"github.com/rjeczalik/notify"
	"go.coder.com/flog"
)

type timedEvent struct {
	CreatedAt time.Time
	notify.EventInfo
}

type eventCache map[string]timedEvent

func (cache eventCache) Add(ev timedEvent) {
	log := flog.New()
	log.Prefix = ev.Path() + ": "
	lastEvent, ok := cache[ev.Path()]
	if ok {
		switch {
		// If the file was quickly created and then destroyed, pretend nothing ever happened.
		case lastEvent.Event() == notify.Create && ev.Event() == notify.Remove:
			delete(cache, ev.Path())
			log.Info("ignored Create then Remove")
			return
		}
	}
	if ok {
		log.Info("ignored duplicate event (%s replaced by %s)", lastEvent.Event(), ev.Event())
	}
	// Only let the latest event for a path have action.
	cache[ev.Path()] = ev
}

// DirectoryEvents returns the list of events that pertain to directories.
// The set of returns events is disjoint with FileEvents.
func (cache eventCache) DirectoryEvents() []timedEvent {
	var r []timedEvent
	for _, ev := range cache {
		info, err := os.Stat(ev.Path())
		if err != nil {
			continue
		}
		if !info.IsDir() {
			continue
		}
		r = append(r, ev)

	}
	return r
}

// FileEvents returns the list of events that pertain to files.
// The set of returns events is disjoint with DirectoryEvents.
func (cache eventCache) FileEvents() []timedEvent {
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
