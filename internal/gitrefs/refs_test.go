package gitrefs_test

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/wader/bump/internal/gitrefs"
)

func TestLocalRepo(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "refs-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	runOrFatal := func(arg ...string) string {
		c := exec.Command(arg[0], arg[1:]...)
		c.Dir = tempDir
		b, err := c.Output()
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	runOrFatal("git", "init", ".")

	actualRefs, err := gitrefs.Refs("file://"+tempDir, gitrefs.AllProtos)
	if err != nil {
		t.Fatal(err)
	}
	expectedRefs := []gitrefs.Ref{{Name: "HEAD", ObjID: "refs/heads/master"}}
	if !reflect.DeepEqual(expectedRefs, actualRefs) {
		t.Errorf("expected %v got %v", expectedRefs, actualRefs)
	}

	runOrFatal("git", "config", "user.email", "test")
	runOrFatal("git", "config", "user.name", "test")
	runOrFatal("git", "commit", "--allow-empty", "--author", "test <test@test>", "--message", "test")
	sha := strings.TrimSpace(runOrFatal("git", "rev-parse", "HEAD"))

	actualRefs, err = gitrefs.Refs("file://"+tempDir, gitrefs.AllProtos)
	if err != nil {
		t.Fatal(err)
	}
	expectedRefs = []gitrefs.Ref{
		{Name: "HEAD", ObjID: sha},
		{Name: "refs/heads/master", ObjID: sha},
	}
	if !reflect.DeepEqual(expectedRefs, actualRefs) {
		t.Errorf("expected %v got %v", expectedRefs, actualRefs)
	}
}

func TestRemoteRepos(t *testing.T) {
	for _, rawurl := range []string{
		"https://git.xiph.org/theora.git",
		"git://git.xiph.org:9418/theora.git",
		"git://git.xiph.org/theora.git",
		"https://code.videolan.org/videolan/x264.git",
		"https://github.com/FFmpeg/FFmpeg.git",
		"git://github.com/FFmpeg/FFmpeg.git",
		"https://aomedia.googlesource.com/aom",
	} {
		t.Run(rawurl, func(t *testing.T) {
			rawurl := rawurl
			t.Parallel()
			refs, err := gitrefs.Refs(rawurl, gitrefs.AllProtos)
			if err != nil {
				t.Fatal(err)
			}
			if len(refs) == 0 {
				t.Error("expected repo to have refs")
			}
		})
	}
}

func hereBytes(s string) []byte {
	return []byte(strings.NewReplacer(`\0`, "\x00").Replace(s[1 : len(s)-1]))
}

func TestGitProtocol(t *testing.T) {
	r := bufio.NewReader(bytes.NewBuffer(hereBytes(`
000eversion 1
00887217a7c7e582c46cec22a130adf4b9d7d950fba0 HEAD\0multi_ack thin-pack side-band side-band-64k ofs-delta shallow no-progress include-tag
00441d3fcd5ced445d1abc402225c0b8a1299641f497 refs/heads/integration
003f7217a7c7e582c46cec22a130adf4b9d7d950fba0 refs/heads/master
003cb88d2441cac0977faf98efc80305012112238d9d refs/tags/v0.9
003c525128480b96c89e6418b1e40909bf6c5b2d580f refs/tags/v1.0
003fe92df48743b7bc7d26bcaabfddde0a1e20cae47c refs/tags/v1.0^{}
0000
`)))
	wBuf := &bytes.Buffer{}
	w := bufio.NewWriter(wBuf)
	rw := bufio.NewReadWriter(r, w)

	u, err := url.Parse("git://host/repo.git")
	if err != nil {
		t.Fatal(err)
	}
	actualRefs, err := gitrefs.GITProtocol(u, rw)
	if err != nil {
		t.Fatal(err)
	}
	w.Flush()

	expectedRefs := []gitrefs.Ref{
		{Name: "HEAD", ObjID: "7217a7c7e582c46cec22a130adf4b9d7d950fba0"},
		{Name: "refs/heads/integration", ObjID: "1d3fcd5ced445d1abc402225c0b8a1299641f497"},
		{Name: "refs/heads/master", ObjID: "7217a7c7e582c46cec22a130adf4b9d7d950fba0"},
		{Name: "refs/tags/v0.9", ObjID: "b88d2441cac0977faf98efc80305012112238d9d"},
		{Name: "refs/tags/v1.0", ObjID: "525128480b96c89e6418b1e40909bf6c5b2d580f"},
		{Name: "refs/tags/v1.0^{}", ObjID: "e92df48743b7bc7d26bcaabfddde0a1e20cae47c"},
	}

	if !reflect.DeepEqual(expectedRefs, actualRefs) {
		t.Errorf("expected %v got %v", expectedRefs, actualRefs)
	}

	actualCommand := wBuf.String()
	expectedCommand := "0033git-upload-pack /repo.git\x00host=host\x00\x00version=1\x00"
	if expectedCommand != actualCommand {
		t.Errorf("expected %v got %v", expectedCommand, actualCommand)
	}
}

