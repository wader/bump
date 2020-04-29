package rereplacer_test

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/wader/bump/internal/rereplacer"
)

func TestReplace(t *testing.T) {
	testCases := []struct {
		s        []byte
		replacer []rereplacer.Replace
		expected []byte
	}{
		{
			s: []byte(`abc`),
			replacer: []rereplacer.Replace{
				{Fn: func(b []byte, sm []int) []byte { return b[sm[0]:sm[1]] }, Re: regexp.MustCompile(`.*`)},
			},
			expected: []byte(`abc`),
		},
		{
			s: []byte(`abc`),
			replacer: []rereplacer.Replace{
				{Fn: func(b []byte, sm []int) []byte { return b[sm[2]:sm[3]] }, Re: regexp.MustCompile(`.*(b).*`)},
			},
			expected: []byte(`b`),
		},
		{
			s: []byte(`abcde`),
			replacer: []rereplacer.Replace{
				{Fn: func(b []byte, sm []int) []byte { return b[sm[2]:sm[5]] }, Re: regexp.MustCompile(`a(b)c(d)e`)},
			},
			expected: []byte(`bcd`),
		},
		{
			s: []byte(`abc`),
			replacer: []rereplacer.Replace{
				{Fn: func(b []byte, sm []int) []byte { return []byte("1a1") }, Re: regexp.MustCompile(`a`)},
				{Fn: func(b []byte, sm []int) []byte { return []byte("2b2") }, Re: regexp.MustCompile(`b`)},
				{Fn: func(b []byte, sm []int) []byte { return []byte("3c3") }, Re: regexp.MustCompile(`c`)},
			},
			expected: []byte(`1a12b23c3`),
		},
		{
			s: []byte(`abc`),
			replacer: []rereplacer.Replace{
				{Fn: func(b []byte, sm []int) []byte { return []byte("1bc") }, Re: regexp.MustCompile(`abc`)},
				{Fn: func(b []byte, sm []int) []byte { return []byte("a2c") }, Re: regexp.MustCompile(`abc`)},
				{Fn: func(b []byte, sm []int) []byte { return []byte("ab3") }, Re: regexp.MustCompile(`abc`)},
			},
			expected: []byte(`123`),
		},
		{
			s: []byte(`aabbcc`),
			replacer: []rereplacer.Replace{
				{Fn: func(b []byte, sm []int) []byte { return []byte("aabb33") }, Re: regexp.MustCompile(`aabbcc`)},
				{Fn: func(b []byte, sm []int) []byte { return []byte("aa22cc") }, Re: regexp.MustCompile(`aabbcc`)},
				{Fn: func(b []byte, sm []int) []byte { return []byte("11bbcc") }, Re: regexp.MustCompile(`aabbcc`)},
			},
			expected: []byte(`112233`),
		},
		{
			s: []byte(`aabaa`),
			replacer: []rereplacer.Replace{
				{Fn: func(b []byte, sm []int) []byte { return []byte("aba") }, Re: regexp.MustCompile(`aabaa`)},
			},
			expected: []byte(`aba`),
		},
		{
			s: []byte(`abc`),
			replacer: []rereplacer.Replace{
				{Fn: func(b []byte, sm []int) []byte { return []byte("1") }, Re: regexp.MustCompile(`ab`)},
				{Fn: func(b []byte, sm []int) []byte { return []byte("2") }, Re: regexp.MustCompile(`bc`)},
			},
			expected: []byte(`1c`),
		},
		{
			s: []byte(`abc`),
			replacer: []rereplacer.Replace{
				{Fn: func(b []byte, sm []int) []byte { return []byte("2") }, Re: regexp.MustCompile(`bc`)},
				{Fn: func(b []byte, sm []int) []byte { return []byte("1") }, Re: regexp.MustCompile(`ab`)},
			},
			expected: []byte(`a2`),
		},
		{
			s: []byte(`aabbcc`),
			replacer: []rereplacer.Replace{
				{Fn: func(b []byte, sm []int) []byte { return []byte("aab333") }, Re: regexp.MustCompile(`aabbcc`)},
				{Fn: func(b []byte, sm []int) []byte { return []byte("aa22cc") }, Re: regexp.MustCompile(`aabbcc`)},
				{Fn: func(b []byte, sm []int) []byte { return []byte("111bcc") }, Re: regexp.MustCompile(`aabbcc`)},
			},
			expected: []byte(`aab333`),
		},
	}
	for _, tC := range testCases {
		t.Run(string(tC.s)+" -> "+string(tC.expected), func(t *testing.T) {
			r := rereplacer.Replacer(tC.replacer)
			actual := r.Replace(tC.s)
			if !bytes.Equal(tC.expected, actual) {
				t.Errorf("expected %q got %q", string(tC.expected), string(actual))
			}
		})
	}
}
