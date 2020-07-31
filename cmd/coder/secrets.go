package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"cdr.dev/coder-cli/internal/entclient"
	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli"
	"golang.org/x/xerrors"

	"go.coder.com/flog"
)

func makeSecretsCmd() cli.Command {
	return cli.Command{
		Name:        "secrets",
		Usage:       "",
		Description: "Interact with secrets objects owned by the active user.",
		Subcommands: []cli.Command{
			{
				Name:        "ls",
				Usage:       "",
				Description: "list all secrets owned by the active user",
				Action:      listSecrets,
			},
			makeCreateSecret(),
			{
				Name:        "rm",
				Usage:       "",
				Description: "",
				Action:      removeSecret,
			},
			{
				Name:        "view",
				Usage:       "",
				Description: "",
				Action:      viewSecret,
			},
		},
		Flags: nil,
	}
}

func makeCreateSecret() cli.Command {
	var (
		fromFile    string
		fromLiteral string
		fromPrompt  bool
		description string
	)

	return cli.Command{
		Name:        "create",
		Usage:       "create a new secret",
		Description: "Create a new secret object to store application secrets and access them securely from within your environments.",
		ArgsUsage:   "[secret_name]",
		Before: func(c *cli.Context) error {
			if c.Args().First() == "" {
				return xerrors.Errorf("[secret_name] is a required argument")
			}
			if fromPrompt && (fromLiteral != "" || fromFile != "") {
				return xerrors.Errorf("--from-prompt cannot be set along with --from-file or --from-literal")
			}
			if fromLiteral != "" && fromFile != "" {
				return xerrors.Errorf("--from-literal and --from-file cannot both be set")
			}
			if !fromPrompt && fromFile == "" && fromLiteral == "" {
				return xerrors.Errorf("one of [--from-literal, --from-file, --from-prompt] is required")
			}
			return nil
		},
		Action: func(c *cli.Context) {
			var (
				client = requireAuth()
				name   = c.Args().First()
				value  string
				err    error
			)
			if fromLiteral != "" {
				value = fromLiteral
			} else if fromFile != "" {
				contents, err := ioutil.ReadFile(fromFile)
				requireSuccess(err, "failed to read file: %v", err)
				value = string(contents)
			} else {
				prompt := promptui.Prompt{
					Label: "value",
					Mask:  '*',
					Validate: func(s string) error {
						if len(s) < 1 {
							return xerrors.Errorf("a length > 0 is required")
						}
						return nil
					},
				}
				value, err = prompt.Run()
				requireSuccess(err, "failed to prompt for value: %v", err)
			}

			err = client.InsertSecret(entclient.InsertSecretReq{
				Name:        name,
				Value:       value,
				Description: description,
			})
			requireSuccess(err, "failed to insert secret: %v", err)
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:        "from-file",
				Usage:       "a file from which to read the value of the secret",
				TakesFile:   true,
				Destination: &fromFile,
			},
			cli.StringFlag{
				Name:        "from-literal",
				Usage:       "the value of the secret",
				Destination: &fromLiteral,
			},
			cli.BoolFlag{
				Name:        "from-prompt",
				Usage:       "enter the secret value through a terminal prompt",
				Destination: &fromPrompt,
			},
			cli.StringFlag{
				Name:        "description",
				Usage:       "a description of the secret",
				Destination: &description,
			},
		},
	}
}

func listSecrets(_ *cli.Context) {
	client := requireAuth()

	secrets, err := client.Secrets()
	requireSuccess(err, "failed to get secrets: %v", err)

	if len(secrets) < 1 {
		flog.Info("No secrets found")
		return
	}

	w := xtabwriter.NewWriter()
	_, err = fmt.Fprintln(w, xtabwriter.StructFieldNames(secrets[0]))
	requireSuccess(err, "failed to write: %v", err)
	for _, s := range secrets {
		s.Value = "******" // value is omitted from bulk responses

		_, err = fmt.Fprintln(w, xtabwriter.StructValues(s))
		requireSuccess(err, "failed to write: %v", err)
	}
	err = w.Flush()
	requireSuccess(err, "failed to flush writer: %v", err)
}

func viewSecret(c *cli.Context) {
	var (
		client = requireAuth()
		name   = c.Args().First()
	)
	if name == "" {
		flog.Fatal("[name] is a required argument")
	}

	secret, err := client.SecretByName(name)
	requireSuccess(err, "failed to get secret by name: %v", err)

	_, err = fmt.Fprintln(os.Stdout, secret.Value)
	requireSuccess(err, "failed to write: %v", err)
}

func removeSecret(c *cli.Context) {
	var (
		client = requireAuth()
		name   = c.Args().First()
	)
	if name == "" {
		flog.Fatal("[name] is a required argument")
	}

	err := client.DeleteSecretByName(name)
	requireSuccess(err, "failed to delete secret: %v", err)

	flog.Info("Successfully deleted secret %q", name)
}
