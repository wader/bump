package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/wader/bump/internal/cli"
	"github.com/wader/bump/internal/github"
	"github.com/wader/bump/internal/githubaction"
)

var version = "dev"

// OS implements bump.OS using os
type OS struct{}

// Args returns os args
func (OS) Args() []string {
	return os.Args
}

// Getenv return env using os env
func (OS) Getenv(name string) string {
	return os.Getenv(name)
}

// Stdout returns os stdout
func (OS) Stdout() io.Writer {
	return os.Stdout
}

// Stderr returns os stderr
func (OS) Stderr() io.Writer {
	return os.Stderr
}

// WriteFile writes os file
func (OS) WriteFile(filename string, data []byte) error {
	return ioutil.WriteFile(filename, data, 0644)
}

// ReadFile read os file
func (OS) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

// Glob returns list of matched os files
func (OS) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

func main() {
	o := OS{}
	var r interface{ Run() []error }

	if github.IsActionEnv(o.Getenv) {
		r = githubaction.Command{
			Version: version,
			OS:      o,
		}
	} else {
		r = cli.Command{
			Version: version,
			OS:      o,
		}
	}

	if errs := r.Run(); errs != nil {
		os.Exit(1)
	}
}
