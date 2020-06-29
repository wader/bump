package re

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wader/bump/internal/filter"
)

// Name of filter
const Name = "re"

// Help text
var Help = `
re:/<regexp>/, re:/<regexp>/<template>/, /<regexp>/ or /<regexp>/<template>/

An alternative regex/template delimited can specified by changing the first
/ into some other character, for example: re:#regexp#template#.

Filter name using a [golang regexp](https://golang.org/pkg/regexp/syntax/).
If name does not match regexp the version will be skipped.

If only a regexp and no template is provided and no submatches are defined the
name will not be changed.

If submatches are defined a submatch named "name" or "value" will be used as
name and value otherwise first submatch will be used as name.

If a template is defined and no submatches was defined it will be used as a
replacement string. If submatches are defined it will be used as a template
to expand $0, ${1}, $name etc.

A regexp can match many times. Use ^$ anchors or (?m:) to match just one time
or per line.

# just filter
static:a,b|/b/
# simple replace
static:aaa|re:/a/b/
# simple replace with # as delimiter
static:aaa|re:#a#b#
# name as first submatch
static:ab|re:/a(.)/
# multiple submatch replace
static:ab:1|/(.)(.)/${0}$2$1/
# named submatch as name and value
static:ab|re:/(?P<name>.)(?P<value>.)/
static:ab|re:/(?P<name>.)(?P<value>.)/|@value
`[1:]

func parse(delim string, s string) (re *regexp.Regexp, expand string, err error) {
	p := strings.Split(s, delim)
	if len(p) == 3 && p[0] == "" && p[2] == "" {
		// /re/ -> ["", "re", ""]
		re, err := regexp.Compile(p[1])
		if err != nil {
			return nil, "", err
		}
		return re, "", nil
	} else if len(p) == 4 && p[0] == "" && p[3] == "" {
		// /re/expand/ -> ["", "re", "expand", ""]
		re, err := regexp.Compile(p[1])
		if err != nil {
			return nil, "", err
		}
		return re, p[2], nil
	} else {
		return nil, "", fmt.Errorf("should be /re/ or /re/template/")
	}
}

// New re regular expression match/replace filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix != Name && prefix != "" {
		return nil, nil
	}

	delim := "/"
	if prefix == Name && len(arg) > 0 {
		delim = arg[0:1]
	}

	re, expand, err := parse(delim, arg)
	if err != nil {
		if prefix == "" {
			return nil, nil
		}
		return nil, err
	}

	return reFilter{re: re, delim: delim, expand: expand}, nil
}

type reFilter struct {
	re     *regexp.Regexp
	delim  string
	expand string
}

func (f reFilter) String() string {
	ss := []string{f.re.String()}
	if f.expand != "" {
		ss = append(ss, f.expand)
	}

	return Name + ":" + f.delim + strings.Join(ss, f.delim) + f.delim
}

func (f reFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	subexpNames := f.re.SubexpNames()

	var filtered filter.Versions
	if f.re.NumSubexp() == 0 {
		for _, v := range versions {
			value, ok := v[versionKey]
			if !ok {
				return nil, "", fmt.Errorf("key %q is missing for %s", versionKey, v)
			}
			if !f.re.MatchString(value) {
				continue
			}

			if f.expand == "" {
				filtered = append(filtered, v)
			} else {
				filtered = append(filtered,
					filter.NewVersionWithName(
						f.re.ReplaceAllLiteralString(value, f.expand),
						v,
					))
			}
		}
	} else {
		for _, v := range versions {
			value, ok := v[versionKey]
			if !ok {
				return nil, "", fmt.Errorf("key %q is missing for %s", versionKey, v)
			}

			for _, sm := range f.re.FindAllStringSubmatchIndex(value, -1) {
				values := map[string]string{}
				for k, v := range v {
					values[k] = v
				}

				versionKeyFound := false
				for smi := 0; smi < f.re.NumSubexp()+1; smi++ {
					subexpName := subexpNames[smi]
					if subexpName == "" || sm[smi*2] == -1 {
						continue
					}

					if subexpName == versionKey {
						versionKeyFound = true
					}

					values[subexpNames[smi]] = value[sm[smi*2]:sm[smi*2+1]]
				}

				if f.expand != "" {
					values[versionKey] = string(f.re.ExpandString(nil, f.expand, value, sm))
				} else if !versionKeyFound && sm[2] != -1 {
					// TODO: no name subexp, use first?
					values[versionKey] = value[sm[2]:sm[3]]
				}

				filtered = append(filtered, filter.NewVersionWithName(values["name"], values))
			}
		}
	}

	return filtered, versionKey, nil
}
