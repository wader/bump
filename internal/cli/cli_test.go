package cli_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wader/bump/internal/cli"
	"github.com/wader/bump/internal/deepequal"
)

const testCaseDelim = "---\n"

type testShell struct {
	cmd string
	env []string
}

type testCaseFile struct {
	name string
	data string
}

type testCase struct {
	parts []interface{}
}

type testCaseComment string
type testCaseExistingFile testCaseFile
type testCaseExpectedWriteFile testCaseFile
type testCaseArgs string
type testCaseExpectedStdout string
type testCaseExpectedStderr string
type testCaseExpectedShell testShell

func (tc testCase) String() string {
	sb := &strings.Builder{}
	for _, p := range tc.parts {
		switch p := p.(type) {
		case testCaseComment:
			fmt.Fprintf(sb, "#%s\n", p)
		case testCaseExistingFile:
			fmt.Fprintf(sb, "/%s:\n", p.name)
			fmt.Fprint(sb, p.data)
		case testCaseArgs:
			fmt.Fprintf(sb, "$%s\n", p)
		case testCaseExpectedWriteFile:
			fmt.Fprintf(sb, "/%s:\n", p.name)
			fmt.Fprint(sb, p.data)
		case testCaseExpectedStdout:
			fmt.Fprintf(sb, ">stdout:\n")
			fmt.Fprint(sb, p)
		case testCaseExpectedStderr:
			fmt.Fprintf(sb, ">stderr:\n")
			fmt.Fprint(sb, p)
		case testCaseExpectedShell:
			fmt.Fprintf(sb, "!%s\n", p.cmd)
			for _, e := range p.env {
				fmt.Fprintln(sb, e)
			}
		default:
			panic("unreachable")
		}
	}
	return sb.String()
}

type testCaseOS struct {
	tc                 testCase
	actualWrittenFiles []testCaseFile
	actualStdoutBuf    *bytes.Buffer
	actualStderrBuf    *bytes.Buffer
	actualShells       []testShell
}

func (t *testCaseOS) Args() []string {
	for _, p := range t.tc.parts {
		if a, ok := p.(testCaseArgs); ok {
			return strings.Fields(string(a))
		}
	}
	return nil
}
func (t *testCaseOS) Getenv(name string) string { panic("not implemented") }
func (t *testCaseOS) Stdout() io.Writer         { return t.actualStdoutBuf }
func (t *testCaseOS) Stderr() io.Writer         { return t.actualStderrBuf }
func (t *testCaseOS) WriteFile(name string, data []byte) error {
	t.actualWrittenFiles = append(t.actualWrittenFiles, testCaseFile{name: name, data: string(data)})
	return nil
}

