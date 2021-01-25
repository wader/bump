// Package gitrefs gets refs from a git repo (like git ls-remote)
// https://github.com/git/git/blob/master/Documentation/technical/http-protocol.txt
// https://github.com/git/git/blob/master/Documentation/technical/pack-protocol.txt
package gitrefs

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wader/bump/internal/gitrefs/pktline"
)

const gitPort = 9418

// AllProtos all protocols
// FileProto might be dangerous if you don't control the url
var AllProtos = []Proto{HTTPProto{}, GitProto{}, FileProto{}}

// Ref is name/object id pair
type Ref struct {
	Name  string
	ObjID string
}

// Refs fetches refs for a remote repo (like git ls-remote)
func Refs(rawurl string, protos []Proto) ([]Ref, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	for _, p := range protos {
		refs, err := p.Refs(u)
		if err == nil && refs == nil {
			continue
		}

		return refs, err
	}

	return nil, fmt.Errorf("unknown url: %s", rawurl)
}

// HEAD\0multi_ack thin-pack -> HEAD
// HEAD -> HEAD
func refName(s string) string {
	n := strings.Index(s, "\x00")
	if n == -1 {
		return s
	}
	return s[0:n]
}

// GITProtocol talk native git protocol
// 000eversion 1
// 00887217a7c7e582c46cec22a130adf4b9d7d950fba0 HEAD\0multi_ack thin-pack side-band side-band-64k ofs-delta shallow no-progress include-tag
// 00441d3fcd5ced445d1abc402225c0b8a1299641f497 refs/heads/integration
// 003f7217a7c7e582c46cec22a130adf4b9d7d950fba0 refs/heads/master
// 003cb88d2441cac0977faf98efc80305012112238d9d refs/tags/v0.9
// 003c525128480b96c89e6418b1e40909bf6c5b2d580f refs/tags/v1.0
// 003fe92df48743b7bc7d26bcaabfddde0a1e20cae47c refs/tags/v1.0^{}
// 0000
func GITProtocol(u *url.URL, rw io.ReadWriter) ([]Ref, error) {
	_, err := pktline.Write(rw, fmt.Sprintf("git-upload-pack %s\x00host=%s\x00\x00version=1\x00", u.Path, u.Host))
	if err != nil {
		return nil, err
	}

	var refs []Ref
	for {
		line, err := pktline.Read(rw)
		if err != nil {
			return nil, err
		}
		if line == "" {
			break
		}
		line = strings.TrimSpace(line)

		objIDName := strings.SplitN(line, ` `, 2)
		if len(objIDName) != 2 {
			return nil, fmt.Errorf("unexpected refs line: %s", line)
		}
		objID := objIDName[0]
		name := refName(objIDName[1])

		if objID == "version" {
			continue
		}

		refs = append(refs, Ref{Name: name, ObjID: objID})
	}

	return refs, nil
}

// HTTPSmartProtocol talk git HTTP protocol
// 001e# service=git-upload-pack\n
// 0000
// 004895dcfa3633004da0049d3d0fa03f80589cbcaf31 refs/heads/maint\0multi_ack\n
// 003fd049f6c27a2244e12041955e262a404c7faba355 refs/heads/master\n
// 003c2cb58b79488a98d2721cea644875a8dd0026b115 refs/tags/v1.0\n
// 003fa3c2e2402b99163d1d59756e5f207ae21cccba4c refs/tags/v1.0^{}\n
// 0000
func HTTPSmartProtocol(r io.Reader) ([]Ref, error) {
	// read "# service=git-upload-pack" line
	_, err := pktline.Read(r)
	if err != nil {
		return nil, err
	}

	// read section start
	line, err := pktline.Read(r)
	if err != nil {
		return nil, err
	}
	if line != "" {
		return nil, fmt.Errorf("unexpected section start line: %s", line)
	}

	var refs []Ref
	for {
		line, err := pktline.Read(r)
		if err != nil {
			return nil, err
		}
		if line == "" {
			break
		}
		line = strings.TrimSpace(line)

		objIDName := strings.SplitN(line, " ", 2)
		if len(objIDName) != 2 {
			return nil, fmt.Errorf("unexpected refs line: %s", line)
		}
		objID := objIDName[0]
		name := refName(objIDName[1])

		refs = append(refs, Ref{Name: name, ObjID: objID})
	}

	return refs, nil
}

