package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"cdr.dev/coder-cli/internal/cmd"
	"cdr.dev/coder-cli/internal/x/xterminal"

	"go.coder.com/flog"
)

var (
	version string = "unknown"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if os.Getenv("PPROF") != "" {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	stdoutState, err := xterminal.MakeOutputRaw(os.Stdout.Fd())
	if err != nil {
		flog.Fatal("set output to raw: %v", err)
	}
	defer xterminal.Restore(os.Stdout.Fd(), stdoutState)

	app := cmd.Make()
	app.Version = fmt.Sprintf("%s %s %s/%s", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)

	err = app.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}
