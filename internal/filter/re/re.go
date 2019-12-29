package re

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/pair"
)

// Name of filter
const Name = "re"

// Help text
var Help = `
re:/<regexp>/, re:/<regexp>/<template>/, /<regexp>/ or /<regexp>/<template>/

Filter name using a [golang regexp](https://golang.org/pkg/regexp/syntax/).
If name does not match regexp the pair will be skipped.

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
# name as first submatch
static:ab|re:/a(.)/
# multiple submatch replace
static:ab:1|/(.)(.)/${0}$2$1/
# named submatch as name and value
static:ab|re:/(?P<name>.)(?P<value>.)/
static:ab|re:/(?P<name>.)(?P<value>.)/|@
`[1:]

func parse(s string) (re *regexp.Regexp, expand string, err error) {
	// TODO: split respect \ escaping?
	p := strings.Split(s, `/`)
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
	re, expand, err := parse(arg)
	if err != nil {
		if prefix == "" {
			return nil, nil
		}
		return nil, err
	}

	return reFilter{re: re, expand: expand}, nil
}

type reFilter struct {
	re     *regexp.Regexp
	expand string
}

func (f reFilter) String() string {
	ss := []string{f.re.String()}
	if f.expand != "" {
		ss = append(ss, f.expand)
	}

	return Name + ":" + "/" + strings.Join(ss, "/") + "/"
}

func (f reFilter) Filter(ps pair.Slice) (pair.Slice, error) {
	nameIndex := 0
	valueIndex := 0
	for i, n := range f.re.SubexpNames() {
		switch n {
		case "name":
			nameIndex = i
		case "value":
			valueIndex = i
		}
	}

	var filtered pair.Slice
	if f.re.NumSubexp() == 0 {
		for _, p := range ps {
			if !f.re.MatchString(p.Name) {
				continue
			}

			if f.expand == "" {
				filtered = append(filtered, p)
			} else {
				filtered = append(filtered, pair.Pair{
					Name:  f.re.ReplaceAllLiteralString(p.Name, f.expand),
					Value: p.Value,
				})
			}
		}
	} else {
		for _, p := range ps {
			for _, sm := range f.re.FindAllStringSubmatchIndex(p.Name, -1) {
				name := ""
				value := ""

				if f.expand == "" {
					if nameIndex != 0 {
						name = p.Name[sm[nameIndex*2]:sm[nameIndex*2+1]]
					} else {
						// we know there is one subexp
						name = p.Name[sm[2]:sm[3]]
					}
				} else {
					name = string(f.re.ExpandString(nil, f.expand, p.Name, sm))
				}

				if valueIndex != 0 {
					value = p.Name[sm[valueIndex*2]:sm[valueIndex*2+1]]
				} else {
					value = p.Value
				}

				filtered = append(filtered, pair.Pair{Name: name, Value: value})
			}
		}
	}

	return filtered, nil
}
