package pipeline

import (
	"reflect"
	"testing"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/pair"
)

type testFilter struct {
	name string
	ps   pair.Slice
}

func (t testFilter) String() string {
	return t.name
}

func (t testFilter) Filter(ps pair.Slice) (pair.Slice, error) {
	return t.ps, nil
}

func testPipeline(t *testing.T, pipelineStr string) Pipeline {
	p, err := New(
		[]filter.NamedFilter{
			{
				Name: "a",
				NewFn: func(prefix string, arg string) (filter.Filter, error) {
					if arg == "a" {
						return testFilter{name: "a", ps: pair.Slice{{Name: "a", Value: "1"}}}, nil
					}
					return nil, nil
				},
			},
		},
		pipelineStr,
	)

	if err != nil {
		t.Fatal(err)
	}

	return p
}

func TestString(t *testing.T) {
	p := testPipeline(t, "a|a")
	expectedString := "a|a"
	actualString := p.String()
	if expectedString != actualString {
		t.Errorf("expected %q got %q", expectedString, actualString)
	}
}

func TestRun(t *testing.T) {
	p := testPipeline(t, "a|a")
	expectedRun := pair.Slice{{Name: "a", Value: "1"}}
	expectedValue := "a"
	actualValue, actualRun, runErr := p.Run(nil, nil)

	if runErr != nil {
		t.Fatal(runErr)
	}
	if expectedValue != actualValue {
		t.Errorf("expected value %q got %q", expectedValue, actualValue)
	}
	if !reflect.DeepEqual(expectedRun, actualRun) {
		t.Errorf("expected %v got %v", expectedRun, actualRun)
	}
}

func TestValue(t *testing.T) {
	p := testPipeline(t, "a|a")
	expectedValue := "a"
	actualValue, errValue := p.Value(nil)

	if errValue != nil {
		t.Fatal(errValue)
	}
	if expectedValue != actualValue {
		t.Errorf("expected value %q got %q", expectedValue, actualValue)
	}
}
