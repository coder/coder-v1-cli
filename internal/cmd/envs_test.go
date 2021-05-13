package cmd

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"testing"
	"time"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"cdr.dev/slog/sloggers/slogtest/assert"
	"github.com/google/go-cmp/cmp"

	"cdr.dev/coder-cli/coder-sdk"
)

func Test_workspaces_ls(t *testing.T) {
	skipIfNoAuth(t)
	res := execute(t, nil, "ws", "ls")
	res.success(t)

	res = execute(t, nil, "ws", "ls", "--output=json")
	res.success(t)

	var workspaces []coder.Workspace
	res.stdoutUnmarshals(t, &workspaces)
}

func Test_workspaces_ls_by_provider(t *testing.T) {
	skipIfNoAuth(t)
	for _, test := range []struct {
		name    string
		command []string
		assert  func(r result)
	}{
		{
			name:    "simple list",
			command: []string{"ws", "ls", "--provider", "built-in"},
			assert:  func(r result) { r.success(t) },
		},
		{
			name:    "list as json",
			command: []string{"ws", "ls", "--provider", "built-in", "--output", "json"},
			assert: func(r result) {
				var workspaces []coder.Workspace
				r.stdoutUnmarshals(t, &workspaces)
			},
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			test.assert(execute(t, nil, test.command...))
		})
	}
}

func Test_workspace_create(t *testing.T) {
	skipIfNoAuth(t)
	ctx := context.Background()

	// Minimum args not received.
	res := execute(t, nil, "workspaces", "create")
	res.error(t)
	res.stderrContains(t, "accepts 1 arg(s), received 0")

	// Successfully output help.
	res = execute(t, nil, "workspaces", "create", "--help")
	res.success(t)
	res.stdoutContains(t, "Create a new Coder workspace.")

	// Image unset
	res = execute(t, nil, "workspaces", "create", "test-workspace")
	res.error(t)
	res.stderrContains(t, "fatal: required flag(s) \"image\" not set")

	// Image not imported
	res = execute(t, nil, "workspaces", "create", "test-workspace", "--image=doestexist")
	res.error(t)
	res.stderrContains(t, "fatal: image not found - did you forget to import this image?")

	ensureImageImported(ctx, t, testCoderClient, "ubuntu")

	name := randString(10)
	cpu := 2.3

	// attempt to remove the workspace on cleanup
	t.Cleanup(func() { _ = execute(t, nil, "ws", "rm", name, "--force") })

	res = execute(t, nil, "ws", "create", name, "--image=ubuntu", fmt.Sprintf("--cpu=%f", cpu))
	res.success(t)

	res = execute(t, nil, "ws", "ls")
	res.success(t)
	res.stdoutContains(t, name)

	var workspaces []coder.Workspace
	res = execute(t, nil, "ws", "ls", "--output=json")
	res.success(t)
	res.stdoutUnmarshals(t, &workspaces)
	workspace := assertWorkspace(t, name, workspaces)
	assert.Equal(t, "workspace cpu", cpu, float64(workspace.CPUCores), floatComparer)

	res = execute(t, nil, "ws", "watch-build", name)
	res.success(t)

	// edit the CPU of the workspace
	cpu = 2.1
	res = execute(t, nil, "ws", "edit", name, fmt.Sprintf("--cpu=%f", cpu), "--follow", "--force")
	res.success(t)

	// assert that the CPU actually did change after edit
	res = execute(t, nil, "ws", "ls", "--output=json")
	res.success(t)
	res.stdoutUnmarshals(t, &workspaces)
	workspace = assertWorkspace(t, name, workspaces)
	assert.Equal(t, "workspace cpu", cpu, float64(workspace.CPUCores), floatComparer)

	res = execute(t, nil, "ws", "rm", name, "--force")
	res.success(t)
}

func assertWorkspace(t *testing.T, name string, workspaces []coder.Workspace) *coder.Workspace {
	for _, e := range workspaces {
		if name == e.Name {
			return &e
		}
	}
	slogtest.Fatal(t, "workspace not found", slog.F("name", name), slog.F("workspaces", workspaces))
	return nil
}

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

//nolint:unparam
func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

var floatComparer = cmp.Comparer(func(x, y float64) bool {
	delta := math.Abs(x - y)
	mean := math.Abs(x+y) / 2.0
	return delta/mean < 0.001
})

// this is a stopgap until we have support for a `coder images` subcommand
// until then, we can use the coder.Client to ensure our integration tests
// work on fresh deployments.
func ensureImageImported(ctx context.Context, t *testing.T, client coder.Client, img string) {
	orgs, err := client.Organizations(ctx)
	assert.Success(t, "get orgs", err)

	var org *coder.Organization
search:
	for _, o := range orgs {
		for _, m := range o.Members {
			if m.Email == os.Getenv("CODER_EMAIL") {
				o := o
				org = &o
				break search
			}
		}
	}
	if org == nil {
		slogtest.Fatal(t, "failed to find org of current user")
		return // help the linter out a bit
	}

	registries, err := client.Registries(ctx, org.ID)
	assert.Success(t, "get registries", err)

	var dockerhubID string
	for _, r := range registries {
		if r.Registry == "index.docker.io" {
			dockerhubID = r.ID
		}
	}
	assert.True(t, "docker hub registry found", dockerhubID != "")

	imgs, err := client.OrganizationImages(ctx, org.ID)
	assert.Success(t, "get org images", err)
	found := false
	for _, i := range imgs {
		if i.Repository == img {
			found = true
		}
	}
	if !found {
		// ignore this error for now as it causes a race with other parallel tests
		_, _ = client.ImportImage(ctx, coder.ImportImageReq{
			RegistryID:      &dockerhubID,
			OrgID:           org.ID,
			Repository:      img,
			Tag:             "latest",
			DefaultCPUCores: 2.5,
			DefaultDiskGB:   22,
			DefaultMemoryGB: 3,
		})
	}
}
