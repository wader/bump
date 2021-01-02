package deepequal

import (
	"fmt"
	"reflect"

	"github.com/pmezard/go-difflib/difflib"
)

type tf interface {
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

func testDeepEqual(fn func(format string, args ...interface{}), name string, expected interface{}, actual interface{}) {
	expectedStr := fmt.Sprintf("%s", expected)
	actualStr := fmt.Sprintf("%s", actual)

	if !reflect.DeepEqual(expected, actual) {
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(expectedStr),
			B:        difflib.SplitLines(actualStr),
			FromFile: fmt.Sprintf("%s expected", name),
			ToFile:   fmt.Sprintf("%s actual", name),
			Context:  3,
		}
		udiff, err := difflib.GetUnifiedDiffString(diff)
		if err != nil {
			panic(err)
		}
		fn("\n%s", udiff)
	}
}

func Error(t tf, name string, expected interface{}, actual interface{}) {
	testDeepEqual(t.Errorf, name, expected, actual)
}

func Fatal(t tf, name string, expected interface{}, actual interface{}) {
	testDeepEqual(t.Fatalf, name, expected, actual)
}
