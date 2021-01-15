package coderutil

import (
	"context"
	"net/url"

	"cdr.dev/coder-cli/coder-sdk"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
)

// DialEnvWsep dials the executor endpoint using the https://github.com/cdr/wsep message protocol.
// The proper resource pool access URL is used.
func DialEnvWsep(ctx context.Context, client *coder.Client, env *coder.Environment) (*websocket.Conn, error) {
	resourcePool, err := client.ResourcePoolByID(ctx, env.ResourcePoolID)
	if err != nil {
		return nil, xerrors.Errorf("get env resource pool: %w", err)
	}
	accessURL, err := url.Parse(resourcePool.AccessURL)
	if err != nil {
		return nil, xerrors.Errorf("invalid resource pool access url: %w", err)
	}

	conn, err := client.DialWsep(ctx, accessURL, env.ID)
	if err != nil {
		return nil, xerrors.Errorf("dial websocket: %w", err)
	}
	return conn, nil
}

// EnvWithPool composes an Environment entity with its associated ResourcePool.
type EnvWithPool struct {
	Env  coder.Environment
	Pool coder.ResourcePool
}

// EnvsWithPool performs the composition of each Environment with its associated ResourcePool.
func EnvsWithPool(ctx context.Context, client *coder.Client, envs []coder.Environment) ([]EnvWithPool, error) {
	pooledEnvs := make([]EnvWithPool, len(envs))
	pools, err := client.ResourcePools(ctx)
	if err != nil {
		return nil, err
	}
	poolMap := make(map[string]coder.ResourcePool, len(pools))
	for _, p := range pools {
		poolMap[p.ID] = p
	}
	for _, e := range envs {
		envPool, ok := poolMap[e.ResourcePoolID]
		if !ok {
			return nil, xerrors.Errorf("fetch env resource pool: %w", coder.ErrNotFound)
		}
		pooledEnvs = append(pooledEnvs, EnvWithPool{
			Env:  e,
			Pool: envPool,
		})
	}
	return pooledEnvs, nil
}
