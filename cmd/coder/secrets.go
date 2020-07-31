package main

import (
	"fmt"
	"os"

	"cdr.dev/coder-cli/internal/entclient"
	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"cdr.dev/coder-cli/internal/x/xvalidate"
	"github.com/spf13/pflag"
	"golang.org/x/xerrors"

	"go.coder.com/flog"

	"go.coder.com/cli"
)

var (
	_ cli.FlaggedCommand = secretsCmd{}
	_ cli.ParentCommand  = secretsCmd{}

	_ cli.FlaggedCommand = &listSecretsCmd{}
	_ cli.FlaggedCommand = &createSecretCmd{}
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
	name, value, description string
}

func (cmd *createSecretCmd) Validate() (e []error) {
	if cmd.name == "" {
		e = append(e, xerrors.New("--name is a required flag"))
	}
	if cmd.value == "" {
		e = append(e, xerrors.New("--value is a required flag"))
	}
	return e
}

func (cmd *createSecretCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "create",
		Usage: `--name MYSQL_KEY --value 123456 --description "MySQL credential for database access"`,
		Desc:  "insert a new secret",
	}
}

func (cmd *createSecretCmd) Run(fl *pflag.FlagSet) {
	var (
		client = requireAuth()
	)
	xvalidate.Validate(cmd)

	err := client.InsertSecret(entclient.InsertSecretReq{
		Name:        cmd.name,
		Value:       cmd.value,
		Description: cmd.description,
	})
	requireSuccess(err, "failed to insert secret: %v", err)
}

func (cmd *createSecretCmd) RegisterFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.name, "name", "", "the name of the secret")
	fl.StringVar(&cmd.value, "value", "", "the value of the secret")
	fl.StringVar(&cmd.description, "description", "", "a description of the secret")
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
