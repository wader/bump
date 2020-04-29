// Package rereplacer is similar to strings.Replacer but with regexp
package rereplacer

import (
	"bytes"
	"regexp"
	"sort"
)

// ReplaceFn is a function returning how to replace a match
type ReplaceFn func(b []byte, sm []int) []byte

// Replace is a one replacement
type Replace struct {
	Re *regexp.Regexp
	Fn ReplaceFn
}

// Replacer is multi regex replacer
type Replacer []Replace

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func rangeOverlap(amin, amax, bmin, bmax int) bool {
	return amin < bmax && bmin < amax
}

func commonEnds(a, b []byte) (int, int) {
	minl := min(len(a), len(b))
	var l, r int
	for l = 0; l < minl && a[l] == b[l]; l++ {
	}
	for r = 0; r < minl-l && r < minl-1 && a[len(a)-r-1] == b[len(b)-r-1]; r++ {
	}
	return l, r
}

// Replace return a new copy with replacements performed
func (r Replacer) Replace(s []byte) []byte {
	type edit struct {
		prio int
		re   *regexp.Regexp
		loc  [2]int
		s    []byte
	}
	var edits []edit

	// collect edits to be made
	for replaceI, replace := range r {
		for _, submatchIndexes := range replace.Re.FindAllSubmatchIndex(s, -1) {
			sm := s[submatchIndexes[0]:submatchIndexes[1]]
			r := replace.Fn(s, submatchIndexes)
			if bytes.Equal(sm, r) {
				// same, skip
				continue
			}

			leftCommon, rightCommon := commonEnds(sm, r)
			edits = append(edits, edit{
				prio: replaceI,
				re:   replace.Re,
				loc:  [2]int{submatchIndexes[0] + leftCommon, submatchIndexes[1] - rightCommon},
				s:    r[leftCommon : len(r)-rightCommon],
			})
		}
	}

	// sort by start edit index and prioritized by replacer index on overlap
	sort.Slice(edits, func(i, j int) bool {
		li := edits[i].loc
		lj := edits[j].loc
		if rangeOverlap(li[0], li[1], lj[0], lj[1]) {
			return edits[i].prio < edits[j].prio
		}
		return li[0] < lj[0]
	})

	// build new using edits
	n := &bytes.Buffer{}
	lastIndex := 0
	for _, e := range edits {
		if e.loc[0] < lastIndex {
			// skip one that were not prioritized
			continue
		}
		n.Write(s[lastIndex:e.loc[0]])
		n.Write(e.s)
		lastIndex = e.loc[1]
	}

	endIndex := len(s)
	if lastIndex != endIndex {
		n.Write(s[lastIndex:endIndex])
	}

	return n.Bytes()
}
