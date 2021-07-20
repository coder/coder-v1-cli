package wsnet

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// dialerFunc is used to reference a dialer returned for caching.
type dialerFunc func(ctx context.Context, key string, options *DialOptions) (*Dialer, error)

// DialCache constructs a new DialerCache.
// The cache clears connections that:
// 1. Are older than the TTL and have no active user-created connections.
// 2. Have been closed.
func DialCache(ttl time.Duration, dialer dialerFunc) *DialerCache {
	dc := &DialerCache{
		ttl:         ttl,
		dialerFunc:  dialer,
		closed:      make(chan struct{}),
		flightGroup: &singleflight.Group{},
		mut:         sync.RWMutex{},
		dialers:     make(map[string]*Dialer),
		atime:       make(map[string]time.Time),
	}
	go dc.init()
	return dc
}

type DialerCache struct {
	dialerFunc  dialerFunc
	ttl         time.Duration
	flightGroup *singleflight.Group

	closed  chan struct{}
	mut     sync.RWMutex
	dialers map[string]*Dialer
	atime   map[string]time.Time
}

// init starts the ticker for evicting connections.
func (d *DialerCache) init() {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	for {
		select {
		case <-d.closed:
			return
		case <-ticker.C:
			d.evict()
		}
	}
}

// evict removes lost/broken/expired connections from the cache.
func (d *DialerCache) evict() {
	var wg sync.WaitGroup
	d.mut.RLock()
	for key, dialer := range d.dialers {
		wg.Add(1)
		key := key
		dialer := dialer
		go func() {
			defer wg.Done()

			evict := false
			select {
			case <-dialer.Closed():
				evict = true
			default:
			}
			if dialer.ActiveConnections() == 0 && time.Since(d.atime[key]) >= d.ttl {
				evict = true
			}
			if !evict {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
				defer cancel()
				err := dialer.Ping(ctx)
				if err != nil {
					evict = true
				}
			}

			if evict {
				_ = dialer.Close()
				d.mut.Lock()
				delete(d.atime, key)
				delete(d.dialers, key)
				d.mut.Unlock()
			}
		}()
	}
	d.mut.RUnlock()
	wg.Wait()
}

// Dial returns a Dialer from the cache if one exists with the key provided,
// or dials a new connection using the dialerFunc.
func (d *DialerCache) Dial(ctx context.Context, key string, options *DialOptions) (*Dialer, bool, error) {
	d.mut.RLock()
	if dialer, ok := d.dialers[key]; ok {
		closed := false
		select {
		case <-dialer.Closed():
			closed = true
		default:
		}
		if !closed {
			d.mut.RUnlock()
			d.mut.Lock()
			d.atime[key] = time.Now()
			d.mut.Unlock()

			return dialer, true, nil
		}
	}
	d.mut.RUnlock()

	dialer, err, _ := d.flightGroup.Do(key, func() (interface{}, error) {
		dialer, err := d.dialerFunc(ctx, key, options)
		if err != nil {
			return nil, err
		}
		d.mut.Lock()
		d.dialers[key] = dialer
		d.atime[key] = time.Now()
		d.mut.Unlock()

		return dialer, nil
	})
	if err != nil {
		return nil, false, err
	}
	return dialer.(*Dialer), false, nil
}

// Close closes all cached dialers.
func (d *DialerCache) Close() error {
	d.mut.Lock()
	defer d.mut.Unlock()

	for key, dialer := range d.dialers {
		d.flightGroup.Forget(key)

		err := dialer.Close()
		if err != nil {
			return err
		}
	}
	close(d.closed)
	return nil
}
