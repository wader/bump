package static

import (
	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/pair"
)

// Name of filter
const Name = "static"

// Help text
var Help = `
static:<name[:value]>,...

Produce version pairs from filter argument.

static:1,2,3|sort
`[1:]

// New static filter
func New(prefix string, arg string) (_ filter.Filter, err error) {
	if prefix != Name {
		return nil, nil
	}

	return staticFilter(pair.SliceFromString(arg)), nil
}

type staticFilter pair.Slice

func (f staticFilter) String() string {
	return Name + ":" + pair.Slice(f).String()
}

func (f staticFilter) Filter(ps pair.Slice) (pair.Slice, error) {
	ps = append(pair.Slice{}, ps...)
	ps = append(ps, f...)
	return ps, nil
}