func (t *testCaseOS) ReadFile(name string) ([]byte, error) {
	for _, p := range t.tc.parts {
		if f, ok := p.(testCaseExistingFile); ok && f.name == name {
			return []byte(f.data), nil
		}
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

func (t *testCaseOS) Glob(pattern string) ([]string, error) {
	var matches []string
	for _, p := range t.tc.parts {
		if f, ok := p.(testCaseExistingFile); ok {
			ok, err := filepath.Match(pattern, f.name)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			matches = append(matches, f.name)
		}
	}
	return matches, nil
}

func (t *testCaseOS) Shell(cmd string, env []string) error {
	t.actualShells = append(t.actualShells, testShell{
		cmd: cmd,
		env: env,
	})
	return nil
}

func (t *testCaseOS) Exec(args []string, env []string) error { panic("not implemented") }

func (t *testCaseOS) String() string {
	sb := &strings.Builder{}
	for _, p := range t.tc.parts {
		switch p := p.(type) {
		case testCaseComment:
			fmt.Fprintf(sb, "#%s\n", p)
		case testCaseExistingFile:
			fmt.Fprintf(sb, "/%s:\n", p.name)
			fmt.Fprint(sb, p.data)
		case testCaseArgs:
			fmt.Fprintf(sb, "$%s\n", string(p))
			for _, awf := range t.actualWrittenFiles {
				fmt.Fprintf(sb, "/%s:\n", awf.name)
				fmt.Fprint(sb, awf.data)
			}
			if t.actualStdoutBuf.Len() > 0 {
				fmt.Fprintf(sb, ">stdout:\n")
				fmt.Fprint(sb, t.actualStdoutBuf.String())
			}
			if t.actualStderrBuf.Len() > 0 {
				fmt.Fprintf(sb, ">stderr:\n")
				fmt.Fprint(sb, t.actualStderrBuf.String())
			}
			for _, s := range t.actualShells {
				fmt.Fprintf(sb, "!%s\n", s.cmd)
				for _, e := range s.env {
					fmt.Fprintln(sb, e)
				}
			}
		case testCaseExpectedWriteFile,
			testCaseExpectedStdout,
			testCaseExpectedStderr,
			testCaseExpectedShell:
			// nop
		default:
			panic(fmt.Sprintf("unreachable %#+v", p))
		}
	}
	return sb.String()
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
	var tcs []testCase

	for _, c := range strings.Split(s, testCaseDelim) {
		tc := testCase{}
		seenRun := false

		for _, section := range sectionParser(regexp.MustCompile(`^([/>].*:)|^[#\$!].*$`), c) {
			n, v := section.Name, section.Value

			switch {
			case strings.HasPrefix(n, "#"):
				tc.parts = append(tc.parts, testCaseComment(strings.TrimPrefix(n, "#")))
			case !seenRun && strings.HasPrefix(n, "/"):
				name := n[1 : len(n)-1]
				tc.parts = append(tc.parts, testCaseExistingFile{name: name, data: v})
			case !seenRun && strings.HasPrefix(n, "$"):
				seenRun = true
				tc.parts = append(tc.parts, testCaseArgs(strings.TrimPrefix(n, "$")))
			case seenRun && n == ">stdout:":
				tc.parts = append(tc.parts, testCaseExpectedStdout(v))
			case seenRun && n == ">stderr:":
				tc.parts = append(tc.parts, testCaseExpectedStderr(v))
			case seenRun && strings.HasPrefix(n, "/"):
				name := n[1 : len(n)-1]
				tc.parts = append(tc.parts, testCaseExpectedWriteFile{name: name, data: v})
			case seenRun && strings.HasPrefix(n, "!"):
				env := strings.Split(v, "\n")
				env = env[0 : len(env)-1]
				tc.parts = append(tc.parts, testCaseExpectedShell{cmd: strings.TrimPrefix(n[1:], "!"), env: env})
			default:
				panic(fmt.Sprintf("%d: unexpected section %q %q", section.LineNr, n, v))
			}
		}

		tcs = append(tcs, tc)
	}

	return tcs
}

func TestParseTestCase(t *testing.T) {
	testCaseText := `
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
`[1:]

	actualTestCases := parseTestCases(testCaseText)
	var actualTestCasesTexts []string
	for _, tc := range actualTestCases {
		actualTestCasesTexts = append(actualTestCasesTexts, tc.String())
	}

	deepequal.Error(t, "test case", testCaseText, strings.Join(actualTestCasesTexts, testCaseDelim))
}

func TestCommand(t *testing.T) {
	const testDataDir = "testdata"
	testDataFiles, err := os.ReadDir(testDataDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, fi := range testDataFiles {
		fi := fi
		t.Run(fi.Name(), func(t *testing.T) {
			t.Parallel()

			testFilePath := filepath.Join(testDataDir, fi.Name())
			b, err := os.ReadFile(testFilePath)
			if err != nil {
				t.Fatal(err)
			}
			tcs := parseTestCases(string(b))
			var actualTexts []string

			for i, tc := range tcs {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					to := &testCaseOS{
						tc:                 tc,
						actualWrittenFiles: []testCaseFile{},
						actualStdoutBuf:    &bytes.Buffer{},
						actualStderrBuf:    &bytes.Buffer{},
						actualShells:       []testShell{},
					}

					cli.Command{Version: "test", OS: to}.Run()
					deepequal.Error(t, "testcase", tc.String(), to.String())

					actualTexts = append(actualTexts, to.String())
				})
			}

			actualText := strings.Join(actualTexts, testCaseDelim)
			_ = actualText

			if v := os.Getenv("WRITE_ACTUAL"); v != "" {
				if err := os.WriteFile(testFilePath, []byte(actualText), 0644); err != nil {
					t.Error(err)
				}
			}
		})
	}
}
