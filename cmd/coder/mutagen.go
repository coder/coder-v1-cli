package main

import (
	"os"
	_ "unsafe"

	"github.com/spf13/pflag"
	"go.coder.com/cli"
)

//go:linkname mutagenMain github.com/mutagen-io/mutagen/cmd/mutagen.main
func mutagenMain()

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
	mutagenMain()
}

