package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"cdr.dev/coder-cli/internal/clog"
	"cdr.dev/coder-cli/internal/cmd"
	"cdr.dev/coder-cli/internal/x/xterminal"
)

// Using a global for the version so it can be set at build time using ldflags.
var version = "unknown"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// If requested, spin up the pprof webserver.
	if os.Getenv("PPROF") != "" {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	stdoutState, err := xterminal.MakeOutputRaw(os.Stdout.Fd())
	if err != nil {
		clog.Log(clog.Fatal(fmt.Sprintf("set output to raw: %s", err)))
		os.Exit(1)
	}
	defer func() {
		// Best effort. Would result in broken terminal on window but nothing we can do about it.
		_ = xterminal.Restore(os.Stdout.Fd(), stdoutState)
	}()

	app := cmd.Make()
	app.Version = fmt.Sprintf("%s %s %s/%s", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if err := app.ExecuteContext(ctx); err != nil {
		clog.Log(err)
		os.Exit(1)
	}
}
