package semver

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	mmsemver "github.com/Masterminds/semver/v3"
	"github.com/wader/bump/internal/filter"
)

// Name of filter
const Name = "semver"

// Help text
var Help = `
semver:<constraint>, semver:<n.n.n-pre+build>, <constraint> or <n.n.n-pre+build>

Use [semver](https://semver.org/) to filter or transform versions.

When a constraint is provided it will be used to find the latest version fulfilling
the constraint.

When a version pattern is provided it will be used to transform a version.

# find latest major 1 version
static:1.1.2,1.1.3,1.2.0|semver:^1
# find latest minor 1.1 version
static:1.1.2,1.1.3,1.2.0|~1.1
# transform into just major.minor
static:1.2.3|n.n
`[1:]

var nRe = regexp.MustCompile("n")

// semver package used to allow leading zeroes but got more strict
// so let's regexp to strip them out for now, maybe in the future use
// own or fork of a semver version and constraint package
var findLeadingZeroes = regexp.MustCompile(`(?:^|\.)0+[1-9]`)

func expandTemplate(v *mmsemver.Version, t string) string {
	prerelease := ""
	if v.Prerelease() != "" {
		prerelease = "-" + v.Prerelease()
	}
	build := ""
	if v.Metadata() != "" {
		build = "+" + v.Metadata()
	}

	s := strings.NewReplacer(
		"-pre", prerelease,
		"+build", build,
	).Replace(t)

	i := 0
	m := map[int]uint64{
		0: v.Major(),
		1: v.Minor(),
		2: v.Patch(),
	}
	return nRe.ReplaceAllStringFunc(s, func(s string) string {
		if n, ok := m[i]; ok {
			i++
			return strconv.FormatUint(n, 10)
		}
		return s
	})
}

// New semver filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	var constraint *mmsemver.Constraints

	if prefix != Name && prefix != "" {
		return nil, nil
	}
	if arg == "" {
		return nil, fmt.Errorf("needs a constraint or version pattern argument")
	}

	constraint, err = mmsemver.NewConstraint(arg)
	if prefix == Name {
		if err != nil {
			return semverFilter{template: arg}, nil
		}
		return semverFilter{constraint: constraint, constraintStr: arg}, nil
	}

	if err == nil {
		return semverFilter{constraint: constraint, constraintStr: arg}, nil
	}

	if strings.HasPrefix(arg, "n.n") {
		return semverFilter{template: arg}, nil
	}

	return nil, nil
}

type semverFilter struct {
	constraintStr string
	template      string
	constraint    *mmsemver.Constraints
}

func (f semverFilter) String() string {
	if f.template != "" {
		return Name + ":" + f.template
	}
	return Name + ":" + f.constraintStr
}

func (f semverFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	type semverVersion struct {
		ver *mmsemver.Version
		v   filter.Version
	}

	var svs []semverVersion
	for _, v := range versions {
		verStr := v[versionKey]
		filteredVerStr := findLeadingZeroes.ReplaceAllStringFunc(verStr, func(s string) string {
			s, hasDot := strings.CutPrefix(s, ".")
			s = strings.TrimLeft(s, "0")
			if hasDot {
				return "." + s
			}
			return s
		})
		ver, err := mmsemver.NewVersion(filteredVerStr)
		// ignore everything that is not valid semver
		if err != nil {
			continue
		}

		if f.template != "" {
			svs = append(svs, semverVersion{ver: ver, v: filter.NewVersionWithName(
				expandTemplate(ver, f.template),
				v,
			)})
		} else {
			svs = append(svs, semverVersion{ver: ver, v: v})
		}
	}

	// if template assume input is already sorted etc
	if f.template != "" {
		var vs filter.Versions
		for _, v := range svs {
			vs = append(vs, v.v)
		}
		return vs, versionKey, nil
	}

	sort.Slice(svs, func(i int, j int) bool {
		return svs[i].ver.LessThan(svs[j].ver)
	})

	var latest *semverVersion
	var latestIndex int
	for i, v := range svs {
		if f.constraint.Check(v.ver) {
			if latest == nil || latest.ver.Compare(v.ver) == -1 {
				latest = &svs[i]
				latestIndex = i
				continue
			}

			if len((*latest).v[versionKey]) <= len(v.v[versionKey]) {
				latest = &svs[i]
				latestIndex = i
				continue
			}
		}
	}
	if latest == nil {
		return nil, "", nil
	}

	var latestAndLower filter.Versions
	for i := latestIndex; i >= 0; i-- {
		latestAndLower = append(latestAndLower, svs[i].v)
	}

	return latestAndLower, versionKey, nil
}
