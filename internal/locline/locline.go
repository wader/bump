// Package locline is used to translate from location to line number in a text
package locline

import (
	"bytes"
)

// LocLine is type for holding data to translate from location to line
type LocLine [][2]int

// New create a new LocLone for text
func New(text []byte) LocLine {
	endIndex := len(text)
	index := 0
	lastIndex := 0
	var ranges [][2]int

	for {
		l := bytes.IndexByte(text[lastIndex:], "\n"[0])
		if l == -1 {
			break
		}
		index = lastIndex + l + 1

		ranges = append(ranges, [2]int{lastIndex, index})
		lastIndex = index
	}

	if index != endIndex {
		ranges = append(ranges, [2]int{lastIndex, endIndex})
	}

	return LocLine(ranges)
}

// Line for location
func (ll LocLine) Line(loc int) int {
	line := 1
	for _, l := range ll {
		if loc >= l[0] && loc < l[1] {
			return line
		}
		line++
	}

	return -1
}
