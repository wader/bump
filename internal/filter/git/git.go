package git

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/pair"
	"github.com/wader/bump/internal/gitrefs"
)

// Name of filter
const Name = "git"

// Help text
var Help = `
git:<repo> or <repo.git>

Produce versions from tags and branches from a git repository. Name will be
the tag or branch name, value the commit hash or tag object.

https://github.com/git/git.git|*
git://github.com/git/git.git|*
`[1:]

var digitRe = regexp.MustCompile(`\d`)
var numAlphaRe = regexp.MustCompile(`\d+[[:alpha:]]+`)

const tagPrefix = "refs/tags/"
const headsPrefix = "refs/heads/"

// refs/tags/n3.1-dev^{}
// refs/tags/n3.1.1
func filterTag(tag string) string {
	if strings.HasSuffix(tag, "{}") {
		return ""
	}

	parts := strings.Split(tag, "/")
	last := parts[len(parts)-1]
	loc := digitRe.FindStringIndex(last)
	if loc == nil {
		return ""
	}
	ver := last[loc[0]:]

	return ver
}

// New git filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix == Name || strings.HasSuffix(arg, ".git") {
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

func (f gitFilter) Filter(ps pair.Slice) (pair.Slice, error) {
	refPairs, err := gitrefs.Refs(f.repo, gitrefs.AllProtos)
	if err != nil {
		return nil, err
	}

	ps = append(pair.Slice{}, ps...)
	for _, p := range refPairs {
		switch {
		case strings.HasPrefix(p.Name, tagPrefix):
			name := filterTag(p.Name)
			if name == "" {
				continue
			}
			ps = append(ps, pair.Pair{Name: name, Value: p.ObjID})
		case strings.HasPrefix(p.Name, headsPrefix):
			ps = append(ps, pair.Pair{Name: p.Name[len(headsPrefix):], Value: p.ObjID})
		default:
			continue
		}
	}

	return ps, nil
}
