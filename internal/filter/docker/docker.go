package docker

import (
	"fmt"

	"github.com/wader/bump/internal/dockerv2"
	"github.com/wader/bump/internal/filter"
)

// Name of filter
const Name = "docker"

// Help text
var Help = `
docker:<image>

Produce versions from a image on ducker hub or other registry.
Currently only supports anonymous access.

docker:alpine|^3
docker:mwader/static-ffmpeg|^4
docker:ghcr.io/nginx-proxy/nginx-proxy|^0.9
`[1:]

// New docker filter
func New(prefix string, arg string) (filter filter.Filter, err error) {
	if prefix != Name {
		return nil, nil
	}
	if arg == "" {
		return nil, fmt.Errorf("needs a image name")
	}

	registry, err := dockerv2.NewFromImage(arg)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, arg)
	}

	return dockerFilter{
		image:    arg,
		registry: registry,
	}, nil
}

type dockerFilter struct {
	image    string
	registry *dockerv2.Registry
}

func (f dockerFilter) String() string {
	return Name + ":" + f.image
}

func (f dockerFilter) Filter(versions filter.Versions, versionKey string) (filter.Versions, string, error) {
	tags, tagsErr := f.registry.Tags()
	if tagsErr != nil {
		return filter.Versions{}, "", tagsErr
	}

	tagNames := append(filter.Versions{}, versions...)
	for _, t := range tags {
		tagNames = append(tagNames, filter.NewVersionWithName(t, nil))
	}

	return tagNames, versionKey, nil
}
