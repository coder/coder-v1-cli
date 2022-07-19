package coderutil

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"golang.org/x/xerrors"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/clog"
)

// DialWorkspaceWsep dials the executor endpoint using the https://github.com/cdr/wsep message protocol.
// The proper workspace provider envproxy access URL is used.
func DialWorkspaceWsep(ctx context.Context, client coder.Client, workspace *coder.Workspace) (*websocket.Conn, error) {
	workspaceProvider, err := client.WorkspaceProviderByID(ctx, workspace.ResourcePoolID)
	if err != nil {
		return nil, xerrors.Errorf("get workspace workspace provider: %w", err)
	}
	accessURL, err := url.Parse(workspaceProvider.EnvproxyAccessURL)
	if err != nil {
		return nil, xerrors.Errorf("invalid workspace provider envproxy access url: %w", err)
	}

	conn, err := client.DialWsep(ctx, accessURL, workspace.ID)
	if err != nil {
		return nil, xerrors.Errorf("dial websocket: %w", err)
	}
	return conn, nil
}

// WorkspaceWithWorkspaceProvider composes an Workspace entity with its associated WorkspaceProvider.
type WorkspaceWithWorkspaceProvider struct {
	Workspace         coder.Workspace
	WorkspaceProvider coder.KubernetesProvider
}

// WorkspacesWithProvider performs the composition of each Workspace with its associated WorkspaceProvider.
func WorkspacesWithProvider(ctx context.Context, client coder.Client, workspaces []coder.Workspace) ([]WorkspaceWithWorkspaceProvider, error) {
	pooledWorkspaces := make([]WorkspaceWithWorkspaceProvider, 0, len(workspaces))
	providers, err := client.WorkspaceProviders(ctx)
	if err != nil {
		return nil, err
	}
	providerMap := make(map[string]coder.KubernetesProvider, len(providers.Kubernetes))
	for _, p := range providers.Kubernetes {
		providerMap[p.ID] = p
	}
	for _, e := range workspaces {
		workspaceProvider, ok := providerMap[e.ResourcePoolID]
		if !ok {
			return nil, xerrors.Errorf("fetch workspace workspace provider: %w", coder.ErrNotFound)
		}
		pooledWorkspaces = append(pooledWorkspaces, WorkspaceWithWorkspaceProvider{
			Workspace:         e,
			WorkspaceProvider: workspaceProvider,
		})
	}
	return pooledWorkspaces, nil
}

// DefaultWorkspaceProvider returns the default provider with which to create workspaces.
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

// WorkspaceTable defines an Workspace-like structure with associated entities composed in a human
// readable form.
type WorkspaceTable struct {
	Name     string  `table:"Name" json:"name"`
	Image    string  `table:"Image" json:"image"`
	CPU      float32 `table:"vCPU" json:"cpu"`
	MemoryGB float32 `table:"MemoryGB" json:"memory_gb"`
	DiskGB   int     `table:"DiskGB" json:"disk_gb"`
	Status   string  `table:"Status" json:"status"`
	Provider string  `table:"Provider" json:"provider"`
	CVM      bool    `table:"CVM" json:"cvm"`
	Username string  `table:"Username" json:"username"`
}

// WorkspacesHumanTable performs the composition of each Workspace with its associated ProviderName and ImageRepo.
func WorkspacesHumanTable(ctx context.Context, client coder.Client, workspaces []coder.Workspace) ([]WorkspaceTable, error) {
	imageMap, err := MakeImageMap(ctx, client, workspaces)
	if err != nil {
		return nil, err
	}

	userMap, err := MakeUserMap(ctx, client, workspaces)
	if err != nil {
		return nil, err
	}

	pooledWorkspaces := make([]WorkspaceTable, 0, len(workspaces))
	providers, err := client.WorkspaceProviders(ctx)
	if err != nil {
		return nil, err
	}
	providerMap := make(map[string]coder.KubernetesProvider, len(providers.Kubernetes))
	for _, p := range providers.Kubernetes {
		providerMap[p.ID] = p
	}
	for _, ws := range workspaces {
		workspaceProvider, ok := providerMap[ws.ResourcePoolID]
		if !ok {
			return nil, xerrors.Errorf("fetch workspace workspace provider: %w", coder.ErrNotFound)
		}
		pooledWorkspaces = append(pooledWorkspaces, WorkspaceTable{
			Name:     ws.Name,
			Image:    fmt.Sprintf("%s:%s", imageMap[ws.ImageID].Repository, ws.ImageTag),
			CPU:      ws.CPUCores,
			MemoryGB: ws.MemoryGB,
			DiskGB:   ws.DiskGB,
			Status:   string(ws.LatestStat.ContainerStatus),
			Provider: workspaceProvider.Name,
			CVM:      ws.UseContainerVM,
			Username: userMap[ws.UserID].Username,
		})
	}
	return pooledWorkspaces, nil
}

func MakeUserMap(ctx context.Context, client coder.Client, workspaces []coder.Workspace) (map[string]*coder.User, error) {
	var (
		mu     sync.Mutex
		egroup = clog.LoggedErrGroup()
	)

	userMap := map[string]*coder.User{}

	// Iterate over all the workspaces to get a list of unique User IDs.
	for _, ws := range workspaces {
		userMap[ws.UserID] = nil
	}

	fetchIds := make([]string, 0, len(userMap))
	for id := range userMap {
		fetchIds = append(fetchIds, id)
	}

	for _, id := range fetchIds {
		id := id
		egroup.Go(func() error {
			user, err := client.UserByID(ctx, id)
			if err != nil {
				return xerrors.Errorf("get user by id: %w", err)
			}
			mu.Lock()
			defer mu.Unlock()

			userMap[id] = user
			return nil
		})
	}

	if err := egroup.Wait(); err != nil {
		return nil, xerrors.Errorf("fetch all workspace users: %w", err)
	}

	return userMap, nil
}

// MakeImageMap fetches all image entities specified in the slice of workspaces, then places them into an ID map.
func MakeImageMap(ctx context.Context, client coder.Client, workspaces []coder.Workspace) (map[string]*coder.Image, error) {
	var (
		mu     sync.Mutex
		egroup = clog.LoggedErrGroup()
	)
	imageMap := make(map[string]*coder.Image)
	for _, e := range workspaces {
		// put all the image IDs into a map to remove duplicates
		imageMap[e.ImageID] = nil
	}
	ids := make([]string, 0, len(imageMap))
	for id := range imageMap {
		// put the deduplicated back into a slice
		// so we can write to the map while iterating
		ids = append(ids, id)
	}
	for _, id := range ids {
		id := id
		egroup.Go(func() error {
			img, err := client.ImageByID(ctx, id)
			if err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			imageMap[id] = img

			return nil
		})
	}
	if err := egroup.Wait(); err != nil {
		return nil, err
	}
	return imageMap, nil
}
