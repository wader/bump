package git

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/gitrefs"
)

// Name of filter
const Name = "git"

// Help text
var Help = `
git:<repo> or <repo.git>

Produce versions from tags for a git repository. Name will be
the version found in the tag, commit the commit hash or tag object.

Use gitrefs filter to get all refs unfiltered.

https://github.com/git/git.git|*
`[1:]

// default ref filter
// refs/tags/<non-digits><version-number> -> version-number
var refFilterRe = regexp.MustCompile(`^refs/tags/[^\d]*([\d\.\-]+)$`)

// New git filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	// TODO hmm
	if prefix == Name ||
		(strings.HasSuffix(arg, ".git") &&
			(prefix == "git" || prefix == "http" || prefix == "https")) {
		if strings.HasPrefix(arg, "//") {
			arg = prefix + ":" + arg
		}
	} else {
		return nil, nil
	}

	if arg == "" {
		return nil, fmt.Errorf("needs a repo")
	}

	return gitFilter{repo: arg}, nil
}

type gitFilter struct {
	repo string
}

func (f gitFilter) String() string {
	return Name + ":" + f.repo
}

func (f gitFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	refPairs, err := gitrefs.Refs(f.repo, gitrefs.AllProtos)
	if err != nil {
		return nil, "", err
	}

	vs := append(filter.Versions{}, versions...)
	for _, p := range refPairs {
		sm := refFilterRe.FindStringSubmatch(p.Name)
		// find first non-empty submatch
		if sm == nil {
			continue
		}
		var name string
		for _, m := range sm[1:] {
			if m != "" {
				name = m
				break
			}
		}
		vs = append(vs, filter.NewVersionWithName(name, map[string]string{"commit": p.ObjID}))
	}

	return vs, versionKey, nil
}
