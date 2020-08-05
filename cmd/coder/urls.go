package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"github.com/urfave/cli"
	"golang.org/x/xerrors"

	"go.coder.com/flog"
)

func makeURLCmd() cli.Command {
	var outputFmt string
	return cli.Command{
		Name:   "urls",
		Usage:  "Interact with environment DevURLs",
		Action: exitHelp,
		Subcommands: []cli.Command{
			makeCreateDevURL(),
			{
				Name:      "ls",
				Usage:     "List all DevURLs for an environment",
				ArgsUsage: "[env_name]",
				Before: func(c *cli.Context) error {
					if !(outputFmt == "json" || outputFmt == "human") {
						return xerrors.Errorf("unknown --output value %q")
					}
					if c.Args().First() == "" {
						return xerrors.New("argument [env_name] is required")
					}
					return nil
				},
				Action: makeListDevURLs(&outputFmt),
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:        "output",
						Usage:       "human | json",
						Value:       "human",
						Destination: &outputFmt,
					},
				},
			},
			{
				Name:      "rm",
				Usage:     "Remove a dev url",
				ArgsUsage: "[env_name] [port]",
				Before: func(c *cli.Context) error {
					var (
						envName = c.Args().First()
						port    = c.Args().Get(1)
					)
					if envName == "" || port == "" {
						return xerrors.Errorf("[env_name] and [port] are required arguments")
					}
					return nil
				},
				Action: removeDevURL,
			},
		},
	}
}

// DevURL is the parsed json response record for a devURL from cemanager
type DevURL struct {
	ID     string `json:"id" tab:"-"`
	URL    string `json:"url"`
	Port   int    `json:"port"`
	Name   string `json:"name" tab:"-"`
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

// Run gets the list of active devURLs from the cemanager for the
// specified environment and outputs info to stdout.
func makeListDevURLs(outputFmt *string) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		envName := c.Args().First()
		devURLs, err := urlList(envName)
		if err != nil {
			return err
		}

		switch *outputFmt {
		case "human":
			err := xtabwriter.WriteTable(len(devURLs), func(i int) interface{} {
				return devURLs[i]
			})
			if err != nil {
				return xerrors.Errorf("write table: %w", err)
			}
		case "json":
			err := json.NewEncoder(os.Stdout).Encode(devURLs)
			if err != nil {
				return xerrors.Errorf("encode DevURLs as json: %w", err)
			}
		default:
			return xerrors.Errorf("unknown --output value %q", *outputFmt)
		}
		return nil
	}
}

func makeCreateDevURL() cli.Command {
	var (
		access  string
		urlname string
	)
	return cli.Command{
		Name:      "create",
		Usage:     "Create a new devurl for an environment",
		ArgsUsage: "[env_name] [port] [--access <level>] [--name <name>]",
		Aliases:   []string{"edit"},
		Before: func(c *cli.Context) error {
			if c.Args().First() == "" || c.Args().Get(1) == "" {
				return xerrors.Errorf("[env_name] and [port] are required arguments")
			}
			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:        "access",
				Usage:       "Set DevURL access to [private | org | authed | public]",
				Value:       "private",
				Destination: &access,
			},
			cli.StringFlag{
				Name:        "name",
				Usage:       "DevURL name",
				Required:    true,
				Destination: &urlname,
			},
		},
		// Run creates or updates a devURL
		Action: func(c *cli.Context) error {
			var (
				envName = c.Args().First()
				port    = c.Args().Get(1)
			)

			if envName == "" {
				cli.ShowCommandHelpAndExit(c, c.Command.FullName(), 1)
			}

			portNum, err := validatePort(port)
			if err != nil {
				cli.ShowCommandHelpAndExit(c, c.Command.FullName(), 1)
			}

			access = strings.ToUpper(access)
			if !accessLevelIsValid(access) {
				cli.ShowCommandHelpAndExit(c, c.Command.FullName(), 1)
			}

			if urlname != "" && !devURLNameValidRx.MatchString(urlname) {
				return xerrors.New("update devurl: name must be < 64 chars in length, begin with a letter and only contain letters or digits.")
			}
			entClient := requireAuth()

			env, err := findEnv(entClient, envName)
			if err != nil {
				return err
			}

			urls, err := urlList(envName)
			if err != nil {
				return err
			}

			urlID, found := devURLID(portNum, urls)
			if found {
				flog.Info("Updating devurl for port %v", port)
				err := entClient.UpdateDevURL(env.ID, urlID, portNum, urlname, access)
				if err != nil {
					return xerrors.Errorf("update DevURL: %w", err)
				}
			} else {
				flog.Info("Adding devurl for port %v", port)
				err := entClient.InsertDevURL(env.ID, portNum, urlname, access)
				if err != nil {
					return xerrors.Errorf("insert DevURL: %w", err)
				}
			}
			return nil
		},
	}
}

// devURLNameValidRx is the regex used to validate devurl names specified
// via the --name subcommand. Named devurls must begin with a letter, and
// consist solely of letters and digits, with a max length of 64 chars.
var devURLNameValidRx = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9]{0,63}$")

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
func removeDevURL(c *cli.Context) error {
	var (
		envName = c.Args().First()
		port    = c.Args().Get(1)
	)

	portNum, err := validatePort(port)
	if err != nil {
		return xerrors.Errorf("validate port: %w", err)
	}

	entClient := requireAuth()
	env, err := findEnv(entClient, envName)
	if err != nil {
		return err
	}

	urls, err := urlList(envName)
	if err != nil {
		return err
	}

	urlID, found := devURLID(portNum, urls)
	if found {
		flog.Info("Deleting devurl for port %v", port)
	} else {
		return xerrors.Errorf("No devurl found for port %v", port)
	}

	err = entClient.DelDevURL(env.ID, urlID)
	if err != nil {
		return xerrors.Errorf("delete DevURL: %w", err)
	}
	return nil
}

// urlList returns the list of active devURLs from the cemanager.
func urlList(envName string) ([]DevURL, error) {
	entClient := requireAuth()
	env, err := findEnv(entClient, envName)
	if err != nil {
		return nil, err
	}

	reqString := "%s/api/environments/%s/devurls?session_token=%s"
	reqURL := fmt.Sprintf(reqString, entClient.BaseURL, env.ID, entClient.Token)

	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, xerrors.Errorf("non-success status code: %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)

	devURLs := make([]DevURL, 0)
	err = dec.Decode(&devURLs)
	if err != nil {
		return nil, err
	}

	return devURLs, nil
}
