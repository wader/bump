package bump

import (
	"io"
	"io/ioutil"
	"os"
)

// OSEnv is a command Enver that uses os
type OSEnv struct{}

func (OSEnv) Args() []string {
	return os.Args
}

func (OSEnv) Stdout() io.Writer {
	return os.Stdout
}

func (OSEnv) Stderr() io.Writer {
	return os.Stderr
}

func (OSEnv) WriteFile(filename string, data []byte) error {
	return ioutil.WriteFile(filename, data, 0644)
}

func (OSEnv) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}
