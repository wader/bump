package bump

import (
	"io"
)

type OS interface {
	Args() []string
	Getenv(name string) string
	Stdout() io.Writer
	Stderr() io.Writer
	WriteFile(filename string, data []byte) error
	ReadFile(filename string) ([]byte, error)
	Glob(pattern string) ([]string, error)
}
