package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
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
	Port   int    `json:"port"`
	Access string `json:"access"`
}

var urlAccessLevel = map[string]string{
	//Remote API endpoint requires these in uppercase
	"PRIVATE": "Only you can access",
	"ORG":     "All members of your organization can access",
	"AUTHED":  "Authenticated users can access",
	"PUBLIC":  "Anyone on the internet can access this link",
}

func validatePort(port string) (int, error) {
	p, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		flog.Error("Invalid port")
		return 0, err
	}
	if p < 1 {
		// port 0 means 'any free port', which we don't support
		err = strconv.ErrRange
		flog.Error("Port must be > 0")
		return 0, err
	}
	return int(p), nil
}

func accessLevelIsValid(level string) bool {
	_, ok := urlAccessLevel[level]
	if !ok {
		flog.Error("Invalid access level")
	}
	return ok
}

type createSubCmd struct {
	access  string
	urlname string
}

func (sub *createSubCmd) RegisterFlags(fl *pflag.FlagSet) {
	fl.StringVarP(&sub.access, "access", "a", "private", "[private | org | authed | public] set devurl access")
	fl.StringVarP(&sub.urlname, "name", "n", "", "devurl name")
}

func (sub createSubCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:    "create",
		Usage:   "<env name> <port> [--access <level>] [--name <name>]",
		Aliases: []string{"edit"},
		Desc:    "create or update a devurl for external access",
	}
}

// devURLNameValidRx is the regex used to validate devurl names specified
// via the --name subcommand. Named devurls must begin with a letter, and
// consist solely of letters and digits, with a max length of 64 chars.
var devURLNameValidRx = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9]{0,63}$")

// Run creates or updates a devURL, specified by env ID and port
// (fl.Arg(0) and fl.Arg(1)), with access level (fl.Arg(2)) on
// the cemanager.
func (sub createSubCmd) Run(fl *pflag.FlagSet) {
	envName := fl.Arg(0)
	port := fl.Arg(1)
	name := fl.Arg(2)
	access := fl.Arg(3)

	if envName == "" {
		exitUsage(fl)
	}

	portNum, err := validatePort(port)
	if err != nil {
		exitUsage(fl)
	}

	access = strings.ToUpper(sub.access)
	if !accessLevelIsValid(access) {
		exitUsage(fl)
	}

	name = sub.urlname
	if name != "" && !devURLNameValidRx.MatchString(name) {
		flog.Error("update devurl: name must be < 64 chars in length, begin with a letter and only contain letters or digits.")
		return
	}
	entClient := requireAuth()

	env := findEnv(entClient, envName)

	urlID, found := devURLID(portNum, urlList(envName))
	if found {
		flog.Info("Updating devurl for port %v", port)
		err := entClient.UpdateDevURL(env.ID, urlID, portNum, name, access)
		if err != nil {
			flog.Error("update devurl: %s", err.Error())
		}
	} else {
		flog.Info("Adding devurl for port %v", port)
		err := entClient.InsertDevURL(env.ID, portNum, name, access)
		if err != nil {
			flog.Error("insert devurl: %s", err.Error())
		}
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

// devURLID returns the ID of a devURL, given the env name and port
// from a list of DevURL records.
// ("", false) is returned if no match is found.
func devURLID(port int, urls []DevURL) (string, bool) {
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

	portNum, err := validatePort(port)
	if err != nil {
		exitUsage(fl)
	}

	entClient := requireAuth()
	env := findEnv(entClient, envName)

	urlID, found := devURLID(portNum, urlList(envName))
	if found {
		flog.Info("Deleting devurl for port %v", port)
	} else {
		flog.Fatal("No devurl found for port %v", port)
	}

	err = entClient.DelDevURL(env.ID, urlID)
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

	return devURLs
}

// Run gets the list of active devURLs from the cemanager for the
// specified environment and outputs info to stdout.
func (cmd urlsCmd) Run(fl *pflag.FlagSet) {
	envName := fl.Arg(0)
	devURLs := urlList(envName)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.TabIndent)
	for _, devURL := range devURLs {
		fmt.Fprintf(w, "%s\t%d\t%s\n", devURL.URL, devURL.Port, devURL.Access)
	}
	w.Flush()
}

func (cmd *urlsCmd) Subcommands() []cli.Command {
	return []cli.Command{
		&createSubCmd{},
		&delSubCmd{},
	}
}
