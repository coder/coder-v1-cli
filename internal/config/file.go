package config

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

// Coder CLI configuration files.
var (
	Session File = "session"
	URL     File = "url"
)
