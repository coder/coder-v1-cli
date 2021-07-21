package wsnet

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// DialCache constructs a new DialerCache.
// The cache clears connections that:
// 1. Are older than the TTL and have no active user-created connections.
// 2. Have been closed.
func DialCache(ttl time.Duration) *DialerCache {
	dc := &DialerCache{
		ttl:         ttl,
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
	ttl         time.Duration
	flightGroup *singleflight.Group
	closed      chan struct{}
	mut         sync.RWMutex

	// Key is the "key" of a dialer, which is usually the workspace ID.
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
			if dialer.activeConnections() == 0 && time.Since(d.atime[key]) >= d.ttl {
				evict = true
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
				defer cancel()
				err := dialer.Ping(ctx)
				if err != nil {
					evict = true
				}
			}

			if !evict {
				return
			}

			_ = dialer.Close()
			// Ensure after Ping and potential delays that we're still testing against
			// the proper dialer.
			if dialer != d.dialers[key] {
				return
			}

			d.mut.Lock()
			defer d.mut.Unlock()
			delete(d.atime, key)
			delete(d.dialers, key)
		}()
	}
	d.mut.RUnlock()
	wg.Wait()
}

// Dial returns a Dialer from the cache if one exists with the key provided,
// or dials a new connection using the dialerFunc.
// The bool returns whether the connection was found in the cache or not.
func (d *DialerCache) Dial(ctx context.Context, key string, dialerFunc func() (*Dialer, error)) (*Dialer, bool, error) {
	select {
	case <-d.closed:
		return nil, false, errors.New("cache closed")
	default:
	}

	d.mut.RLock()
	dialer, ok := d.dialers[key]
	d.mut.RUnlock()
	if ok {
		d.mut.Lock()
		d.atime[key] = time.Now()
		d.mut.Unlock()

		return dialer, true, nil
	}

	rawDialer, err, _ := d.flightGroup.Do(key, func() (interface{}, error) {
		dialer, err := dialerFunc()
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
	select {
	case <-d.closed:
		return nil, false, errors.New("cache closed")
	default:
	}

	return rawDialer.(*Dialer), false, nil
}

// Close closes all cached dialers.
func (d *DialerCache) Close() error {
	d.mut.Lock()
	defer d.mut.Unlock()

	for _, dialer := range d.dialers {
		err := dialer.Close()
		if err != nil {
			return err
		}
	}
	close(d.closed)
	return nil
}
