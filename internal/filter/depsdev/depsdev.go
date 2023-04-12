package depsdev

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/wader/bump/internal/filter"
)

const depsDevURLTemplate = `https://api.deps.dev/v3alpha/systems/%s/packages/%s`

// Name of filter
const Name = "depsdev"

// Help text
var Help = `
depsdev:<system>:<package>

Produce versions from https://deps.dev.

Supported package systems npm, go, maven, pypi and cargo.

depsdev:npm:react|*
depsdev:go:golang.org/x/net
depsdev:maven:log4j:log4j|^1
depsdev:pypi:av|*
depsdev:cargo:serde|*
`[1:]

// New depsdev filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix != Name {
		return nil, nil
	}
	if arg == "" {
		return nil, fmt.Errorf("needs a image name")
	}

	parts := strings.SplitN(arg, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("requires depsdev:<system>:<package>")
	}

	return depsDevFilter{
		system:   parts[0],
		package_: parts[1],
	}, nil
}

type depsDevFilter struct {
	system   string
	package_ string
}

func (f depsDevFilter) String() string {
	return Name + ":" + f.system + ":" + f.package_
}

func (f depsDevFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	var response struct {
		Versions []struct {
			VersionKey struct {
				Version string `json:"version"`
			} `json:"versionKey"`
		} `json:"versions"`
	}

	r, err := http.Get(
		fmt.Sprintf(
			depsDevURLTemplate,
			url.PathEscape(f.system),
			url.PathEscape(f.package_)),
	)
	if err != nil {
		return nil, "", err
	}
	defer r.Body.Close()

	if r.StatusCode/100 != 2 {
		return nil, "", fmt.Errorf("error response: %s", r.Status)
	}

	jd := json.NewDecoder(r.Body)
	if err := jd.Decode(&response); err != nil {
		return nil, "", err
	}

	var vs filter.Versions
	for _, v := range response.Versions {
		vs = append(vs, filter.NewVersionWithName(
			// TODO: better way, go versions start with "v"
			strings.TrimLeft(v.VersionKey.Version, "v"),
			nil,
		))
	}

	return vs, versionKey, nil
}
