package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/spf13/pflag"

	"go.coder.com/cli"
	"go.coder.com/flog"
)

type urlsCmd struct{}

type DevURL struct {
	URL    string `json:"url"`
	Port   string `json:"port"`
	Access string `json:"access"`
}

func (cmd urlsCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "urls",
		Usage: "<env name>",
		Desc:  "get all development urls for external access",
	}
}

func (cmd urlsCmd) Run(fl *pflag.FlagSet) {
	var envName = fl.Arg(0)

	if envName == "" {
		exitUsage(fl)
	}

	entClient := requireAuth()

	env := findEnv(entClient, envName)

	reqString := "%s/api/environments/%s/devurls?session_token=%s"
	reqUrl := fmt.Sprintf(reqString, entClient.BaseURL, env.ID, entClient.Token)

	resp, err := http.Get(reqUrl)
	if err != nil {
		flog.Fatal("%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		flog.Fatal("non-success status code: %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)

	var devURLs = make([]DevURL, 0)
	err = dec.Decode(&devURLs)
	if err != nil {
		flog.Fatal("%v", err)
	}

	if len(devURLs) == 0 {
		fmt.Printf("no dev urls were found for environment: %s\n", envName)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.TabIndent)
	for _, devURL := range devURLs {
		fmt.Fprintf(w, "%s\t%s\t%s\n", devURL.URL, devURL.Port, devURL.Access)
	}
	w.Flush()
}
