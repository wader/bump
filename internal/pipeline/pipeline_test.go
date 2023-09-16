package pipeline_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wader/bump/internal/deepequal"
	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/all"
	"github.com/wader/bump/internal/pipeline"
)

type testCase struct {
	lineNr              int
	pipelineStr         string
	expectedPipelineStr string
	expectedErr         string
	testFilterCases     []testFilterCase
}

type testFilterCase struct {
	lineNr           int
	versions         filter.Versions
	expectedVersions filter.Versions
	expectedValue    string
	expectedErr      string
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

			versions := strings.TrimSpace(parts[0])
			result := strings.TrimSpace(parts[1])

			if strings.HasPrefix(result, errPrefix) {
				tc.testFilterCases = append(tc.testFilterCases, testFilterCase{
					lineNr:      lineNr,
					versions:    filter.NewVersionsFromString(versions),
					expectedErr: strings.TrimPrefix(result, errPrefix),
				})
			} else {
				resultParts := strings.SplitN(result, " ", 2)
				value := ""
				if len(resultParts) == 2 {
					value = resultParts[1]
				}

				tc.testFilterCases = append(tc.testFilterCases, testFilterCase{
					lineNr:           lineNr,
					versions:         filter.NewVersionsFromString(versions),
					expectedVersions: filter.NewVersionsFromString(resultParts[0]),
					expectedValue:    value,
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
					expectedErr: strings.TrimPrefix(expectedPipelineStr, errPrefix),
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
    a:key=1 -> a:key=1 value
    a,b:key=2 -> a,b:key=2 value
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
			testFilterCases: []testFilterCase{
				{
					lineNr: 3,
				},
				{
					lineNr:           4,
					versions:         filter.Versions{map[string]string{"name": "a", "key": "1"}},
					expectedVersions: filter.Versions{map[string]string{"name": "a", "key": "1"}},
					expectedValue:    "value",
				},
				{
					lineNr: 5,
					versions: filter.Versions{
						map[string]string{"name": "a"},
						map[string]string{"name": "b", "key": "2"},
					},
					expectedVersions: filter.Versions{
						map[string]string{"name": "a"},
						map[string]string{"name": "b", "key": "2"},
					},
					expectedValue: "value",
				},
			},
		},
		{lineNr: 6, pipelineStr: "test", expectedErr: "test"},
		{lineNr: 8, pipelineStr: "/re/template/", expectedPipelineStr: "re:/re/template/"},
		{
			lineNr:              9,
			pipelineStr:         "re:/re/",
			expectedPipelineStr: "re:/re/",
			testFilterCases: []testFilterCase{
				{
					lineNr:      10,
					expectedErr: "test",
				},
			},
		},
	}

	deepequal.Error(t, "parse", expected, actual)
}

func testPipelineTestCase(t *testing.T, tcs []testCase) {
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("%d", tc.lineNr), func(t *testing.T) {
			tc := tc
			p, err := pipeline.New(all.Filters(), tc.pipelineStr)
			if tc.expectedErr != "" {
				if err == nil {
					t.Fatalf("expected error %q got success", tc.expectedErr)
				} else if tc.expectedErr != err.Error() {
					t.Fatalf("expected error %q got %q", tc.expectedErr, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected %v got error %q", tc.expectedPipelineStr, err)
				} else {
					deepequal.Error(t, "pipeline string", tc.expectedPipelineStr, p.String())
				}
			}

			for _, ft := range tc.testFilterCases {
				t.Run(fmt.Sprintf("%d", ft.lineNr), func(t *testing.T) {
					actualValue, actualVersions, err := p.Run(pipeline.DefaultVersionKey, ft.versions, nil)

					if ft.expectedErr != "" {
						if err == nil {
							t.Fatalf("expected error %q got success", ft.expectedErr)
						} else if err.Error() != ft.expectedErr {
							t.Fatalf("expected error %q got %q", ft.expectedErr, err)
						}
					} else {
						if err != nil {
							t.Fatalf("expected %v got error %q", ft.expectedVersions, err)
						} else {
							deepequal.Error(t, "versions", ft.expectedVersions, actualVersions)
							if ft.expectedValue != actualValue {
								t.Errorf("expected %q, got %q", ft.expectedValue, actualValue)
							}
						}
					}
				})
			}
		})
	}
}

func TestPipeline(t *testing.T) {
	const testDataDir = "testdata"
	testDataFiles, err := os.ReadDir(testDataDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, fi := range testDataFiles {
		fi := fi
		t.Run(fi.Name(), func(t *testing.T) {
			t.Parallel()
			b, err := os.ReadFile(filepath.Join(testDataDir, fi.Name()))
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
	vs   filter.Versions
}

func (t testFilter) String() string {
	return t.name
}

func (t testFilter) Filter(versions filter.Versions, versionKey string) (newVersions filter.Versions, newVersionKey string, err error) {
	return t.vs, versionKey, nil
}

func testPipeline(t *testing.T, pipelineStr string) pipeline.Pipeline {
	p, err := pipeline.New(
		[]filter.NamedFilter{
			{
				Name: "a",
				NewFn: func(prefix string, arg string) (filter.Filter, error) {
					if arg == "a" {
						return testFilter{name: "a", vs: filter.Versions{filter.NewVersionWithName("a", nil)}}, nil
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
	expectedRun := filter.Versions{map[string]string{"name": "a"}}
	expectedValue := "a"
	actualValue, actualRun, runErr := p.Run(pipeline.DefaultVersionKey, nil, nil)

	if runErr != nil {
		t.Fatal(runErr)
	}
	if expectedValue != actualValue {
		t.Errorf("expected value %q got %q", expectedValue, actualValue)
	}
	deepequal.Error(t, "run", expectedRun, actualRun)
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
