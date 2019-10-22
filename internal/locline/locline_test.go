package locline

import (
	"reflect"
	"testing"
)

func TestLine(t *testing.T) {
	testCases := []struct {
		desc     string
		text     string
		expected []int
	}{
		{
			desc:     "empty",
			text:     "",
			expected: nil,
		},
		{
			desc:     "no new line",
			text:     "a",
			expected: []int{1},
		},
		{
			desc:     "empty line",
			text:     "\n",
			expected: []int{1},
		},
		{
			desc:     "no newline",
			text:     "aaa",
			expected: []int{1, 1, 1},
		},
		{
			desc:     "one line",
			text:     "aaa\n",
			expected: []int{1, 1, 1, 1},
		},
		{
			desc:     "two lines, no ending newline",
			text:     "aaa\nbbb",
			expected: []int{1, 1, 1, 1, 2, 2, 2},
		},
		{
			desc:     "two lines",
			text:     "aaa\nbbb\n",
			expected: []int{1, 1, 1, 1, 2, 2, 2, 2},
		},
		{
			desc:     "one char lines",
			text:     "a\nb\nc\n",
			expected: []int{1, 1, 2, 2, 3, 3},
		},
		{
			desc:     "empty lines",
			text:     "\n\n\n",
			expected: []int{1, 2, 3},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ll := New([]byte(tC.text))

			var actual []int
			for i := 0; i < len(tC.text); i++ {
				actual = append(actual, ll.Line(i))
			}

			if !reflect.DeepEqual(tC.expected, actual) {
				t.Errorf("expected %v, got %v", tC.expected, actual)
			}
		})
	}
}

func TestOutOfBounds(t *testing.T) {
	ll := New([]byte("a"))
	if ll.Line(1) != -1 {
		t.Error("expected 1 to be outside lines")
	}
}
