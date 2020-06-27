package filter_test

import (
	"reflect"
	"testing"

	"github.com/wader/bump/internal/filter"
)

func TestTestFromString(t *testing.T) {
	testCases := []struct {
		s        string
		expected filter.Version
	}{
		{s: "a", expected: map[string]string{"name": "a"}},
		{s: "a:b=", expected: map[string]string{"name": "a", "b": ""}},
		{s: "a:b=1", expected: map[string]string{"name": "a", "b": "1"}},
	}
	for _, tC := range testCases {
		t.Run(tC.s, func(t *testing.T) {
			actual := filter.NewVersionFromString(tC.s)
			if !reflect.DeepEqual(tC.expected, actual) {
				t.Errorf("expected %v, got %v", tC.expected, actual)
			}
			actualString := actual.String()
			if tC.s != actualString {
				t.Errorf("expected %v, got %v", tC.s, actualString)
			}
		})
	}
}
