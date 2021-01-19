package lexer

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type lexer struct {
	s string
	p int
}

// ScanFn scans one character
// s is one char or empty if at end
// returns a string on success, empty if needs more or error on error
type ScanFn func(s string) (string, error)

func charToNice(s string) string {
	switch s {
	case "":
		return "end"
	default:
		return fmt.Sprintf("%q", s)
	}
}

func (l *lexer) scan(fn ScanFn) (string, int, error) {
	slen := len(l.s)

	for ; l.p < slen; l.p++ {
		if t, err := fn(l.s[l.p : l.p+1]); err != nil {
			return "", l.p, err
		} else if t != "" {
			return t, l.p - len(t), nil
		}
	}

	t, err := fn("")
	return t, slen, err
}

// Scan a string
func Scan(s string, fn ScanFn) (int, error) {
	l := &lexer{s: s}
	for {
		t, p, err := l.scan(fn)
		if err != nil || t == "" {
			return p, err
		}
	}
}

// Var names and assigns string on success
func Var(name string, dest *string, fn ScanFn) func(s string) (string, error) {
	return func(c string) (string, error) {
		t, err := fn(c)
		if err != nil {
			return t, fmt.Errorf("%s: %w", name, err)
		}
		if t != "" {
			*dest = t
		}
		return t, err
	}
}

// Re scans using a regexp
func Re(re *regexp.Regexp) func(s string) (string, error) {
	start := true
	sb := strings.Builder{}
	return func(c string) (string, error) {
		if start && !re.MatchString(c) {
			return "", fmt.Errorf("unexpected %s, expected %s", charToNice(c), re)
		}
		start = false
		if re.MatchString(c) {
			sb.WriteString(c)
			return "", nil
		}
		return sb.String(), nil
	}
}

// Rest consumes the rest of the string assert it is at least min length
func Rest(min int) func(s string) (string, error) {
	sb := strings.Builder{}
	return func(c string) (string, error) {
		if c == "" {
			if sb.Len() < min {
				return "", fmt.Errorf("unexpected end")
			}
			return sb.String(), nil
		}
		sb.WriteString(c)
		return "", nil
	}
}

// Quoted scans a quoted string using q as quote character
func Quoted(q string) func(s string) (string, error) {
	const (
		Start = iota
		InRe
		Escape
		End
	)
	state := Start
	sb := strings.Builder{}

	return func(c string) (string, error) {
		if c == "" && state != End {
			return "", fmt.Errorf("found no quote ending")
		}

		switch state {
		case Start:
			if c != q {
				return "", fmt.Errorf("unexpected %s, expected quote start", charToNice(c))
			}
			state = InRe
			return "", nil
		case InRe:
			if c == `\` {
				state = Escape
			} else if c == q {
				state = End
			} else {
				sb.WriteString(c)
			}
			return "", nil
		case Escape:
			if c != q {
				sb.WriteString(`\`)
			}
			sb.WriteString(c)
			state = InRe
			return "", nil
		case End:
			return sb.String(), nil
		}

		return "", errors.New("should not be reached")
	}
}

// Or scans until one succeeds
func Or(fns ...ScanFn) func(s string) (string, error) {
	return func(c string) (string, error) {
		if len(fns) == 0 && c != "" {
			return "", errors.New("found no match")
		}

		var newFns []ScanFn
		for _, fn := range fns {
			if s, err := fn(c); err != nil {
				continue
			} else if s != "" {
				return s, nil
			}
			newFns = append(newFns, fn)
		}
		fns = newFns

		return "", nil
	}
}

// Concat scans all in order
func Concat(fns ...ScanFn) func(s string) (string, error) {
	i := 0

	return func(c string) (string, error) {
		if i == len(fns) {
			if c == "" {
				return "", nil
			}
			return "", fmt.Errorf("unexpected %s", charToNice(c))
		}
		fn := fns[i]

		s, err := fn(c)
		if err != nil {
			return s, err
		} else if s != "" {
			i++
			return s, nil
		}
		return "", nil
	}
}
