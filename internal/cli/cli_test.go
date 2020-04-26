package cli_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wader/bump/internal/cli"
)

type testEnv struct {
	LineNr int

	RunArgs []string
	Files   map[string]string

	ExpectedFiles  map[string]string
	ExpectedStdout string
	ExpectedStderr string

	ActualFiles     map[string]string
	ActualStdoutBuf *bytes.Buffer
	ActualStderrBuf *bytes.Buffer
}

func (e *testEnv) Args() []string {
	return e.RunArgs
}

func (e *testEnv) Stdout() io.Writer {
	return e.ActualStdoutBuf
}

func (e *testEnv) Stderr() io.Writer {
	return e.ActualStderrBuf
}

func (e *testEnv) WriteFile(name string, data []byte) error {
	e.ActualFiles[name] = string(data)
	return nil
}

func (e *testEnv) ReadFile(name string) ([]byte, error) {
	if data, ok := e.Files[name]; ok {
		return []byte(data), nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

func (e *testEnv) Glob(pattern string) ([]string, error) {
	var matches []string

	for name := range e.Files {
		ok, err := filepath.Match(pattern, name)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		matches = append(matches, name)
	}

	return matches, nil
}

func testDeepEqual(fn func(format string, args ...interface{}), name string, expected interface{}, actual interface{}) {
	expectedStr := fmt.Sprintf("%#v", expected)
	actualStr := fmt.Sprintf("%#v", actual)
	if !reflect.DeepEqual(expected, actual) {
		diff := ""
		for i := len(diff); i < len(expectedStr) && i < len(actualStr); i++ {
			if expectedStr[i] != actualStr[i] {
				diff += "^"
			} else {
				diff += " "
			}
		}
		fn(`
%s
expected: %s
  actual: %s
    diff: %s`[1:],
			name, expectedStr, actualStr, diff)
	}
}

func errorDeepEqual(t *testing.T, name string, expected interface{}, actual interface{}) {
	testDeepEqual(t.Errorf, name, expected, actual)
}

func fatalDeepEqual(t *testing.T, name string, expected interface{}, actual interface{}) {
	testDeepEqual(t.Fatalf, name, expected, actual)
}

type section struct {
	LineNr int
	Name   string
	Value  string
}

func sectionParser(re *regexp.Regexp, s string) []section {
	var sections []section

	firstMatch := func(ss []string, fn func(s string) bool) string {
		for _, s := range ss {
			if fn(s) {
				return s
			}
		}
		return ""
	}

	const lineDelim = '\n'
	var cs *section
	lineNr := 0
	lines := strings.Split(s, "\n")
	// skip last if empty because of how split works "a\n" -> ["a", ""]
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for _, l := range lines {
		lineNr++

		sm := re.FindStringSubmatch(l)
		if cs == nil || len(sm) > 0 {
			sections = append(sections, section{})
			cs = &sections[len(sections)-1]

			cs.LineNr = lineNr
			cs.Name = firstMatch(sm, func(s string) bool { return len(s) != 0 })
		} else {
			// TODO: use builder somehow if performance is needed
			cs.Value += l + string(lineDelim)
		}

	}

	return sections
}

func TestSectionParser(t *testing.T) {
	actualSections := sectionParser(
		regexp.MustCompile(`^(?:(a:)|(b:))$`),
		`
a:
c
c
b:
a:
c
a:
`[1:])

	expectedSections := []section{
		{LineNr: 1, Name: "a:", Value: "c\nc\n"},
		{LineNr: 4, Name: "b:", Value: ""},
		{LineNr: 5, Name: "a:", Value: "c\n"},
		{LineNr: 7, Name: "a:", Value: ""},
	}

	errorDeepEqual(t, "sections", expectedSections, actualSections)
}

func parseTestEnvs(s string) []testEnv {
	var tes []testEnv

	for _, c := range strings.Split(s, "---\n") {
		te := testEnv{}
		te.Files = map[string]string{}

		te.ActualStdoutBuf = &bytes.Buffer{}
		te.ActualStderrBuf = &bytes.Buffer{}
		te.ActualFiles = map[string]string{}

		te.ExpectedFiles = map[string]string{}

		// match "name:" or "$args" sections
		seenRun := false
		for _, section := range sectionParser(regexp.MustCompile(`^([/>!].*:)|\$.*$`), c) {
			n, v := section.Name, section.Value
			name := n[1 : len(n)-1]

			switch {
			case !seenRun && strings.HasPrefix(n, "/"):
				te.Files[name] = v
			case !seenRun && strings.HasPrefix(n, "$"):
				seenRun = true
				args := strings.Fields(strings.TrimPrefix(n, "$"))
				te.RunArgs = args
			case seenRun && n == ">stdout:":
				te.ExpectedStdout = v
			case seenRun && n == ">stderr:":
				te.ExpectedStderr = v
			case seenRun && strings.HasPrefix(n, "/"):
				te.ExpectedFiles[name] = v
			default:
				panic(fmt.Sprintf("%d: unexpected section %q %q", section.LineNr, n, v))
			}
		}

		tes = append(tes, te)
	}

	return tes
}

func TestParseTestEnv(t *testing.T) {
	actualEnvs := parseTestEnvs(`
/a:
input content a
$ a b
/a:
expected content a
>stdout:
expected stdout
>stderr:
expected stderr
---
/a2:
input content a2
$ a2 b2
/a2:
expected content a2
>stdout:
expected stdout2
>stderr:
expected stderr2
`[1:])

	expectedEnvs := []testEnv{
		{
			RunArgs:         []string{"a", "b"},
			Files:           map[string]string{"a": "input content a\n"},
			ExpectedFiles:   map[string]string{"a": "expected content a\n"},
			ExpectedStdout:  "expected stdout\n",
			ExpectedStderr:  "expected stderr\n",
			ActualStdoutBuf: &bytes.Buffer{},
			ActualStderrBuf: &bytes.Buffer{},
			ActualFiles:     map[string]string{},
		},
		{
			RunArgs:         []string{"a2", "b2"},
			Files:           map[string]string{"a2": "input content a2\n"},
			ExpectedFiles:   map[string]string{"a2": "expected content a2\n"},
			ExpectedStdout:  "expected stdout2\n",
			ExpectedStderr:  "expected stderr2\n",
			ActualStdoutBuf: &bytes.Buffer{},
			ActualStderrBuf: &bytes.Buffer{},
			ActualFiles:     map[string]string{},
		},
	}

	errorDeepEqual(t, "testenv", expectedEnvs, actualEnvs)
}

func testCommandEnv(t *testing.T, te testEnv) {
	cli.Command{Version: "test", Env: &te}.Run()
	errorDeepEqual(t, "files", te.ExpectedFiles, te.ActualFiles)
	errorDeepEqual(t, "stdout", te.ExpectedStdout, te.ActualStdoutBuf.String())
	errorDeepEqual(t, "stderr", te.ExpectedStderr, te.ActualStderrBuf.String())
}

func TestCommand(t *testing.T) {
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
			tes := parseTestEnvs(string(b))
			for _, te := range tes {
				t.Run(strconv.Itoa(te.LineNr), func(t *testing.T) {
					testCommandEnv(t, te)
				})
			}
		})
	}
}
