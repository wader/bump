package pipeline_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/all"
	"github.com/wader/bump/internal/filter/pair"
	"github.com/wader/bump/internal/pipeline"
)

type testCase struct {
	lineNr              int
	pipelineStr         string
	expectedPipelineStr string
	err                 string
	filterTests         []testFilterCase
}

type testFilterCase struct {
	lineNr        int
	pairs         pair.Slice
	expectedPairs pair.Slice
	err           string
}

func parseTestCase(s string) []testCase {
	const errPrefix = "error:"
	var cases []testCase
	var tc testCase
	lineNr := 0

	for _, l := range strings.Split(s, "\n") {
		lineNr++

		if strings.TrimSpace(l) == "" || strings.HasPrefix(l, "#") {
			continue
		}

		if strings.HasPrefix(l, " ") {
			parts := strings.Split(l, "->")

			pairs := strings.TrimSpace(parts[0])
			result := strings.TrimSpace(parts[1])

			if strings.HasPrefix(result, errPrefix) {
				tc.filterTests = append(tc.filterTests, testFilterCase{
					lineNr: lineNr,
					err:    strings.TrimPrefix(result, errPrefix),
				})
			} else {
				tc.filterTests = append(tc.filterTests, testFilterCase{
					lineNr:        lineNr,
					pairs:         pair.SliceFromString(pairs),
					expectedPairs: pair.SliceFromString(result),
				})
			}
		} else {
			if tc.pipelineStr != "" {
				cases = append(cases, tc)
			}

			parts := strings.Split(l, "->")
			pipelineStr := strings.TrimSpace(parts[0])
			expectedPipelineStr := strings.TrimSpace(parts[1])

			if strings.HasPrefix(expectedPipelineStr, errPrefix) {
				tc = testCase{
					lineNr:      lineNr,
					pipelineStr: pipelineStr,
					err:         strings.TrimPrefix(expectedPipelineStr, errPrefix),
				}
			} else {
				tc = testCase{
					lineNr:              lineNr,
					pipelineStr:         pipelineStr,
					expectedPipelineStr: expectedPipelineStr,
				}
			}
		}
	}

	if tc.pipelineStr != "" {
		cases = append(cases, tc)
	}

	return cases
}

func TestParseTestCase(t *testing.T) {
	actual := parseTestCase(`
# test
expr -> expected
    ->
    a:1 -> a:1
    a,b:2 -> a,b:2
test -> error:test

/re/template/ -> re:/re/template/
re:/re/ -> re:/re/
    -> error:test
`[1:])

	expected := []testCase{
		{
			lineNr:              2,
			pipelineStr:         "expr",
			expectedPipelineStr: "expected",
			filterTests: []testFilterCase{
				{
					lineNr: 3,
				},
				{
					lineNr:        4,
					pairs:         pair.Slice{{Name: "a", Value: "1"}},
					expectedPairs: pair.Slice{{Name: "a", Value: "1"}}},
				{
					lineNr:        5,
					pairs:         pair.Slice{{Name: "a"}, {Name: "b", Value: "2"}},
					expectedPairs: pair.Slice{{Name: "a"}, {Name: "b", Value: "2"}},
				},
			},
		},
		{lineNr: 6, pipelineStr: "test", err: "test"},
		{lineNr: 8, pipelineStr: "/re/template/", expectedPipelineStr: "re:/re/template/"},
		{
			lineNr:              9,
			pipelineStr:         "re:/re/",
			expectedPipelineStr: "re:/re/",
			filterTests: []testFilterCase{
				{
					lineNr: 10,
					err:    "test",
				},
			},
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %#v expected %#v", actual, expected)
	}
}

func testPipelineTestCase(t *testing.T, tcs []testCase) {
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("%d", tc.lineNr), func(t *testing.T) {
			p, err := pipeline.New(all.Filters(), tc.pipelineStr)
			if tc.err != "" {
				if err == nil {
					t.Fatalf("expected error %q got success", tc.err)
				} else if tc.err != err.Error() {
					t.Fatalf("expected error %q got %q", tc.err, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected %v got error %q", tc.expectedPipelineStr, err)
				} else if tc.expectedPipelineStr != p.String() {
					t.Fatalf("expected %q got %q", tc.expectedPipelineStr, p.String())
				}
			}

			for _, ft := range tc.filterTests {
				t.Run(fmt.Sprintf("%d", ft.lineNr), func(t *testing.T) {
					_, actualPp, err := p.Run(ft.pairs, nil)

					if ft.err != "" {
						if err == nil || err.Error() != ft.err {
							t.Fatalf("expected error %q got %q", ft.err, err)
						}
					} else {
						if err != nil {
							t.Fatalf("expected %v got error %q", ft.expectedPairs, err)
						} else if !reflect.DeepEqual(ft.expectedPairs, actualPp) {
							t.Fatalf("expected %v got %v", ft.expectedPairs, actualPp)
						}
					}
				})
			}
		})
	}
}

func TestPipeline(t *testing.T) {
	const testDataDir = "testdata"
	testDataFiles, err := ioutil.ReadDir(testDataDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, fi := range testDataFiles {
		fi := fi
		t.Run(fi.Name(), func(t *testing.T) {
			t.Parallel()
			b, err := ioutil.ReadFile(filepath.Join(testDataDir, fi.Name()))
			if err != nil {
				t.Fatal(err)
			}
			tcs := parseTestCase(string(b))
			testPipelineTestCase(t, tcs)
		})
	}
}

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

func testPipeline(t *testing.T, pipelineStr string) pipeline.Pipeline {
	p, err := pipeline.New(
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
