package sync

import (
	"time"

	"github.com/rjeczalik/notify"
)

type timedEvent struct {
	CreatedAt time.Time
	notify.EventInfo
}
