package activity

import (
	"time"

	"cdr.dev/coder-cli/internal/entclient"
	"go.coder.com/flog"
	"golang.org/x/time/rate"
)

const pushInterval = time.Minute

// Pusher pushes activity metrics no more than once per pushInterval. Pushes
// within the same interval are a no-op.
type Pusher struct {
	envID  string
	source string

	client *entclient.Client
	rate   *rate.Limiter
}

func NewPusher(c *entclient.Client, envID, source string) *Pusher {
	return &Pusher{
		envID:  envID,
		source: source,
		client: c,
		rate:   rate.NewLimiter(rate.Every(pushInterval), 1),
	}
}

func (p *Pusher) Push() {
	if !p.rate.Allow() {
		return
	}

	err := p.client.PushActivity(p.source, p.envID)
	if err != nil {
		flog.Error("push activity: %s", err.Error())
	}
}
