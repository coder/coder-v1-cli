package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/pflag"

	"go.coder.com/cli"
	"go.coder.com/flog"
)

type urlCmd struct {
}

type DevURL struct {
	Url string `json:"url"`
}

func (cmd urlCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "url",
		Usage: "<env name> <port>",
		Desc:  "get a development url for external access",
	}
}

func (cmd urlCmd) Run(fl *pflag.FlagSet) {
	var (
		envName = fl.Arg(0)
		port    = fl.Arg(1)
	)
	if envName == "" || port == "" {
		exitUsage(fl)
	}

	entClient := requireAuth()

	env := findEnv(entClient, envName)

	reqString := "%s/api/environments/%s/devurl?port=%s&session_token=%s"
	reqUrl := fmt.Sprintf(reqString, entClient.BaseURL, env.ID, port, entClient.Token)

	resp, err := http.Get(reqUrl)
	if err != nil {
		flog.Fatal("%v", err)
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)

	var devUrl DevURL
	err = dec.Decode(&devUrl)
	if err != nil {
		flog.Fatal("%v", err)
	}

	fmt.Println(devUrl.Url)
}
