package value

import (
	"fmt"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/pair"
)

// Name of filter
const Name = "value"

// Help text
var Help = `
value or @

Swap name and value for each pair. Useful to have last in a pipeline
to use git hash instead of tag name etc.

static:a:1|@
static:a:1|value
`[1:]

// New value filter
// Used to swap name and value for each pair.
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix != Name && prefix != "@" {
		return nil, nil
	}
	if arg != "" {
		return nil, fmt.Errorf("arg should be empty")
	}
	return valueFilter{}, nil
}

type valueFilter struct{}

func (f valueFilter) String() string {
	return Name
}

func (f valueFilter) Filter(ps pair.Slice) (pair.Slice, error) {
	var sps pair.Slice
	for _, p := range ps {
		sps = append(sps, pair.Pair{Name: p.Value, Value: p.Name})
	}
	return sps, nil
}
