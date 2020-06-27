package docker

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/wader/bump/internal/filter"
)

// Name of filter
const Name = "docker"

// Help text
var Help = `
docker:<image>

Produce versions from a image on ducker hub.

docker:alpine|^3
`[1:]

// TODO: other registry?
const defaultIndex = `https://index.docker.io/v1/repositories/%s/tags`

type tag struct {
	Name string `json:"name"`
}

// New docker filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix != Name {
		return nil, nil
	}
	if arg == "" {
		return nil, fmt.Errorf("needs a image name")
	}

	return dockerFilter{imageName: arg}, nil
}

type dockerFilter struct {
	imageName string
}

func (f dockerFilter) String() string {
	return Name + ":" + f.imageName
}

func (f dockerFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	r, err := http.Get(fmt.Sprintf(defaultIndex, f.imageName))
	if err != nil {
		return nil, versionKey, err
	}
	defer r.Body.Close()

	if r.StatusCode/100 != 2 {
		return nil, "", fmt.Errorf("error response: %s", r.Status)
	}

	var tags []tag
	err = json.NewDecoder(r.Body).Decode(&tags)
	if err != nil {
		return nil, "", err
	}

	tagNames := append(filter.Versions{}, versions...)
	for _, t := range tags {
		tagNames = append(tagNames, filter.NewVersionWithName(t.Name, nil))
	}

	return tagNames, versionKey, nil
}
