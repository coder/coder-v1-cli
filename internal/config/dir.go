package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/kirsle/configdir"
	"golang.org/x/xerrors"
)

func dir() (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", xerrors.Errorf("get home dir: %w", err)
	}
	return filepath.Join(homedir, ".coder"), nil
}

// MigrateFromOld ensures that users of the CLI are properly authenticated with the new ~/.coder/credentials.yaml
// authentication schema.
func MigrateFromOld() error {
	olddir := configdir.LocalConfig("coder")
	_, err := os.Stat(olddir)
	if err != nil {
		// if we can't stat the old config dir, assume it does not exist
		return nil
	}
	session, err := ioutil.ReadFile(filepath.Join(olddir, "session"))
	if err != nil {
		return xerrors.Errorf("read session: %w", err)
	}
	url, err := ioutil.ReadFile(filepath.Join(olddir, "url"))
	if err != nil {
		return xerrors.Errorf("read session: %w", err)
	}
	creds := Credentials{
		SessionToken: string(session),
		DashboardURL: string(url),
	}
	if err := CredentialsFile.WriteYAML(creds); err != nil {
		return xerrors.Errorf("write credentials file: %w", err)
	}
	if err := os.RemoveAll(olddir); err != nil {
		return xerrors.Errorf("remove old config dir: %w", err)
	}
	return nil
}

// open opens a file in the configuration directory,
// creating all intermediate directories.
func open(path string, flag int, mode os.FileMode) (*os.File, error) {
	configDir, err := dir()
	if err != nil {
		return nil, err
	}
	path = filepath.Join(configDir, path)

	err = os.MkdirAll(filepath.Dir(path), 0750)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(path, flag, mode)
}

func write(path string, mode os.FileMode, dat []byte) error {
	fi, err := open(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, mode)
	if err != nil {
		return err
	}
	defer fi.Close()
	_, err = fi.Write(dat)
	return err
}

func read(path string) ([]byte, error) {
	fi, err := open(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer fi.Close()
	return ioutil.ReadAll(fi)
}

func rm(path string) error {
	configDir, err := dir()
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(configDir, path))
}
