package semver

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	mmsemver "github.com/Masterminds/semver/v3"
	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/pair"
)

// Name of filter
const Name = "semver"

// Help text
var Help = `
semver:<constraint>, semver:<n.n.n-pre+build>, <constraint> or <n.n.n-pre+build>

Use [semver](https://semver.org/) to filter or transform versions.

When a constraint is provided it will be used to find the latest version fulfilling
the constraint.

When a verison pattern is provied it will be used to transform a version.

# find latest major 1 version
static:1.1.2,1.1.3,1.2.0|semver:^1
# find latest minor 1.1 version
static:1.1.2,1.1.3,1.2.0|~1.1
# transform into just major.minor
static:1.2.3|n.n
`[1:]

var nRe = regexp.MustCompile("n")

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

func (f semverFilter) Filter(ps pair.Slice) (pair.Slice, error) {
	type vPair struct {
		ver *mmsemver.Version
		p   pair.Pair
	}

	var vps []vPair
	for _, p := range ps {
		ver, err := mmsemver.NewVersion(p.Name)
		// ignore everything that is not valid semver
		if err != nil {
			continue
		}

		if f.template != "" {
			vps = append(vps, vPair{ver: ver, p: pair.Pair{
				Name:  expandTemplate(ver, f.template),
				Value: p.Value,
			}})
		} else {
			vps = append(vps, vPair{ver: ver, p: p})
		}
	}
	sort.Slice(vps, func(i int, j int) bool {
		return vps[i].ver.LessThan(vps[j].ver)
	})

	if f.template != "" {
		var nps pair.Slice
		for _, v := range vps {
			nps = append(nps, v.p)
		}
		return nps, nil
	}

	var latest *vPair
	for i, v := range vps {
		if f.constraint.Check(v.ver) {
			latest = &vps[i]
		}
	}
	if latest == nil {
		return nil, nil
	}

	return pair.Slice{(*latest).p}, nil
}
