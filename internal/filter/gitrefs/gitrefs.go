package gitrefs

import (
	"fmt"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/pair"
	"github.com/wader/bump/internal/gitrefs"
)

// Name of filter
const Name = "gitrefs"

// Help text
var Help = `
gitrefs:<repo>

Produce versions from all refs for a git repository.

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

func (f gitRefsFilter) Filter(ps pair.Slice) (pair.Slice, error) {
	refPairs, err := gitrefs.Refs(f.repo, gitrefs.AllProtos)
	if err != nil {
		return nil, err
	}

	ps = append(pair.Slice{}, ps...)
	for _, p := range refPairs {
		ps = append(ps, pair.Pair{Name: p.Name, Value: p.ObjID})
	}

	return ps, nil
}
