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

type ScanFn func(s string) (string, error)

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

func Scan(s string, fn ScanFn) (int, error) {
	l := &lexer{s: s}
	for {
		t, p, err := l.scan(fn)
		if err != nil || t == "" {
			return p, err
		}
	}
}

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

func Re(re *regexp.Regexp) func(s string) (string, error) {
	start := true
	sb := strings.Builder{}
	return func(c string) (string, error) {
		if start && !re.MatchString(c) {
			return "", fmt.Errorf("expected %s", re)
		}
		start = false
		if re.MatchString(c) {
			sb.WriteString(c)
			return "", nil
		}
		return sb.String(), nil
	}
}

func Rest(min int) func(s string) (string, error) {
	sb := strings.Builder{}
	return func(c string) (string, error) {
		if c == "" {
			if sb.Len() < min {
				return "", fmt.Errorf("expected more characters")
			}
			return sb.String(), nil
		}
		sb.WriteString(c)
		return "", nil
	}
}

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
			return "", fmt.Errorf("found no ending %s", q)
		}

		switch state {
		case Start:
			if c != q {
				return "", fmt.Errorf("expected %s", q)
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

func Or(fns ...ScanFn) func(s string) (string, error) {
	return func(c string) (string, error) {
		if len(fns) == 0 && c != "" {
			return "", errors.New("no match")
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

func Concat(fns ...ScanFn) func(s string) (string, error) {
	i := 0

	return func(c string) (string, error) {
		if i == len(fns) {
			if c == "" {
				return "", nil
			}
			return "", fmt.Errorf("unexpected %s", c)
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
