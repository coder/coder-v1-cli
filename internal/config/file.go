package config

type File string

func (f File) Delete() error {
	return rm(string(f))
}

func (f File) Write(s string) error {
	return write(string(f), 0600, []byte(s))
}

func (f File) Read() (string, error) {
	byt, err := read(string(f))
	return string(byt), err
}

var (
	Session File = "session"
	URL     File = "url"
)
