package bump

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Env is an environment the a bump is running in
type Env interface {
	Args() []string
	Stdout() io.Writer
	Stderr() io.Writer
	WriteFile(filename string, data []byte) error
	ReadFile(filename string) ([]byte, error)
	Glob(pattern string) ([]string, error)
}

// OSEnv is a Enver that uses os
type OSEnv struct{}

// Args returns os args
func (OSEnv) Args() []string {
	return os.Args
}

// Stdout returns os stdout
func (OSEnv) Stdout() io.Writer {
	return os.Stdout
}

// Stderr returns os stderr
func (OSEnv) Stderr() io.Writer {
	return os.Stderr
}

// WriteFile writes os file
func (OSEnv) WriteFile(filename string, data []byte) error {
	return ioutil.WriteFile(filename, data, 0644)
}

// ReadFile read os file
func (OSEnv) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

// Glob returns list of matched os files
func (OSEnv) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
