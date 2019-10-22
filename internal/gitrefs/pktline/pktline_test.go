package pktline

import (
	"bytes"
	"testing"
)

func Test(t *testing.T) {
	testCases := []struct {
		pktLine []byte
		line    string
	}{
		{
			[]byte("001e# service=git-upload-pack\n"),
			"# service=git-upload-pack\n",
		},
		{
			[]byte("004895dcfa3633004da0049d3d0fa03f80589cbcaf31 refs/heads/maint\x00multi_ack\n"),
			"95dcfa3633004da0049d3d0fa03f80589cbcaf31 refs/heads/maint\x00multi_ack\n",
		},
		{
			[]byte("003fd049f6c27a2244e12041955e262a404c7faba355 refs/heads/master\n"),
			"d049f6c27a2244e12041955e262a404c7faba355 refs/heads/master\n",
		},
		{
			[]byte("003c2cb58b79488a98d2721cea644875a8dd0026b115 refs/tags/v1.0\n"),
			"2cb58b79488a98d2721cea644875a8dd0026b115 refs/tags/v1.0\n",
		},
		{
			[]byte("003fa3c2e2402b99163d1d59756e5f207ae21cccba4c refs/tags/v1.0^{}\n"),
			"a3c2e2402b99163d1d59756e5f207ae21cccba4c refs/tags/v1.0^{}\n",
		},
		{
			[]byte("0000"),
			"",
		},
	}
	for _, tC := range testCases {
		tC := tC
		t.Run(tC.line, func(t *testing.T) {
			actualLine, err := Read(bytes.NewReader(tC.pktLine))
			if err != nil {
				t.Error(err)
			} else if tC.line != actualLine {
				t.Errorf("expected %q got %q", tC.line, actualLine)
			}

			b := &bytes.Buffer{}
			_, err = Write(b, tC.line)
			if err != nil {
				t.Error(err)
			} else if bytes.Compare(tC.pktLine, b.Bytes()) != 0 {
				t.Errorf("expected %q got %q", tC.pktLine, b.Bytes())
			}
		})
	}
}
