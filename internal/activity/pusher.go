package activity

import (
	"context"
	"time"

	"golang.org/x/time/rate"

	"cdr.dev/coder-cli/coder-sdk"

	"go.coder.com/flog"
)

const pushInterval = time.Minute

// Pusher pushes activity metrics no more than once per pushInterval. Pushes
// within the same interval are a no-op.
type Pusher struct {
	envID  string
	source string

	client *coder.Client
	rate   *rate.Limiter // Use a rate limiter to control the sampling rate.
}

// NewPusher instantiates a new instance of Pusher.
func NewPusher(c *coder.Client, envID, source string) *Pusher {
	return &Pusher{
		envID:  envID,
		source: source,
		client: c,
		// Sample only 1 per interval to avoid spamming the api.
		rate: rate.NewLimiter(rate.Every(pushInterval), 1),
	}
}

// Push pushes activity, abiding by a rate limit.
func (p *Pusher) Push(ctx context.Context) {
	// If we already sampled data within the allowable range, do nothing.
	if !p.rate.Allow() {
		return
	}

	if err := p.client.PushActivity(ctx, p.source, p.envID); err != nil {
		flog.Error("push activity: %s", err)
	}
}