func TestHTTPSmartProtocol(t *testing.T) {
	r := bytes.NewBuffer(hereBytes(`
001e# service=git-upload-pack
0000004895dcfa3633004da0049d3d0fa03f80589cbcaf31 refs/heads/maint\0multi_ack
003fd049f6c27a2244e12041955e262a404c7faba355 refs/heads/master
003c2cb58b79488a98d2721cea644875a8dd0026b115 refs/tags/v1.0
003fa3c2e2402b99163d1d59756e5f207ae21cccba4c refs/tags/v1.0^{}
0000
`))

	actualRefs, err := gitrefs.HTTPSmartProtocol(r)
	if err != nil {
		t.Fatal(err)
	}

	expectedRefs := []gitrefs.Ref{
		{Name: "refs/heads/maint", ObjID: "95dcfa3633004da0049d3d0fa03f80589cbcaf31"},
		{Name: "refs/heads/master", ObjID: "d049f6c27a2244e12041955e262a404c7faba355"},
		{Name: "refs/tags/v1.0", ObjID: "2cb58b79488a98d2721cea644875a8dd0026b115"},
		{Name: "refs/tags/v1.0^{}", ObjID: "a3c2e2402b99163d1d59756e5f207ae21cccba4c"},
	}

	if !reflect.DeepEqual(expectedRefs, actualRefs) {
		t.Errorf("expected %v got %v", expectedRefs, actualRefs)
	}
}

func TestHTTPDumbProtocol(t *testing.T) {
	r := bytes.NewBuffer(hereBytes(`
95dcfa3633004da0049d3d0fa03f80589cbcaf31	refs/heads/maint
d049f6c27a2244e12041955e262a404c7faba355	refs/heads/master
2cb58b79488a98d2721cea644875a8dd0026b115	refs/tags/v1.0
a3c2e2402b99163d1d59756e5f207ae21cccba4c	refs/tags/v1.0^{}
`))

	actualRefs, err := gitrefs.HTTPDumbProtocol(r)
	if err != nil {
		t.Fatal(err)
	}

	expectedRefs := []gitrefs.Ref{
		{Name: "refs/heads/maint", ObjID: "95dcfa3633004da0049d3d0fa03f80589cbcaf31"},
		{Name: "refs/heads/master", ObjID: "d049f6c27a2244e12041955e262a404c7faba355"},
		{Name: "refs/tags/v1.0", ObjID: "2cb58b79488a98d2721cea644875a8dd0026b115"},
		{Name: "refs/tags/v1.0^{}", ObjID: "a3c2e2402b99163d1d59756e5f207ae21cccba4c"},
	}

	if !reflect.DeepEqual(expectedRefs, actualRefs) {
		t.Errorf("expected %v got %v", expectedRefs, actualRefs)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (r roundTripFunc) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return r(req)
}

func TestHTTPClient(t *testing.T) {
	roundTripCalled := false
	hp := &gitrefs.HTTPProto{Client: &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (resp *http.Response, err error) {
			roundTripCalled = true
			return &http.Response{StatusCode: 200, Body: ioutil.NopCloser(&bytes.Buffer{})}, nil
		}),
	}}
	u, _ := url.Parse("http://test/repo.git")
	_, _ = hp.Refs(u)
	if !roundTripCalled {
		t.Error("expected custom client RoundTrip to be called")
	}
}
