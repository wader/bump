package filter

import (
	"errors"
	"regexp"
	"strings"

	"github.com/wader/bump/internal/filter/pair"
)

// ErrNoFilterMatching no filter matches filter expression
var ErrNoFilterMatching = errors.New("no filter matches")

// Filter filters version pairs
type Filter interface {
	String() string
	Filter(ps pair.Slice) (pair.Slice, error)
}

// NewFilterFn function used to create a new filter
type NewFilterFn func(prefix string, arg string) (Filter, error)

// NamedFilter is a struct with filter name, help and new function
type NamedFilter struct {
	Name  string
	Help  string
	NewFn NewFilterFn
}

var filterNameArgRe = regexp.MustCompile(`^(\w+):(.*)$`)

// New creates a new filter from expression based on list of filter create functions
func New(filters []NamedFilter, filterExp string) (Filter, error) {
	nameArgSM := filterNameArgRe.FindStringSubmatch(filterExp)
	var name, arg string
	if len(nameArgSM) == 3 {
		name = nameArgSM[1]
		arg = nameArgSM[2]
	} else {
		arg = filterExp
	}

	// match name, "name:..." or "name" without args
	for _, nf := range filters {
		if f, err := nf.NewFn(name, arg); err != nil {
			return nil, err
		} else if f != nil {
			return f, nil
		}
	}

	// fuzzy arg as prefix, "@", "sort" etc
	for _, nf := range filters {
		if f, err := nf.NewFn(arg, ""); f != nil {
			return f, err
		}
	}

	// fuzzy "^4", "/re/" etc
	if name == "" {
		for _, nf := range filters {
			if f, err := nf.NewFn("", arg); f != nil {
				return f, err
			}
		}
	}

	return nil, ErrNoFilterMatching
}

// ParseHelp text
func ParseHelp(help string) (syntax []string, description string, examples []string) {
	syntaxSplitRe := regexp.MustCompile(`(, | or )`)
	parts := strings.Split(help, "\n\n")
	syntax = syntaxSplitRe.Split(parts[0], -1)
	description = strings.Join(parts[1:len(parts)-1], "\n\n")
	examples = strings.Split(strings.TrimSpace(parts[len(parts)-1]), "\n")

	return syntax, description, examples
}
