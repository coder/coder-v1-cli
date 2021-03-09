package coderutil

import (
	"context"

	"cdr.dev/coder-cli/coder-sdk"
)

// ProviderByName searches linearly for a workspace provider by its name.
func ProviderByName(ctx context.Context, client coder.Client, name string) (*coder.KubernetesProvider, error) {
	providers, err := client.WorkspaceProviders(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range providers.Kubernetes {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, coder.ErrNotFound
}
