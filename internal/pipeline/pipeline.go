package pipeline

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wader/bump/internal/filter"
)

// DefaultVersionKey is the default start key for pipelines
const DefaultVersionKey = "name"

// Pipeline is a slice of filters
type Pipeline []filter.Filter

var cntrlRe = regexp.MustCompile(`[[:cntrl:]]`)

func hasControlCharacters(s string) bool {
	return cntrlRe.MatchString(s)
}

// New pipeline
func New(filters []filter.NamedFilter, pipelineStr string) (pipeline Pipeline, err error) {
	var ppl []filter.Filter

	parts := strings.Split(pipelineStr, `|`)

	for _, filterExp := range parts {
		filterExp = strings.TrimSpace(filterExp)
		f, err := filter.NewFilter(filters, filterExp)
		if err != nil {
			return nil, err
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
func (pl Pipeline) Run(inVersionKey string, inVersions filter.Versions, logFn func(format string, v ...any)) (outValue string, outVersions filter.Versions, err error) {
	vs := inVersions
	versionKey := inVersionKey

	for _, f := range pl {
		beforeVersionKey := versionKey
		vs, versionKey, err = f.Filter(vs, versionKey)
		if err != nil {
			return "", nil, err
		}

		if logFn != nil {
			if logFn != nil {
				logFn("%s:", f)
				for _, v := range vs {
					logFn("  %v", v)
				}
				if len(vs) == 0 {
					logFn("    (none)")
				}
				logFn("  @ %s -> %s", beforeVersionKey, versionKey)
			}
		}
	}

	if len(vs) == 0 {
		return "", vs, nil
	}

	value := vs[0][versionKey]
	if hasControlCharacters(value) {
		return "", nil, fmt.Errorf("value %q for key %q version %s contains control characters", value, versionKey, vs[0])
	}

	if logFn != nil {
		logFn("  value %s", value)
	}

	return value, vs, nil
}

// Value run the pipeline and return one value or error
func (pl Pipeline) Value(logFn func(format string, v ...any)) (value string, err error) {
	v, pp, err := pl.Run(DefaultVersionKey, nil, logFn)
	if err != nil {
		return "", err
	}

	if len(pp) == 0 {
		return "", fmt.Errorf("no version found")
	}

	return v, err
}
