package main

import (
	"fmt"
	"os"

	"github.com/wader/bump/internal/bump"
	"github.com/wader/bump/internal/githubaction"
)

var version = "dev"

func main() {
	var errs []error

	if os.Getenv("GITHUB_ACTION") != "" {
		errs = githubaction.Run(version)
	} else {
		errs = (bump.Command{
			Version: version,
			Env:     bump.OSEnv{},
		}).Run()
	}

	if errs != nil {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
