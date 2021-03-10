package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/kirsle/configdir"
)

var configRoot = configdir.LocalConfig("coder")

// SetRoot overrides the package-level config root configuration.
func SetRoot(root string) {
	configRoot = root
}

// open opens a file in the configuration directory,
// creating all intermediate directories.
func open(path string, flag int, mode os.FileMode) (*os.File, error) {
	path = filepath.Join(configRoot, path)

	err := os.MkdirAll(filepath.Dir(path), 0750)
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
	return os.Remove(filepath.Join(configRoot, path))
}
