package main

import (
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/pkg/browser"
	"github.com/spf13/pflag"
	"go.coder.com/cli"
	"go.coder.com/flog"

	"cdr.dev/coder-cli/internal/config"
	"cdr.dev/coder-cli/internal/loginsrv"
)

type loginCmd struct {
}

func (cmd loginCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "login",
		Usage: "[Coder Enterprise URL]",
		Desc:  "authenticate this client for future operations",
	}
}
func (cmd loginCmd) Run(fl *pflag.FlagSet) {
	rawURL := fl.Arg(0)
	if rawURL == "" {
		exitUsage(fl)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		flog.Fatal("parse url: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		flog.Fatal("create login server: %+v", err)
	}
	defer listener.Close()

	srv := &loginsrv.Server{
		TokenCond: sync.NewCond(&sync.Mutex{}),
	}
	go func() {
		_ = http.Serve(
			listener, srv,
		)
	}()

	err = config.URL.Write(
		(&url.URL{Scheme: u.Scheme, Host: u.Host}).String(),
	)
	if err != nil {
		flog.Fatal("write url: %v", err)
	}

	authURL := url.URL{
		Scheme:   u.Scheme,
		Host:     u.Host,
		Path:     "/internal-auth/",
		RawQuery: "local_service=http://" + listener.Addr().String(),
	}

	err = browser.OpenURL(authURL.String())
	if err != nil {
		// Tell the user to visit the URL instead.
		flog.Info("visit %s to login", authURL.String())
	}
	srv.TokenCond.L.Lock()
	srv.TokenCond.Wait()
	err = config.Session.Write(srv.Token)
	srv.TokenCond.L.Unlock()
	if err != nil {
		flog.Fatal("set session: %v", err)
	}
	flog.Success("logged in")
}
