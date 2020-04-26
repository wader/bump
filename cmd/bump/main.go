package main

import (
	"os"

	"github.com/wader/bump/internal/bump"
	"github.com/wader/bump/internal/cli"
	"github.com/wader/bump/internal/githubaction"
)

var version = "dev"

func main() {
	var r interface{ Run() []error }

	if os.Getenv("GITHUB_ACTION") != "" {
		r = githubaction.Command{
			Version: version,
			Env:     bump.OSEnv{},
		}
	} else {
		r = cli.Command{
			Version: version,
			Env:     bump.OSEnv{},
		}
	}

	if errs := r.Run(); errs != nil {
		os.Exit(1)
	}
}
