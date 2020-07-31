package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"cdr.dev/coder-cli/internal/entclient"
	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"cdr.dev/coder-cli/internal/x/xvalidate"
	"github.com/manifoldco/promptui"
	"github.com/spf13/pflag"
	"golang.org/x/xerrors"

	"go.coder.com/flog"

	"go.coder.com/cli"
)

type secretsCmd struct {
}

func (cmd secretsCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "secrets",
		Usage: "[subcommand]",
		Desc:  "interact with secrets",
	}
}

func (cmd secretsCmd) Run(fl *pflag.FlagSet) {
	exitUsage(fl)
}

func (cmd secretsCmd) RegisterFlags(fl *pflag.FlagSet) {}

func (cmd secretsCmd) Subcommands() []cli.Command {
	return []cli.Command{
		&listSecretsCmd{},
		&viewSecretsCmd{},
		&createSecretCmd{},
		&deleteSecretsCmd{},
	}
}

type listSecretsCmd struct{}

func (cmd *listSecretsCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name: "ls",
		Desc: "list all secrets",
	}
}

func (cmd *listSecretsCmd) Run(fl *pflag.FlagSet) {
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

func (cmd *listSecretsCmd) RegisterFlags(fl *pflag.FlagSet) {}

type viewSecretsCmd struct{}

func (cmd viewSecretsCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "view",
		Usage: "[secret_name]",
		Desc:  "view a secret",
	}
}

func (cmd viewSecretsCmd) Run(fl *pflag.FlagSet) {
	var (
		client = requireAuth()
		name   = fl.Arg(0)
	)
	if name == "" {
		exitUsage(fl)
	}

	secret, err := client.SecretByName(name)
	requireSuccess(err, "failed to get secret by name: %v", err)

	_, err = fmt.Fprintln(os.Stdout, secret.Value)
	requireSuccess(err, "failed to write: %v", err)
}

type createSecretCmd struct {
	description string
	fromFile    string
	fromLiteral string
	fromPrompt  bool
}

func (cmd *createSecretCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "create",
		Usage: `[secret_name] [...flags]`,
		Desc:  "create a new secret",
	}
}

func (cmd *createSecretCmd) Validate(fl *pflag.FlagSet) (e []error) {
	if cmd.fromPrompt && (cmd.fromLiteral != "" || cmd.fromFile != "") {
		e = append(e, xerrors.Errorf("--from-prompt cannot be set along with --from-file or --from-literal"))
	}
	if cmd.fromLiteral != "" && cmd.fromFile != "" {
		e = append(e, xerrors.Errorf("--from-literal and --from-file cannot both be set"))
	}
	if !cmd.fromPrompt && cmd.fromFile == "" && cmd.fromLiteral == "" {
		e = append(e, xerrors.Errorf("one of [--from-literal, --from-file, --from-prompt] is required"))
	}
	return e
}

func (cmd *createSecretCmd) Run(fl *pflag.FlagSet) {
	var (
		client = requireAuth()
		name   = fl.Arg(0)
		value  string
		err    error
	)
	if name == "" {
		exitUsage(fl)
	}
	xvalidate.Validate(fl, cmd)

	if cmd.fromLiteral != "" {
		value = cmd.fromLiteral
	} else if cmd.fromFile != "" {
		contents, err := ioutil.ReadFile(cmd.fromFile)
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
		Description: cmd.description,
	})
	requireSuccess(err, "failed to insert secret: %v", err)
}

func (cmd *createSecretCmd) RegisterFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.fromFile, "from-file", "", "specify a file from which to read the value of the secret")
	fl.StringVar(&cmd.fromLiteral, "from-literal", "", "specify the value of the secret")
	fl.BoolVar(&cmd.fromPrompt, "from-prompt", false, "specify the value of the secret through a prompt")
	fl.StringVar(&cmd.description, "description", "", "specify a description of the secret")
}

type deleteSecretsCmd struct{}

func (cmd *deleteSecretsCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "rm",
		Usage: "[secret_name]",
		Desc:  "remove a secret",
	}
}

func (cmd *deleteSecretsCmd) Run(fl *pflag.FlagSet) {
	var (
		client = requireAuth()
		name   = fl.Arg(0)
	)
	if name == "" {
		exitUsage(fl)
	}

	err := client.DeleteSecretByName(name)
	requireSuccess(err, "failed to delete secret: %v", err)

	flog.Info("Successfully deleted secret %q", name)
}