// HTTPDumbProtocol talk git dump HTTP protocol
// 95dcfa3633004da0049d3d0fa03f80589cbcaf31\trefs/heads/maint\n
// d049f6c27a2244e12041955e262a404c7faba355\trefs/heads/master\n
// 2cb58b79488a98d2721cea644875a8dd0026b115\trefs/tags/v1.0\n
// a3c2e2402b99163d1d59756e5f207ae21cccba4c\trefs/tags/v1.0^{}\n
func HTTPDumbProtocol(r io.Reader) ([]Ref, error) {
	scanner := bufio.NewScanner(r)

	var refs []Ref
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("unexpected refs line: %s", line)
		}
		refs = append(refs, Ref{Name: parts[1], ObjID: parts[0]})
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	return refs, nil
}

// Proto is a git protocol
type Proto interface {
	Refs(u *url.URL) ([]Ref, error)
}

// HTTPProto implements git http protocol
type HTTPProto struct {
	Client *http.Client // http.DefaultClient if nil
}

// Refs from http repo
func (h HTTPProto) Refs(u *url.URL) ([]Ref, error) {
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, nil
	}

	client := h.Client
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequest(http.MethodGet, u.String()+"/info/refs?service=git-upload-pack", nil)
	if err != nil {
		return nil, err
	}
	// some git hosts behave differently based on this, github allows
	// to skip .git if set for example
	req.Header.Set("User-Agent", "git/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") == "application/x-git-upload-pack-advertisement" {
		return HTTPSmartProtocol(resp.Body)
	}
	return HTTPDumbProtocol(resp.Body)
}

// GitProto implements gits own protocol
type GitProto struct{}

// Refs from git repo
func (GitProto) Refs(u *url.URL) ([]Ref, error) {
	if u.Scheme != "git" {
		return nil, nil
	}

	address := u.Host
	if u.Port() == "" {
		address = address + ":" + strconv.Itoa(gitPort)
	}
	n, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	defer n.Close()
	return GITProtocol(u, n)
}

func readSymref(gitPath string, p string) (string, error) {
	fp := filepath.Join(gitPath, p)
	fi, err := os.Stat(fp)
	if err != nil {
		return "", err
	}

	// if symlink try read content of dest file otherwise return just name of dest
	if fi.Mode()&os.ModeSymlink != 0 {
		dst, err := os.Readlink(fp)
		if err != nil {
			return "", err
		}

		b, err := ioutil.ReadFile(filepath.Join(gitPath, dst))
		if err != nil {
			return dst, nil
		}
		return strings.TrimSpace(string(b)), nil
	}

	// if "ref: path" try read content of file otherwise return just name of file
	b, err := ioutil.ReadFile(fp)
	if err != nil {
		return "", nil
	}

	// "ref: path"
	parts := strings.SplitN(strings.TrimSpace(string(b)), ": ", 2)
	if len(parts) != 2 {
		return "", errors.New("unknown ref format")
	}

	dst := parts[1]
	b, err = ioutil.ReadFile(filepath.Join(gitPath, dst))
	if err != nil {
		return dst, nil
	}
	return strings.TrimSpace(string(b)), nil
}

// FileProto implements reading local git repo
type FileProto struct{}

// Refs from file repo
func (f FileProto) Refs(u *url.URL) ([]Ref, error) {
	if u.Scheme != "file" {
		return nil, nil
	}

	// bare or normal repo?
	gitPath := u.Path
	testPath := filepath.Join(u.Path, ".git")
	if fi, err := os.Stat(testPath); err == nil && fi.IsDir() {
		gitPath = testPath
	}

	var refs []Ref

	ref, err := readSymref(gitPath, "HEAD")
	if err != nil {
		return nil, err
	}
	refs = append(refs, Ref{Name: "HEAD", ObjID: ref})

	err = filepath.Walk(
		filepath.Join(gitPath, "refs"),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}

			b, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			refs = append(refs, Ref{
				Name:  strings.TrimPrefix(path, gitPath+string(os.PathSeparator)),
				ObjID: strings.TrimSpace(string(b)),
			})

			return nil
		})
	if err != nil {
		return nil, err
	}

	return refs, nil
}
