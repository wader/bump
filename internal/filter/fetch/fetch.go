package fetch

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/pair"
)

// Name of filter
const Name = "fetch"

// Help text
var Help = `
fetch:<url>, <http://> or <https://>

Fetch a URL and produce one version pair with the content as name.

fetch:http://libjpeg.sourceforge.net|/latest release is version (\w+)/
`[1:]

// New http fetch filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	var urlStr string

	if prefix == Name {
		urlStr = arg
	} else if strings.HasPrefix(arg, "//") {
		for _, p := range []string{"http", "https"} {
			if prefix != p {
				continue
			}

			urlStr = prefix + ":" + arg
			break
		}
	} else {
		return nil, nil
	}

	if urlStr == "" {
		if prefix != Name {
			return nil, nil
		}
		return nil, fmt.Errorf("needs a url")
	}

	return fetchFilter{urlStr: urlStr}, nil
}

type fetchFilter struct {
	urlStr string
}

func (f fetchFilter) String() string {
	return Name + ":" + f.urlStr
}

func (f fetchFilter) Filter(ps pair.Slice) (pair.Slice, error) {
	r, err := http.Get(f.urlStr)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode/100 != 2 {
		return nil, fmt.Errorf("error response: %s", r.Status)
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	return pair.Slice{pair.Pair{Name: string(b)}}, nil
}
