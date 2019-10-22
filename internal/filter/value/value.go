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

Use value instead of name.

static:a:1|@
static:a:1|value
`[1:]

// New value filter
// Implements Valuer interface. Used last in a pipeline marks that the value
// instead of name should be the result
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
	return ps, nil
}

func (f valueFilter) Value() {}
