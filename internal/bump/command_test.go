package bump

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type testEnv struct {
	InputArgs  []string
	InputFiles map[string]string

	ExpectedFiles  map[string]string
	ExpectedStdout string
	ExpectedStderr string
	ExpectedErrors string

	ActualFiles     map[string]string
	ActualStdoutBuf *bytes.Buffer
	ActualStderrBuf *bytes.Buffer
}

func (e *testEnv) Args() []string {
	return e.InputArgs
}

func (e *testEnv) Stdout() io.Writer {
	return e.ActualStdoutBuf
}

func (e *testEnv) Stderr() io.Writer {
	return e.ActualStderrBuf
}

func (e *testEnv) WriteFile(filename string, data []byte) error {
	e.ActualFiles[filename] = string(data)
	return nil
}

func (e *testEnv) ReadFile(filename string) ([]byte, error) {
	if data, ok := e.InputFiles[filename]; ok {
		return []byte(data), nil
	}
	return nil, os.ErrNotExist
}

func parseTestEnv(s string) *testEnv {
	te := &testEnv{}

	te.InputFiles = map[string]string{}

	te.ActualStdoutBuf = &bytes.Buffer{}
	te.ActualStderrBuf = &bytes.Buffer{}
	te.ActualFiles = map[string]string{}

	te.ExpectedFiles = map[string]string{}

	for _, p := range strings.Split(s, "###") {
		if p == "" {
			continue
		}
		lineAndRest := strings.SplitN(p, "\n", 2)
		line := strings.TrimSpace(lineAndRest[0])
		rest := lineAndRest[1]

		lineArgs := strings.Fields(line)

		switch lineArgs[0] {
		case "args":
			te.InputArgs = lineArgs[1:]
		case "input":
			te.InputFiles[lineArgs[1]] = rest
		case "expected":
			te.ExpectedFiles[lineArgs[1]] = rest
		case "stdout":
			te.ExpectedStdout = rest
		case "stderr":
			te.ExpectedStderr = rest
		case "errors":
			te.ExpectedErrors = rest
		}
	}

	return te
}

func TestParseTestEnv(t *testing.T) {
	te := parseTestEnv(`
### args a b
### input a
input content a
### expected a
expected content a
### errors
expected errors
### stdout
expected stdout
### stderr
expected stderr
`[1:])

	expectedEnv := &testEnv{
		InputArgs:       []string{"a", "b"},
		InputFiles:      map[string]string{"a": "input content a\n"},
		ExpectedFiles:   map[string]string{"a": "expected content a\n"},
		ExpectedStdout:  "expected stdout\n",
		ExpectedStderr:  "expected stderr\n",
		ExpectedErrors:  "expected errors\n",
		ActualStdoutBuf: &bytes.Buffer{},
		ActualStderrBuf: &bytes.Buffer{},
		ActualFiles:     map[string]string{},
	}

	if !reflect.DeepEqual(expectedEnv, te) {
		t.Errorf("expected:\n%#v\ngot:\n%#v\n", expectedEnv, te)
	}
}

func testCommandEnv(t *testing.T, te *testEnv) {
	actualErrors := Command{Version: "test", Env: te}.Run()
	actualErrorsStr := ""
	for _, err := range actualErrors {
		actualErrorsStr += err.Error() + "\n"
	}

	if !reflect.DeepEqual(te.ExpectedFiles, te.ActualFiles) {
		t.Errorf("expected files expected %#v got %#v", te.ExpectedFiles, te.ActualFiles)
	}
	if te.ExpectedStdout != te.ActualStdoutBuf.String() {
		t.Errorf("stdout expected:\n'%s' got:\n'%s'", te.ExpectedStdout, te.ActualStdoutBuf.String())
	}
	if te.ExpectedStderr != te.ActualStderrBuf.String() {
		t.Errorf("stderr expected:\n'%s' got:\n'%s'", te.ExpectedStderr, te.ActualStderrBuf.String())
	}
	if te.ExpectedErrors != actualErrorsStr {
		t.Errorf("errors expected %q got %q", te.ExpectedErrors, actualErrorsStr)
	}
}

func TestCommand(t *testing.T) {
	const testDataDir = "testdata/command"
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
			te := parseTestEnv(string(b))
			testCommandEnv(t, te)
		})
	}
}
