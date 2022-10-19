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
