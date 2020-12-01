package key

import (
	"fmt"
	"strings"

	"github.com/wader/bump/internal/filter"
)

// Name of filter
const Name = "key"

// Help text
var Help = `
key:<name> or @<name>

Change default key for a pipeline. Useful to have last in a pipeline
to use git commit hash instead of tag name etc or in the middle of
a pipeline if you want to regexp filter on something else than name.

static:1.0:hello=world|@hello
static:1.0:hello=world|@name
static:1.0:hello=world|key:hello
`[1:]

// New key filter
// Used to change default key in a pipeline.
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix != Name && prefix != "" {
		return nil, nil
	}

	if prefix == Name {
		if arg == "" {
			return nil, fmt.Errorf("should be key:<name> or @<name>")
		}
		return valueFilter{key: arg}, nil
	} else if strings.HasPrefix(arg, "@") {
		return valueFilter{key: arg[1:]}, nil
	}

	return nil, nil
}

type valueFilter struct {
	key string
}

func (f valueFilter) String() string {
	return Name + ":" + f.key
}

func (f valueFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	return versions, f.key, nil
}
