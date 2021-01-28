package config

import (
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

// File provides convenience methods for interacting with *os.File.
type File string

// Delete deletes the file.
func (f File) Delete() error {
	return rm(string(f))
}

// Write writes the string to the file.
func (f File) Write(s string) error {
	return write(string(f), 0600, []byte(s))
}

// Read reads the file to a string.
func (f File) Read() (string, error) {
	byt, err := read(string(f))
	return string(byt), err
}

// UnmarshalYAML reads the file and unmarshals as yaml.
func (f File) UnmarshalYAML(out interface{}) error {
	byt, err := read(string(f))
	if err != nil {
		return xerrors.Errorf("read file: %w", err)
	}
	if err := yaml.Unmarshal(byt, out); err != nil {
		return xerrors.Errorf("unmarshal yaml: %w", err)
	}
	return nil
}

// WriteYAML writes the file as yaml.
func (f File) WriteYAML(in interface{}) error {
	encoded, err := yaml.Marshal(in)
	if err != nil {
		return err
	}
	return f.Write(string(encoded))
}

const (
	CredentialsFile File = "credentials.yaml"
	ConfigFile      File = "config.yaml"
)

type Credentials struct {
	DashboardURL string `yaml:"url"`
	SessionToken string `yaml:"session"`
}

type CoderConfig struct {
	Version  string        `yaml:"version"`
	Defaults CoderDefaults `yaml:"defaults"`
}

type CoderDefaults struct {
	Environment string `yaml:"environment"`
	Editor      Editor `yaml:"editor"`
}

type Editor string

const (
	EditorVSCode        Editor = "vscode"
	EditorBrowserVSCode Editor = "browser-vscode"
	EditorGoland        Editor = "goland"
	EditorWebStorm      Editor = "webstorm"
)
