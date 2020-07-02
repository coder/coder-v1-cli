package sync

import (
	"os/exec"
	"os"
)

func mutagenCmd(args ...string) (*exec.Cmd, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, err
	}

	return &exec.Cmd{
		Path: path,
		Args: append([]string{"mutagen"}, args...),
	}, nil
}

func becomeMutagen(args ...string) error {
	c, err := mutagenCmd(args...)
	if err != nil {
		return err
	}


	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

