package deepequal_test

import (
	"fmt"
	"testing"

	"github.com/wader/bump/internal/deepequal"
)

type tfFn func(format string, args ...any)

func (fn tfFn) Errorf(format string, args ...any) {
	fn(format, args...)
}

func (fn tfFn) Fatalf(format string, args ...any) {
	fn(format, args...)
}

func TestError(t *testing.T) {
	deepequal.Error(
		tfFn(func(format string, args ...any) {
			expected := `
--- name expected
+++ name actual
@@ -1 +1 @@
-aaaaaaaaa
+aaaaaabba
`
			actual := fmt.Sprintf(format, args...)
			if expected != actual {
				t.Errorf("expected %s, got %s", expected, actual)
			}
		}),
		"name",
		"aaaaaaaaa", "aaaaaabba",
	)
}
