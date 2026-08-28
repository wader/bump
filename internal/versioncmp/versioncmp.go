package versioncmp

import (
	"strconv"
	"unicode"
)

func Split(a string) []any {
	if len(a) == 0 {
		return nil
	}

	lastIsDigit := unicode.IsDigit(rune(a[0]))
	lastIndex := 0
	var parts []any

	add := func(isNumber bool, s string) {
		if isNumber {
			n, _ := strconv.ParseInt(s, 10, 64)
			parts = append(parts, n)
		} else {
			parts = append(parts, s)
		}
	}

	for i, r := range a[1:] {
		isDigit := unicode.IsDigit(r)
		if isDigit != lastIsDigit {
			add(lastIsDigit, a[lastIndex:i+1])
			lastIsDigit = isDigit
			lastIndex = i + 1
			continue
		}
	}

	if lastIndex != len(a) {
		add(lastIsDigit, a[lastIndex:])
	}

	return parts
}

func Cmp(a, b string) bool {
	ap := Split(a)
	bp := Split(b)
	for i := 0; i < len(ap) && i < len(bp); i++ {
		ae := ap[i]
		be := bp[i]

		switch ae := ae.(type) {
		case int64:
			switch be := be.(type) {
			case int64:
				if ae == be {
					continue
				}
				return ae < be
			default:
				return true
			}
		case string:
			switch be := be.(type) {
			case string:
				if ae == be {
					continue
				}
				return ae < be
			default:
				return false
			}
		}
	}

	return len(ap) <= len(bp)
}
