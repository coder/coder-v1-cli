package main

import (
	"github.com/mutagen-io/mutagen/pkg/command/mutagen"
	"github.com/spf13/pflag"
	"go.coder.com/cli"
	"os"
)

type mutagenCmd struct {

}

func (m *mutagenCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:    "mutagen",
		Usage:   "[mutagen args...]",
		Desc:    "call the embedded mutagen",
		RawArgs: true,
	}
}

func (m *mutagenCmd) Run(_ *pflag.FlagSet) {
	// Pop out first argument (coder) so mutagen thinks its mutagen.
	copy(os.Args, os.Args[1:])
	os.Args = os.Args[:len(os.Args)-1]
	mutagen.Main()
}

