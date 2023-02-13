package svn

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/wader/bump/internal/filter"
)

// curl -H "Depth: 1" -X PROPFIND http://svn.code.sf.net/p/lame/svn/tags
/*
<D:multistatus xmlns:D="DAV:">
	<D:response xmlns:S="http://subversion.tigris.org/xmlns/svn/" xmlns:C="http://subversion.tigris.org/xmlns/custom/" xmlns:V="http://subversion.tigris.org/xmlns/dav/" xmlns:lp1="DAV:" xmlns:lp3="http://subversion.tigris.org/xmlns/dav/" xmlns:lp2="http://apache.org/dav/props/">
		<D:href>/p/lame/svn/tags/</D:href>
		...
	</D:response>
	<D:response xmlns:S="http://subversion.tigris.org/xmlns/svn/" xmlns:C="http://subversion.tigris.org/xmlns/custom/" xmlns:V="http://subversion.tigris.org/xmlns/dav/" xmlns:lp1="DAV:" xmlns:lp3="http://subversion.tigris.org/xmlns/dav/" xmlns:lp2="http://apache.org/dav/props/">
		<D:href>/p/lame/svn/tags/RELEASE__3_100/</D:href>
		...
		<D:propstat>
			<lp1:prop>
				<lp1:version-name>6403</lp1:version-name>
				...
			</lp1:prop>
			<D:status>HTTP/1.1 200 OK</D:status>
		</D:propstat>
	</D:response>
</D:multistatus>
*/

// Name of filter
const Name = "svn"

// Help text
var Help = `
svn:<repo>

Produce versions from tags and branches from a subversion repository. Name will
be the tag or branch name, version the revision.

svn:https://svn.apache.org/repos/asf/subversion|*
`[1:]

type multistatus struct {
	Response []struct {
		Href        string `xml:"DAV: href"`
		VersionName string `xml:"DAV: propstat>prop>version-name"`
	} `xml:"DAV: response"`
}

// New svn filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix != Name {
		return nil, nil
	}

	if arg == "" {
		return nil, fmt.Errorf("needs a repo url")
	}

	return svnFilter{repo: arg}, nil
}

type svnFilter struct {
	repo string
}

func (f svnFilter) String() string {
	return Name + ":" + f.repo
}

var elmRE = regexp.MustCompile(`</?[^ >]*?>`)

func (f svnFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	req, err := http.NewRequest("PROPFIND", f.repo+"/tags/", nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Depth", "1")

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer r.Body.Close()

	if r.StatusCode/100 != 2 {
		return nil, "", fmt.Errorf("error response: %s", r.Status)
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, "", err
	}
	// HACK:
	// go 1.20+ encoding/xml don't allow invalid xml with with colon in namespace
	// as we only care about propstat > prop > version-name let's just mangle the exceeding colons for now
	// https://issues.apache.org/jira/browse/SVN-1971
	bodyBytes = elmRE.ReplaceAllFunc(bodyBytes, func(b []byte) []byte {
		colons := 0
		for i, c := range b {
			if c != ':' {
				continue
			}
			colons++
			if colons >= 2 {
				b[i] = '_'
			}
		}
		return b
	})

	var m multistatus
	if err := xml.Unmarshal(bodyBytes, &m); err != nil {
		return nil, "", err
	}

	vs := append(filter.Versions{}, versions...)
	for _, r := range m.Response {
		// ".../svn/tags/a/" -> {..., "svn", "tags", "a", ""}
		parts := strings.Split(r.Href, "/")
		if len(parts) < 3 {
			continue
		}

		parent := parts[len(parts)-3]
		v := parts[len(parts)-2]
		if parent != "tags" {
			continue
		}

		vs = append(vs, filter.NewVersionWithName(v, map[string]string{"version": r.VersionName}))
	}

	return vs, versionKey, nil
}
