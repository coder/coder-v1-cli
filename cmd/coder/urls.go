package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/pflag"

	"go.coder.com/cli"
	"go.coder.com/flog"
)

type urlsCmd struct{}

// DevURL is the parsed json response record for a devURL from cemanager
type DevURL struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Port   string `json:"port"`
	Access string `json:"access"`
}

var urlAccessLevel = map[string]string{
	//Remote API endpoint requires these in uppercase
	"PRIVATE": "Only you can access",
	"ORG":     "All members of your organization can access",
	"AUTHED":  "Authenticated users can access",
	"PUBLIC":  "Anyone on the internet can access this link",
}

func portIsValid(port string) bool {
	p, err := strconv.ParseUint(port, 10, 16)
	if p < 1 {
		// port 0 means 'any free port', which we don't support
		err = strconv.ErrRange
	}
	if err != nil {
		fmt.Println("Invalid port")
	}
	return err == nil
}

func accessLevelIsValid(level string) bool {
	_, ok := urlAccessLevel[level]
	if !ok {
		fmt.Println("Invalid access level")
	}
	return ok
}

type createSubCmd struct {
	access string
}

func (sub *createSubCmd) RegisterFlags(fl *pflag.FlagSet) {
	fl.StringVarP(&sub.access, "access", "a", "private", "[private | org | authed | public] set devurl access")
}

func (sub createSubCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "create",
		Usage: "<env name> <port> [--access <level>]",
		Desc:  "create/update a devurl for external access",
	}
}

// Run creates or updates a devURL, specified by env ID and port
// (fl.Arg(0) and fl.Arg(1)), with access level (fl.Arg(2)) on
// the cemanager.
func (sub createSubCmd) Run(fl *pflag.FlagSet) {
	envName := fl.Arg(0)
	port := fl.Arg(1)
	access := fl.Arg(2)

	if envName == "" {
		exitUsage(fl)
	}

	if !portIsValid(port) {
		exitUsage(fl)
	}

	access = strings.ToUpper(sub.access)
	if !accessLevelIsValid(access) {
		exitUsage(fl)
	}

	entClient := requireAuth()

	env := findEnv(entClient, envName)

	_, found := devURLID(port, urlList(envName))
	if found {
		fmt.Printf("Updating devurl for port %v\n", port)
	} else {
		fmt.Printf("Adding devurl for port %v\n", port)
	}

	err := entClient.UpsertDevURL(env.ID, port, access)
	if err != nil {
		flog.Error("upsert devurl: %s", err.Error())
	}
}

type delSubCmd struct{}

func (sub delSubCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "del",
		Usage: "<env name> <port>",
		Desc:  "delete a devurl",
	}
}

// devURLID returns the ID of a devURL, given the env name and port.
// ("", false) is returned if no match is found.
func devURLID(port string, urls []DevURL) (string, bool) {
	for _, url := range urls {
		if url.Port == port {
			return url.ID, true
		}
	}
	return "", false
}

// Run deletes a devURL, specified by env ID and port, from the cemanager.
func (sub delSubCmd) Run(fl *pflag.FlagSet) {
	envName := fl.Arg(0)
	port := fl.Arg(1)

	if envName == "" {
		exitUsage(fl)
	}

	if !portIsValid(port) {
		exitUsage(fl)
	}

	entClient := requireAuth()

	env := findEnv(entClient, envName)

	urlID, found := devURLID(port, urlList(envName))
	if found {
		fmt.Printf("Deleting devurl for port %v\n", port)
	} else {
		flog.Fatal("No devurl found for port %v", port)
	}

	err := entClient.DelDevURL(env.ID, urlID)
	if err != nil {
		flog.Error("delete devurl: %s", err.Error())
	}
}

func (cmd urlsCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "urls",
		Usage: "<env name>",
		Desc:  "get all development urls for external access",
	}
}

// urlList returns the list of active devURLs from the cemanager.
func urlList(envName string) []DevURL {
	entClient := requireAuth()
	env := findEnv(entClient, envName)

	reqString := "%s/api/environments/%s/devurls?session_token=%s"
	reqURL := fmt.Sprintf(reqString, entClient.BaseURL, env.ID, entClient.Token)

	resp, err := http.Get(reqURL)
	if err != nil {
		flog.Fatal("%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		flog.Fatal("non-success status code: %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)

	devURLs := make([]DevURL, 0)
	err = dec.Decode(&devURLs)
	if err != nil {
		flog.Fatal("%v", err)
	}

	if len(devURLs) == 0 {
		fmt.Printf("no dev urls were found for environment: %s\n", envName)
	}

	return devURLs
}

// Run gets the list of active devURLs from the cemanager for the
// specified environment and outputs info to stdout.
func (cmd urlsCmd) Run(fl *pflag.FlagSet) {
	envName := fl.Arg(0)
	devURLs := urlList(envName)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.TabIndent)
	for _, devURL := range devURLs {
		fmt.Fprintf(w, "%s\t%s\t%s\n", devURL.URL, devURL.Port, devURL.Access)
	}
	w.Flush()
}

func (cmd *urlsCmd) Subcommands() []cli.Command {
	return []cli.Command{
		&createSubCmd{},
		&delSubCmd{},
	}
}
