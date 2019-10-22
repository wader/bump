package pipeline

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/pair"
)

// Pipeline is a slice of filters
type Pipeline []filter.Filter

var controlCharactersRe = regexp.MustCompile(`[[:cntrl:]]`)

func hasControlCharacters(s string) bool {
	return controlCharactersRe.MatchString(s)
}

// New pipeline
func New(filters []filter.NamedFilter, pipelineStr string) (pipeline Pipeline, err error) {
	var ppl []filter.Filter

	parts := strings.Split(pipelineStr, `|`)

	for i, filterExp := range parts {
		filterExp = strings.TrimSpace(filterExp)
		f, err := filter.New(filters, filterExp)
		if err != nil {
			return nil, err
		}

		// value/@ filter only makes sense to have last
		if _, ok := f.(filter.Valuer); ok && i != len(parts)-1 {
			return nil, fmt.Errorf("value filter must be last")
		}

		ppl = append(ppl, f)
	}

	return Pipeline(ppl), nil
}

func (pl Pipeline) String() string {
	var ss []string
	for _, p := range pl {
		ss = append(ss, p.String())

	}

	return strings.Join(ss, "|")
}

// Run pipeline
func (pl Pipeline) Run(inPp pair.Slice, logFn func(format string, v ...interface{})) (value string, pp pair.Slice, err error) {
	var lastF filter.Filter
	pp = inPp

	for _, f := range pl {
		before := pp
		pp, err = f.Filter(pp)
		if err != nil {
			return "", nil, err
		}

		if logFn != nil {
			after := pair.Slice(pp)
			removed := before.Minus(after)
			added := after.Minus(before)
			if logFn != nil {
				logFn("%s:", f)
				logFn("  > %+v", before)
				logFn("  + %+v", added)
				logFn("  - %+v", removed)
				logFn("  = %+v", pp)
			}
		}

		lastF = f
	}

	if len(pp) == 0 {
		return "", pp, nil
	}

	if hasControlCharacters(pp[0].Name) || hasControlCharacters(pp[0].Value) {

	}

	// if value/@ filter is last return value instead of name
	if _, ok := lastF.(filter.Valuer); ok {
		if hasControlCharacters(pp[0].Value) {
			return "", nil, fmt.Errorf("value contains control characters %q", pp[0].Value)
		}
		return pp[0].Value, pp, nil
	}

	if hasControlCharacters(pp[0].Name) {
		return "", nil, fmt.Errorf("name contains control characters %q", pp[0].Name)
	}

	return pp[0].Name, pp, nil
}

// Value run the pipeline and return one value or error
func (pl Pipeline) Value(logFn func(format string, v ...interface{})) (value string, err error) {
	v, pp, err := pl.Run(nil, logFn)
	if err != nil {
		return "", err
	}

	if len(pp) == 0 {
		return "", fmt.Errorf("no version found")
	}

	return v, err
}
