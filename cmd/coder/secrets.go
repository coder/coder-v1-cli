package main

import (
	"fmt"
	"os"

	"cdr.dev/coder-cli/internal/entclient"
	"cdr.dev/coder-cli/internal/xcli"
	"github.com/spf13/pflag"
	"golang.org/x/xerrors"

	"go.coder.com/flog"

	"go.coder.com/cli"
)

var (
	_ cli.FlaggedCommand = secretsCmd{}
	_ cli.ParentCommand  = secretsCmd{}

	_ cli.FlaggedCommand = &listSecretsCmd{}
	_ cli.FlaggedCommand = &addSecretCmd{}
)

type secretsCmd struct {
}

func (cmd secretsCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name: "secrets",
		Desc: "interact with secrets owned by the authenticated user",
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
		&addSecretCmd{},
		&deleteSecretsCmd{},
	}
}

type listSecretsCmd struct{}

func (cmd listSecretsCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name: "ls",
		Desc: "list all secrets owned by the authenticated user",
	}
}

func (cmd listSecretsCmd) Run(fl *pflag.FlagSet) {
	client := requireAuth()

	secrets, err := client.Secrets()
	xcli.RequireSuccess(err, "failed to get secrets: %v", err)

	w := xcli.HumanReadableWriter()
	if len(secrets) > 0 {
		_, err := fmt.Fprintln(w, xcli.TabDelimitedStructHeaders(secrets[0]))
		xcli.RequireSuccess(err, "failed to write: %v", err)
	}
	for _, s := range secrets {
		s.Value = "******" // value is omitted from bulk responses

		_, err = fmt.Fprintln(w, xcli.TabDelimitedStructValues(s))
		xcli.RequireSuccess(err, "failed to write: %v", err)
	}
	err = w.Flush()
	xcli.RequireSuccess(err, "failed to flush writer: %v", err)
}

func (cmd *listSecretsCmd) RegisterFlags(fl *pflag.FlagSet) {}

type viewSecretsCmd struct{}

func (cmd viewSecretsCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "view",
		Usage: "[secret_name]",
		Desc:  "view a secret owned by the authenticated user",
	}
}

func (cmd viewSecretsCmd) Run(fl *pflag.FlagSet) {
	var (
		client = requireAuth()
		name   = fl.Arg(0)
	)

	secret, err := client.SecretByName(name)
	xcli.RequireSuccess(err, "failed to get secret by name: %v", err)

	_, err = fmt.Fprintln(os.Stdout, secret.Value)
	xcli.RequireSuccess(err, "failed to write: %v", err)
}

type addSecretCmd struct {
	name, value, description string
}

func (cmd *addSecretCmd) Validate() (e []error) {
	if cmd.name == "" {
		e = append(e, xerrors.New("--name is a required flag"))
	}
	if cmd.value == "" {
		e = append(e, xerrors.New("--value is a required flag"))
	}
	return e
}

func (cmd *addSecretCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "add",
		Usage: `--name MYSQL_KEY --value 123456 --description "MySQL credential for database access"`,
		Desc:  "insert a new secret",
	}
}

func (cmd *addSecretCmd) Run(fl *pflag.FlagSet) {
	var (
		client = requireAuth()
	)
	xcli.Validate(cmd)

	err := client.InsertSecret(entclient.InsertSecretReq{
		Name:        cmd.name,
		Value:       cmd.value,
		Description: cmd.description,
	})
	xcli.RequireSuccess(err, "failed to insert secret: %v", err)
}

func (cmd *addSecretCmd) RegisterFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.name, "name", "", "the name of the secret")
	fl.StringVar(&cmd.value, "value", "", "the value of the secret")
	fl.StringVar(&cmd.description, "description", "", "a description of the secret")
}

type deleteSecretsCmd struct{}

func (cmd *deleteSecretsCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "rm",
		Usage: "[secret_name]",
		Desc:  "remove a secret by name",
	}
}

func (cmd *deleteSecretsCmd) Run(fl *pflag.FlagSet) {
	var (
		client = requireAuth()
		name   = fl.Arg(0)
	)

	err := client.DeleteSecretByName(name)
	xcli.RequireSuccess(err, "failed to delete secret: %v", err)

	flog.Info("Successfully deleted secret %q", name)
}
