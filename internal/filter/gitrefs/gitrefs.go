package gitrefs

import (
	"fmt"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/gitrefs"
)

// Name of filter
const Name = "gitrefs"

// Help text
var Help = `
gitrefs:<repo>

Produce versions from all refs for a git repository. Name will be the whole ref
like "refs/tags/v2.7.3" and commit will be the commit hash.

Use git filter to get versions from only tags.

gitrefs:https://github.com/git/git.git
`[1:]

// New gitrefs filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix != Name {
		return nil, nil
	}

	if arg == "" {
		return nil, fmt.Errorf("needs a repo")
	}

	return gitRefsFilter{repo: arg}, nil
}

type gitRefsFilter struct {
	repo string
}

func (f gitRefsFilter) String() string {
	return Name + ":" + f.repo
}

func (f gitRefsFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	refPairs, err := gitrefs.Refs(f.repo, gitrefs.AllProtos)
	if err != nil {
		return nil, "", err
	}

	vs := append(filter.Versions{}, versions...)
	for _, p := range refPairs {
		vs = append(vs, filter.NewVersionWithName(p.Name, map[string]string{"commit": p.ObjID}))
	}

	return vs, versionKey, nil
}
