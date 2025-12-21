package dockerv2_test

import (
	"testing"

	"github.com/wader/bump/internal/dockerv2"
)

func TestParseWWWAuth(t *testing.T) {
	w, err := dockerv2.ParseWWWAuth(`Bearer realm="https://ghcr.io/token",service="ghcr.io",scope="repository:org/image:pull"`)
	if err != nil {
		t.Fatal(err)
	}
	if w.Scheme != "Bearer" {
		t.Fatalf("schema %s", w.Scheme)
	}
	if v := w.Params["service"]; v != "ghcr.io" {
		t.Fatalf("service %s", v)
	}
}

func TestParseLinkHeader(t *testing.T) {
	p, err := dockerv2.ParseLinkHeader(`</v2/library/debian/tags/list?last=oldoldstable-20210208&n=1000>; rel="next"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 1 {
		t.Fatalf("len(p) != 1, is %d", len(p))
	}
	expectedRawURL := `/v2/library/debian/tags/list?last=oldoldstable-20210208&n=1000`
	actualRawURL := p[0].RawURL
	if expectedRawURL != actualRawURL {
		t.Fatalf("expected RawURL %s, got %s", expectedRawURL, actualRawURL)
	}
	expectedParamsRel := "next"
	actualParamsRel := p[0].Params["rel"]
	if expectedParamsRel != actualParamsRel {
		t.Fatalf("expected Params[rel] %s, got %s", expectedParamsRel, actualParamsRel)
	}
}
