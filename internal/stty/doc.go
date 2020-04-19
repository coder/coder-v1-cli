// Package stty provides facilities for configuring the calling tty.
package stty

import (
	"os/exec"

	"golang.org/x/xerrors"
)

func EnableCBreak() error {
	out, err := exec.Command("stty", "cbreak").CombinedOutput()
	if err != nil {
		return xerrors.Errorf("stty: %w\n%s", err, out)
	}
	return nil
}

func DisableEcho() error {
	out, err := exec.Command("stty", "-echo").CombinedOutput()
	if err != nil {
		return xerrors.Errorf("stty: %w\n%s", err, out)
	}
	return nil
}

func SubTermMode() error {
	err := EnableCBreak()
	if err != nil {
		return err
	}
	err = DisableEcho()
	if err != nil {
		return err
	}
	return nil
}
