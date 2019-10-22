package naivediff

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	testCases := []struct {
		a        string
		b        string
		context  int
		expected string
	}{
		{
			a:       "a",
			b:       "b",
			context: 3,
			expected: `
@@ -1 +1 @@
-a
\ No newline at end of file
+b
\ No newline at end of file
`[1:],
		},
		{
			a:       "a\n",
			b:       "b\n",
			context: 3,
			expected: `
@@ -1 +1 @@
-a
+b
`[1:],
		},
		{
			a:       "a\na",
			b:       "b\na",
			context: 3,
			expected: `
@@ -1,2 +1,2 @@
-a
+b
 a
\ No newline at end of file
`[1:],
		},
		{
			a:       "a\n1\n2\n3\n4",
			b:       "b\n1\n2\n3\n4",
			context: 3,
			expected: `
@@ -1,4 +1,4 @@
-a
+b
 1
 2
 3
`[1:],
		},
		{
			a: `
1
2
3
4a
5
6
7
`[1:],
			b: `
1
2
3
4b
5
6
7
`[1:],
			context: 2,
			expected: `
@@ -2,5 +2,5 @@
 2
 3
-4a
+4b
 5
 6
`[1:],
		},
		{
			a: `
1
2
3
4a
5a
6
7
`[1:],
			b: `
1
2
3
4b
5b
6
7
`[1:],
			context: 2,
			expected: `
@@ -2,6 +2,6 @@
 2
 3
-4a
-5a
+4b
+5b
 6
 7
`[1:],
		},
		{
			a: `
1a
2
3
4
5
6
7a
`[1:],
			b: `
1b
2
3
4
5
6
7b
`[1:],
			context: 2,
			expected: `
@@ -1,3 +1,3 @@
-1a
+1b
 2
 3
@@ -5,3 +5,3 @@
 5
 6
-7a
+7b
`[1:],
		},
		{
			a: `
1
2a
3
4
5
6a
7
`[1:],
			b: `
1
2b
3
4
5
6b
7
`[1:],
			context: 2,
			expected: `
@@ -1,7 +1,7 @@
 1
-2a
+2b
 3
 4
 5
-6a
+6b
 7
`[1:],
		},
	}
	for _, tC := range testCases {
		t.Run(tC.a, func(t *testing.T) {
			aFile, _ := ioutil.TempFile("", "naivediff")
			defer os.Remove(aFile.Name())
			io.Copy(aFile, bytes.NewBufferString(tC.a))
			aFile.Close()
			bFile, _ := ioutil.TempFile("", "naivediff")
			defer os.Remove(bFile.Name())
			io.Copy(bFile, bytes.NewBufferString(tC.b))
			bFile.Close()
			c := exec.Command("diff", "-U", strconv.Itoa(tC.context), aFile.Name(), bFile.Name())
			diffBuf, _ := c.Output()
			realDiff := strings.Join(strings.Split(string(diffBuf), "\n")[2:], "\n")

			actual := Diff(tC.a, tC.b, tC.context)
			if actual != tC.expected {
				t.Errorf("got:\n'%s', expected:\n'%s'", actual, tC.expected)
			}
			if actual != realDiff {
				t.Errorf("got:\n'%s', real diff:\n'%s'", actual, realDiff)
			}
		})
	}
}

func TestPanicOnDifferentNumberOfLines(t *testing.T) {
	defer func() {
		if x := recover(); x == nil {
			t.Error("expected panic")
		}
	}()
	Diff("\n", "", 2)
}
