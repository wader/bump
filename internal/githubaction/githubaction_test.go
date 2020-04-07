package githubaction_test

import (
	"testing"

	"github.com/wader/bump/internal/bump"
	"github.com/wader/bump/internal/githubaction"
)

func TestCheckTemplateReplaceFn(t *testing.T) {
	c := &bump.Check{
		Name:   "aaa",
		Latest: "3",
		Currents: []bump.Current{
			{Version: "1"},
			{Version: "2"},
		},
	}

	tf := githubaction.CheckTemplateReplaceFn(c)
	expected := "Update aaa from 1, 2 to 3"
	actual := tf("Update $name from $current to $version")
	if expected != actual {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}
