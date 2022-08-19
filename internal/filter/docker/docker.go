package docker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

// TODO: support other registries
// TODO: auth?
const authURLTemplate = `https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull`
const listTagsURLTemplate = `https://index.docker.io/v2/%s/tags/list`

// image -> library/image
// repo/image -> repo/image
func argToRepoImage(a string) (string, error) {
	parts := strings.Split(a, "/")
	switch {
	case len(parts) == 0:
		return "", fmt.Errorf("invalid name")
	case len(parts) == 1:
		return "library/" + parts[0], nil
	default:
		return a, nil
	}
}

// New docker filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix != Name {
		return nil, nil
	}
	if arg == "" {
		return nil, fmt.Errorf("needs a image name")
	}

	repoImage, err := argToRepoImage(arg)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, arg)
	}

	return dockerFilter{
		imageName: arg,
		repoImage: repoImage,
	}, nil
}

type dockerFilter struct {
	imageName string
	repoImage string
}

func (f dockerFilter) String() string {
	return Name + ":" + f.imageName
}

func (f dockerFilter) getToken() (string, error) {
	r, err := http.Get(fmt.Sprintf(authURLTemplate, f.repoImage))
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer r.Body.Close()

	var resp struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	return resp.Token, nil
}

func (f dockerFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	token, err := f.getToken()
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(listTagsURLTemplate, f.repoImage), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, versionKey, err
	}
	defer r.Body.Close()

	if r.StatusCode/100 != 2 {
		return nil, "", fmt.Errorf("error response: %s", r.Status)
	}

	var resp struct {
		Tags []string `json:"tags"`
	}

	err = json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		return nil, "", err
	}

	tagNames := append(filter.Versions{}, versions...)
	for _, t := range resp.Tags {
		tagNames = append(tagNames, filter.NewVersionWithName(t, nil))
	}

	return tagNames, versionKey, nil
}
