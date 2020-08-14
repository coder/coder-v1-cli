package activity

import (
	"context"
	"time"

	"cdr.dev/coder-cli/coder-sdk"
	"golang.org/x/time/rate"

	"go.coder.com/flog"
)

const pushInterval = time.Minute

// Pusher pushes activity metrics no more than once per pushInterval. Pushes
// within the same interval are a no-op.
type Pusher struct {
	envID  string
	source string

	client *coder.Client
	rate   *rate.Limiter
}

// NewPusher instantiates a new instance of Pusher
func NewPusher(c *coder.Client, envID, source string) *Pusher {
	return &Pusher{
		envID:  envID,
		source: source,
		client: c,
		rate:   rate.NewLimiter(rate.Every(pushInterval), 1),
	}
}

// Push pushes activity, abiding by a rate limit
func (p *Pusher) Push(ctx context.Context) {
	if !p.rate.Allow() {
		return
	}

	err := p.client.PushActivity(ctx, p.source, p.envID)
	if err != nil {
		flog.Error("push activity: %s", err.Error())
	}
}
