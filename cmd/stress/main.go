package main

import (
	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/cmd"
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
	"time"
)

const (
	level = 1
	statSeconds = 30
	orgID = "default"
	namePrefix = "stress"
	imageID = "5f282a6a-2b0fd967178a95ac8078ce31" // master.cdr.dev ubuntu
	imageTag = "latest"
	cpuCores = 1
	memoryGB = 1
	diskGB = 10
)

func main() {
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)
	c := cmd.RequireAuth()
	uid := time.Now().Unix()

	for i := 0; i < level; i++ {
		i := i //https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			er := coder.CreateEnvironmentRequest{
				Name:     fmt.Sprintf("%s-%d-%d", namePrefix, uid, i),
				ImageID:  imageID,
				ImageTag: imageTag,
				CPUCores: cpuCores,
				MemoryGB: memoryGB,
				DiskGB:   diskGB,
			}

			log.Printf("%s: Creating environment\n", er.Name)
			env, err := c.CreateEnvironment(ctx, orgID, er)
			if err != nil {
				return fmt.Errorf("create environment: %w", err)
			}

			log.Printf("%s: Waiting for environment to be ready\n", er.Name)
			err = c.WaitForEnvironmentReady(ctx, env.ID)
			if err != nil {
				return fmt.Errorf("wait for environment ready: %w", err)
			}

			log.Printf("%s: Watching environment stats for %d seconds\n", er.Name, statSeconds)
			err = c.WatchEnvironmentStats(ctx, env.ID, time.Second * statSeconds)
			if err != nil {
				return fmt.Errorf("watch environment stats: %w", err)
			}

			log.Printf("%s: Deleting environment\n", er.Name)
			err = c.DeleteEnvironment(ctx, env.ID)
			if err != nil {
				return fmt.Errorf("delete environment: %w", err)
			}
			return nil
		})
	}
	err := g.Wait()
	if err != nil {
		log.Fatal(err)
	}
}