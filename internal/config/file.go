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

// The following files configure and authenticate the coder-cli in ~/.coder/.
const (
	CredentialsFile File = "credentials.yaml"
	ConfigFile      File = "config.yaml"
)

// Credentials defines the schema for the ~/.coder/credentials.yaml file.
type Credentials struct {
	DashboardURL string `yaml:"url"`
	SessionToken string `yaml:"session"`
}

// CoderConfig defines the schema for the ~/.coder/config.yaml file.
type CoderConfig struct {
	Version  string        `yaml:"version"`
	Defaults CoderDefaults `yaml:"defaults"`
}

// CoderDefaults defines the schema for the default user configuration located in the global config file.
type CoderDefaults struct {
	Environment string `yaml:"environment"`
	Editor      Editor `yaml:"editor"`
}

// Editor defines an editor which coder-cli can open.
type Editor string

// The following editors may be specified as targets of `coder open`.
const (
	EditorVSCode        Editor = "vscode"
	EditorBrowserVSCode Editor = "browser-vscode"
	EditorGoland        Editor = "goland"
	EditorWebStorm      Editor = "webstorm"
)
