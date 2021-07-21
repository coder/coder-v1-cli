package wsnet

import (
	"context"
	"testing"
	"time"

	"cdr.dev/slog/sloggers/slogtest"
	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	dialFunc := func(connectAddr string) func() (*Dialer, error) {
		return func() (*Dialer, error) {
			return DialWebsocket(context.Background(), connectAddr, nil, nil)
		}
	}

	t.Run("Caches", func(t *testing.T) {
		connectAddr, listenAddr := createDumbBroker(t)
		l, err := Listen(context.Background(), slogtest.Make(t, nil), listenAddr, "")
		require.NoError(t, err)
		defer l.Close()

		cache := DialCache(time.Hour)
		_, cached, err := cache.Dial(context.Background(), "example", dialFunc(connectAddr))
		require.NoError(t, err)
		require.Equal(t, cached, false)
		_, cached, err = cache.Dial(context.Background(), "example", dialFunc(connectAddr))
		require.NoError(t, err)
		require.Equal(t, cached, true)
	})

	t.Run("Create If Closed", func(t *testing.T) {
		connectAddr, listenAddr := createDumbBroker(t)
		l, err := Listen(context.Background(), slogtest.Make(t, nil), listenAddr, "")
		require.NoError(t, err)
		defer l.Close()

		cache := DialCache(time.Hour)

		conn, cached, err := cache.Dial(context.Background(), "example", dialFunc(connectAddr))
		require.NoError(t, err)
		require.Equal(t, cached, false)
		require.NoError(t, conn.Close())
		_, cached, err = cache.Dial(context.Background(), "example", dialFunc(connectAddr))
		require.NoError(t, err)
		require.Equal(t, cached, false)
	})

	t.Run("Evict No Connections", func(t *testing.T) {
		connectAddr, listenAddr := createDumbBroker(t)
		l, err := Listen(context.Background(), slogtest.Make(t, nil), listenAddr, "")
		require.NoError(t, err)
		defer l.Close()

		cache := DialCache(0)

		_, cached, err := cache.Dial(context.Background(), "example", dialFunc(connectAddr))
		require.NoError(t, err)
		require.Equal(t, cached, false)
		cache.evict()
		_, cached, err = cache.Dial(context.Background(), "example", dialFunc(connectAddr))
		require.NoError(t, err)
		require.Equal(t, cached, false)
	})
}
