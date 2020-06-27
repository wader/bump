package sort

import (
	"fmt"
	"sort"

	"github.com/wader/bump/internal/filter"
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

func (f sortFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	svs := append(filter.Versions{}, versions...)
	sort.Slice(svs, func(i int, j int) bool {
		return svs[i][versionKey] > svs[j][versionKey]
	})
	return svs, versionKey, nil
}
