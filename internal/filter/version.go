package filter

import "strings"

var newlineUnescape = strings.NewReplacer(`\n`, "\n")
var newlineEscape = strings.NewReplacer("\n", `\n`)

// Version is a version with associated values
// Key "name" is the version number "1.2.3" or some something symbolic like "master".
// Other keys can be "commit" etc.
type Version map[string]string

// NewVersionWithName build a new version with name and values
func NewVersionWithName(name string, values map[string]string) Version {
	newValues := map[string]string{}
	for k, v := range values {
		newValues[k] = v
	}
	newValues["name"] = name

	return Version(newValues)
}

// NewVersionFromString build a version from a string
func NewVersionFromString(s string) Version {
	nameValues := strings.SplitN(s, ":", 2)
	name := newlineUnescape.Replace(nameValues[0])
	values := map[string]string{}
	if len(nameValues) > 1 {
		keyValues := strings.Split(nameValues[1], ":")
		for _, keyValues := range keyValues {
			keyValueParts := strings.SplitN(keyValues, "=", 2)
			key := keyValueParts[0]
			value := ""
			if len(keyValueParts) == 2 {
				value = keyValueParts[1]
			}
			values[newlineUnescape.Replace(key)] = newlineUnescape.Replace(value)
		}
	}
	return NewVersionWithName(name, values)
}

func (p Version) String() string {
	var ss = []string{}
	if s, ok := p["name"]; ok {
		ss = append(ss, newlineEscape.Replace(s))
	}
	for k, v := range p {
		if k == "name" {
			continue
		}
		ss = append(ss, newlineEscape.Replace(k)+"="+newlineEscape.Replace(v))
	}
	return strings.Join(ss, ":")
}

// Versions is a slice of versions
type Versions []Version

// NewVersionsFromString build a slice of versions from string
func NewVersionsFromString(s string) Versions {
	if s == "" {
		return nil
	}
	var vs Versions
	for _, sp := range strings.Split(s, ",") {
		vs = append(vs, NewVersionFromString(sp))
	}
	return vs
}

func (vs Versions) String() string {
	var ss []string
	for _, v := range vs {
		ss = append(ss, v.String())
	}
	return strings.Join(ss, ",")
}

// Minus treat versions as set keyed on name and build new set with minus m names
func (vs Versions) Minus(m Versions) Versions {
	ns := map[string]Version{}
	for _, p := range vs {
		ns[p["name"]] = p
	}
	for _, p := range m {
		delete(ns, p["name"])
	}
	var nvs Versions
	for _, v := range ns {
		nvs = append(nvs, v)
	}
	return nvs
}
