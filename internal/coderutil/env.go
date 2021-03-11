package coderutil

import (
	"context"
	"fmt"
	"net/url"

	"golang.org/x/xerrors"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/coder-sdk"
)

// DialEnvWsep dials the executor endpoint using the https://github.com/cdr/wsep message protocol.
// The proper workspace provider envproxy access URL is used.
func DialEnvWsep(ctx context.Context, client coder.Client, env *coder.Environment) (*websocket.Conn, error) {
	workspaceProvider, err := client.WorkspaceProviderByID(ctx, env.ResourcePoolID)
	if err != nil {
		return nil, xerrors.Errorf("get env workspace provider: %w", err)
	}
	accessURL, err := url.Parse(workspaceProvider.EnvproxyAccessURL)
	if err != nil {
		return nil, xerrors.Errorf("invalid workspace provider envproxy access url: %w", err)
	}

	conn, err := client.DialWsep(ctx, accessURL, env.ID)
	if err != nil {
		return nil, xerrors.Errorf("dial websocket: %w", err)
	}
	return conn, nil
}

// EnvWithWorkspaceProvider composes an Environment entity with its associated WorkspaceProvider.
type EnvWithWorkspaceProvider struct {
	Env               coder.Environment
	WorkspaceProvider coder.KubernetesProvider
}

// EnvsWithProvider performs the composition of each Environment with its associated WorkspaceProvider.
func EnvsWithProvider(ctx context.Context, client coder.Client, envs []coder.Environment) ([]EnvWithWorkspaceProvider, error) {
	pooledEnvs := make([]EnvWithWorkspaceProvider, 0, len(envs))
	providers, err := client.WorkspaceProviders(ctx)
	if err != nil {
		return nil, err
	}
	providerMap := make(map[string]coder.KubernetesProvider, len(providers.Kubernetes))
	for _, p := range providers.Kubernetes {
		providerMap[p.ID] = p
	}
	for _, e := range envs {
		envProvider, ok := providerMap[e.ResourcePoolID]
		if !ok {
			return nil, xerrors.Errorf("fetch env workspace provider: %w", coder.ErrNotFound)
		}
		pooledEnvs = append(pooledEnvs, EnvWithWorkspaceProvider{
			Env:               e,
			WorkspaceProvider: envProvider,
		})
	}
	return pooledEnvs, nil
}

// DefaultWorkspaceProvider returns the default provider with which to create environments.
func DefaultWorkspaceProvider(ctx context.Context, c coder.Client) (*coder.KubernetesProvider, error) {
	provider, err := c.WorkspaceProviders(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range provider.Kubernetes {
		if p.BuiltIn {
			return &p, nil
		}
	}
	return nil, coder.ErrNotFound
}

type EnvTable struct {
	Name     string  `table:"Name"`
	Image    string  `table:"Image"`
	CPU      float32 `table:"vCPU"`
	MemoryGB float32 `table:"MemoryGB"`
	DiskGB   int     `table:"DiskGB"`
	Status   string  `table:"Status"`
	Provider string  `table:"Provider"`
	CVM      bool    `table:"CVM"`
}

// EnvsHumanTable performs the composition of each Environment with its associated ProviderName and ImageRepo.
func EnvsHumanTable(ctx context.Context, client coder.Client, envs []coder.Environment) ([]EnvTable, error) {
	imageMap := make(map[string]*coder.Image)
	for _, e := range envs {
		imageMap[e.ImageID] = nil
	}
	// TODO: make this concurrent
	for id := range imageMap {
		img, err := client.ImageByID(ctx, id)
		if err != nil {
			return nil, err
		}
		imageMap[id] = img
	}

	pooledEnvs := make([]EnvTable, 0, len(envs))
	providers, err := client.WorkspaceProviders(ctx)
	if err != nil {
		return nil, err
	}
	providerMap := make(map[string]coder.KubernetesProvider, len(providers.Kubernetes))
	for _, p := range providers.Kubernetes {
		providerMap[p.ID] = p
	}
	for _, e := range envs {
		envProvider, ok := providerMap[e.ResourcePoolID]
		if !ok {
			return nil, xerrors.Errorf("fetch env workspace provider: %w", coder.ErrNotFound)
		}
		pooledEnvs = append(pooledEnvs, EnvTable{
			Name:     e.Name,
			Image:    fmt.Sprintf("%s:%s", imageMap[e.ImageID].Repository, e.ImageTag),
			CPU:      e.CPUCores,
			MemoryGB: e.MemoryGB,
			DiskGB:   e.DiskGB,
			Status:   string(e.LatestStat.ContainerStatus),
			Provider: envProvider.Name,
			CVM:      e.UseContainerVM,
		})
	}
	return pooledEnvs, nil
}
