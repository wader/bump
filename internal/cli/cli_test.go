package cli_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wader/bump/internal/cli"
	"github.com/wader/bump/internal/deepequal"
)

type testShell struct {
	Cmd string
	Env []string
}

type testCase struct {
	LineNr int

	RunArgs []string
	Files   map[string]string

	ExpectedFiles  map[string]string
	ExpectedStdout string
	ExpectedStderr string
	ExpectedShells []testShell

	ActualFiles     map[string]string
	ActualStdoutBuf *bytes.Buffer
	ActualStderrBuf *bytes.Buffer
	ActualShells    []testShell
}

func (tc *testCase) Args() []string {
	return tc.RunArgs
}

func (e *testCase) Getenv(name string) string {
	panic("not implemented")
}

func (e *testCase) Stdout() io.Writer {
	return e.ActualStdoutBuf
}

func (e *testCase) Stderr() io.Writer {
	return e.ActualStderrBuf
}

func (e *testCase) WriteFile(name string, data []byte) error {
	e.ActualFiles[name] = string(data)
	return nil
}

func (e *testCase) ReadFile(name string) ([]byte, error) {
	if data, ok := e.Files[name]; ok {
		return []byte(data), nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

func (e *testCase) Glob(pattern string) ([]string, error) {
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

func (e *testCase) Shell(cmd string, env []string) error {
	e.ActualShells = append(e.ActualShells, testShell{
		Cmd: cmd,
		Env: env,
	})
	return nil
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

	const lineDelim = "\n"
	var cs *section
	lineNr := 0
	lines := strings.Split(s, lineDelim)
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
			cs.Value += l + lineDelim
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

	deepequal.Error(t, "sections", expectedSections, actualSections)
}

func parseTestCases(s string) []testCase {
	var tes []testCase

	for _, c := range strings.Split(s, "---\n") {
		te := testCase{}
		te.Files = map[string]string{}

		te.ActualStdoutBuf = &bytes.Buffer{}
		te.ActualStderrBuf = &bytes.Buffer{}
		te.ActualFiles = map[string]string{}
		te.ExpectedFiles = map[string]string{}

		seenRun := false
		// NOTE: !
		for _, section := range sectionParser(regexp.MustCompile(`^([/>].*:)|[#\$!].*$`), c) {
			n, v := section.Name, section.Value
			name := n[1 : len(n)-1]

			switch {
			case strings.HasPrefix(n, "#"):
				continue
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
			case seenRun && strings.HasPrefix(n, "!"):
				env := strings.Split(v, "\n")
				env = env[0 : len(env)-1]
				te.ExpectedShells = append(te.ExpectedShells, testShell{
					Cmd: strings.TrimPrefix(n[1:], "!"),
					Env: env,
				})
			default:
				panic(fmt.Sprintf("%d: unexpected section %q %q", section.LineNr, n, v))
			}
		}

		tes = append(tes, te)
	}

	return tes
}

func TestParseTestCase(t *testing.T) {
	actualTestCase := parseTestCases(`
/a:
input content a
$ a b
/a:
expected content a
>stdout:
expected stdout
>stderr:
expected stderr
!command a b
enva=valuea
envb=valueb
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
!command2 a2 b2
enva2=valuea2
envb2=valueb2
!command22 a22 b22
enva22=valuea22
envb22=valueb22
`[1:])

	expectedTestCase := []testCase{
		{
			RunArgs:        []string{"a", "b"},
			Files:          map[string]string{"a": "input content a\n"},
			ExpectedFiles:  map[string]string{"a": "expected content a\n"},
			ExpectedStdout: "expected stdout\n",
			ExpectedStderr: "expected stderr\n",
			ExpectedShells: []testShell{
				{Cmd: "command a b", Env: []string{"enva=valuea", "envb=valueb"}},
			},
			ActualStdoutBuf: &bytes.Buffer{},
			ActualStderrBuf: &bytes.Buffer{},
			ActualFiles:     map[string]string{},
		},
		{
			RunArgs:        []string{"a2", "b2"},
			Files:          map[string]string{"a2": "input content a2\n"},
			ExpectedFiles:  map[string]string{"a2": "expected content a2\n"},
			ExpectedStdout: "expected stdout2\n",
			ExpectedStderr: "expected stderr2\n",
			ExpectedShells: []testShell{
				{Cmd: "command2 a2 b2", Env: []string{"enva2=valuea2", "envb2=valueb2"}},
				{Cmd: "command22 a22 b22", Env: []string{"enva22=valuea22", "envb22=valueb22"}},
			},
			ActualStdoutBuf: &bytes.Buffer{},
			ActualStderrBuf: &bytes.Buffer{},
			ActualFiles:     map[string]string{},
		},
	}

	deepequal.Error(t, "testcase", expectedTestCase, actualTestCase)
}

func testCommandTestCase(t *testing.T, te testCase) {
	cli.Command{Version: "test", OS: &te}.Run()
	deepequal.Error(t, "files", te.ExpectedFiles, te.ActualFiles)
	deepequal.Error(t, "stdout", te.ExpectedStdout, te.ActualStdoutBuf.String())
	deepequal.Error(t, "stderr", te.ExpectedStderr, te.ActualStderrBuf.String())
	deepequal.Error(t, "shell", te.ExpectedShells, te.ActualShells)
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
			tcs := parseTestCases(string(b))
			for _, tc := range tcs {
				t.Run(strconv.Itoa(tc.LineNr), func(t *testing.T) {
					testCommandTestCase(t, tc)
				})
			}
		})
	}
}
