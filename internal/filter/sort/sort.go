package sort

import (
	"fmt"
	"sort"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/versioncmp"
)

// Name of filter
const Name = "sort"

// Help text
var Help = `
sort

Sort versions reverse alphabetically.

static:a,b,c|sort
`[1:]

type sortType int

const (
	sortAlphabetical sortType = iota
	sortVersion
)

func (s sortType) String() string {
	switch s {
	case sortAlphabetical:
		return "alphabetical"
	case sortVersion:
		return "version"
	default:
		panic("unreachable")
	}
}

// New sort filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix != Name {
		return nil, nil
	}
	var sortType sortType
	if arg == "" || arg == "alphabetical" {
		sortType = sortAlphabetical
	} else if arg == "version" {
		sortType = sortVersion
	} else {
		return nil, fmt.Errorf("arg should be empty, alphabetical or version")
	}
	return sortFilter{sortType: sortType}, nil
}

type sortFilter struct {
	sortType sortType
}

func (f sortFilter) String() string {
	return Name + ":" + f.sortType.String()
}

func (f sortFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	svs := append(filter.Versions{}, versions...)

	switch f.sortType {
	case sortAlphabetical:
		sort.Slice(svs, func(i int, j int) bool {
			return svs[i][versionKey] < svs[j][versionKey]
		})
	case sortVersion:
		sort.Slice(svs, func(i int, j int) bool {
			return !versioncmp.Cmp(svs[i][versionKey], svs[j][versionKey])
		})
	default:
		panic("unreachable")
	}

	// svs = slicex.Reverse(svs)

	return svs, versionKey, nil
}
