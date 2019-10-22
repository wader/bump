package sort

import (
	"fmt"
	"sort"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/pair"
)

// Name of filter
const Name = "sort"

// Help text
var Help = `
sort

Sort versions reverse alphabetically.

static:a,b,c|sort
`[1:]

// New sort filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix != Name {
		return nil, nil
	}
	if arg != "" {
		return nil, fmt.Errorf("arg should be empty")
	}
	return sortFilter{}, nil
}

type sortFilter struct{}

func (f sortFilter) String() string {
	return Name
}

func (f sortFilter) Filter(ps pair.Slice) (pair.Slice, error) {
	sort.Slice(ps, func(i int, j int) bool {
		return ps[i].Name > ps[j].Name
	})
	return ps, nil
}
