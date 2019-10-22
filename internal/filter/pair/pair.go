package pair

import "strings"

// Pair is a version defined as a name (can be non-numeric) and a value
type Pair struct {
	Name  string // "1.2.3", "master" etc
	Value string // Value name is pointing to, commit hash etc
}

var newlineUnescape = strings.NewReplacer(`\n`, "\n")
var newlineEscape = strings.NewReplacer("\n", `\n`)

// FromString build a pair from a string
func FromString(s string) Pair {
	nameValue := strings.SplitN(s, ":", 2)
	name := newlineUnescape.Replace(nameValue[0])
	value := ""
	if len(nameValue) > 1 {
		value = newlineUnescape.Replace(nameValue[1])
	}
	return Pair{Name: name, Value: value}
}

func (p Pair) String() string {
	var ps = []string{newlineEscape.Replace(p.Name)}
	if p.Value != "" {
		ps = append(ps, newlineEscape.Replace(p.Value))
	}
	return strings.Join(ps, ":")
}

// Slice of pairs
type Slice []Pair

// SliceFromString build a slice of pairs from string
func SliceFromString(s string) Slice {
	if s == "" {
		return nil
	}
	var ps Slice
	for _, sp := range strings.Split(s, ",") {
		ps = append(ps, FromString(sp))
	}
	return Slice(ps)
}

func (ps Slice) String() string {
	var ss []string
	for _, p := range ps {
		ss = append(ss, p.String())
	}
	return strings.Join(ss, ",")
}

// Minus treat slices as pair sets and build new set with ps names minus m names
func (ps Slice) Minus(m Slice) Slice {
	nm := map[string]Pair{}
	for _, p := range ps {
		nm[p.Name] = p
	}
	for _, p := range m {
		delete(nm, p.Name)
	}
	var n Slice
	for _, p := range nm {
		n = append(n, p)
	}
	return Slice(n)
}
