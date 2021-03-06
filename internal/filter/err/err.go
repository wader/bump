package err

import (
	"errors"

	"github.com/wader/bump/internal/filter"
)

// Name of filter
const Name = "err"

// Help text
var Help = `
err:<error>

Fail with error message. Used for testing.

err:test
`[1:]

// New err filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix != Name {
		return nil, nil
	}
	return errFilter{err: errors.New(arg)}, nil
}

type errFilter struct {
	err error
}

func (f errFilter) String() string {
	return Name + ":" + f.err.Error()
}

func (f errFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	return nil, versionKey, f.err
}
