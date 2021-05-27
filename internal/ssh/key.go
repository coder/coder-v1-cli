package ssh

import (
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/xerrors"
)

const (
	coderPrivateKey = "coder_enterprise"
	privateKeyPerms = 0600
)

func WriteSSHKey(key []byte) error {
	filepath, err := DefaultPrivateKeyPath()
	if err != nil {
		return xerrors.Errorf("get private key filepath: %w", err)
	}

	err = os.WriteFile(filepath, key, privateKeyPerms)
	if err != nil {
		return xerrors.Errorf("write file: %w", err)
	}

	return nil
}

func DefaultPrivateKeyPath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", xerrors.Errorf("get current user: %w", err)
	}

	return filepath.Join(usr.HomeDir, ".ssh", coderPrivateKey), nil
}
