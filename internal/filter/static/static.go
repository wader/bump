package static

import (
	"github.com/wader/bump/internal/filter"
)

// Name of filter
const Name = "static"

// Help text
var Help = `
static:<name[:key=value:...]>,...

Produce versions from filter argument.

static:1,2,3,4:key=value:a=b|sort
`[1:]

// New static filter
func New(prefix string, arg string) (_ filter.Filter, err error) {
	if prefix != Name {
		return nil, nil
	}

	return staticFilter(filter.NewVersionsFromString(arg)), nil
}

type staticFilter filter.Versions

func (f staticFilter) String() string {
	return Name + ":" + filter.Versions(f).String()
}

func (f staticFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	vs := append(filter.Versions{}, versions...)
	vs = append(vs, f...)
	return vs, versionKey, nil
}
