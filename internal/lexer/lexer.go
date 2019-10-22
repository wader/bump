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

type Token struct {
	Name string
	Dest *string
	Fn   ScanFn
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

func Tokenize(s string, ts []Token) (int, error) {
	l := &lexer{s: s}
	for _, t := range ts {
		s, p, err := l.scan(t.Fn)
		if err != nil {
			if t.Name != "" {
				return p, fmt.Errorf("%s: %w", t.Name, err)
			}
			return p, err
		}
		if t.Dest != nil {
			*t.Dest = s
		}
	}
	return 0, nil
}

func Re(re *regexp.Regexp) func(s string) (string, error) {
	start := true
	sb := strings.Builder{}
	return func(s string) (string, error) {
		if start && !re.MatchString(s) {
			return "", fmt.Errorf("expected %s", re)
		}
		start = false
		if re.MatchString(s) {
			sb.WriteString(s)
			return "", nil
		}
		return sb.String(), nil
	}
}

func Rest(min int) func(s string) (string, error) {
	sb := strings.Builder{}
	return func(s string) (string, error) {
		if s == "" {
			if sb.Len() < min {
				return "", fmt.Errorf("expected more characters")
			}
			return sb.String(), nil
		}
		sb.WriteString(s)
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

	return func(s string) (string, error) {
		if s == "" && state != End {
			return "", fmt.Errorf("found no ending %s", q)
		}

		switch state {
		case Start:
			if s != q {
				return "", fmt.Errorf("expected %s", q)
			}
			state = InRe
			return "", nil
		case InRe:
			if s == `\` {
				state = Escape
			} else if s == q {
				state = End
			} else {
				sb.WriteString(s)
			}
			return "", nil
		case Escape:
			if s != q {
				sb.WriteString(`\`)
			}
			sb.WriteString(s)
			state = InRe
			return "", nil
		case End:
			return sb.String(), nil
		}

		return "", errors.New("should not be reached")
	}
}
