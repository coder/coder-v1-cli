package main

import (
	"context"
	"io"
	"os"

	"cdr.dev/coder-cli/internal/entclient"
	"github.com/spf13/pflag"
	"go.coder.com/cli"
	"go.coder.com/flog"
	"nhooyr.io/websocket"
)

type logsCmd struct {
	container string
	follow    bool
}

func (c *logsCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "logs",
		Usage: "<env name> [flags]",
		Desc:  "get logs from an environment container",
	}
}

func (c *logsCmd) RegisterFlags(fl *pflag.FlagSet) {
	fl.BoolVar(&c.follow, "follow", false, "Specify true to stream the logs")
	fl.StringVar(&c.container, "container", "", "Stream logs from a container. Defaults to main development container")
}

func (c *logsCmd) Run(fl *pflag.FlagSet) {
	if len(fl.Args()) < 1 {
		exitUsage(fl)
	}

	var (
		envName   = fl.Arg(0)
		entClient = requireAuth()
		env       = findEnv(entClient, envName)
	)

	conn, err := entClient.Logs(context.Background(), env, entclient.LogOptions{
		Container: c.container,
		Follow:    c.follow,
	})
	if err != nil {
		flog.Fatal("%v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	_, rd, err := conn.Reader(context.Background())
	if err != nil {
		flog.Fatal("%v", err)
	}

	io.Copy(os.Stdout, rd)
}
