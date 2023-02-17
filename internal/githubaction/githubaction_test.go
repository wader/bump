package githubaction_test

import (
	"testing"

	"github.com/wader/bump/internal/bump"
	"github.com/wader/bump/internal/githubaction"
)

func Test(t *testing.T) {
	c := &bump.Check{
		Name:   "aaa",
		Latest: "3",
		Currents: []bump.Current{
			{Version: "1"},
			{Version: "2"},
			{Version: "2"},
		},
		Messages: []bump.CheckMessage{
			{Message: "msg1 $NAME/$CURRENT/$LATEST"},
			{Message: "msg2 $NAME/$CURRENT/$LATEST"},
		},
		Links: []bump.CheckLink{
			{Title: "title 1 $NAME/$CURRENT/$LATEST", URL: "https://1/$NAME/$CURRENT/$LATEST"},
			{Title: "title 2 $NAME/$CURRENT/$LATEST", URL: "https://2/$NAME/$CURRENT/$LATEST"},
		},
	}

	tf := githubaction.CheckTemplateReplaceFn(c)

	testCases := []struct {
		template string
		expected string
	}{
		{`Update {{.Name}} to {{.Latest}} from {{join .Current ", "}}`, `Update aaa to 3 from 1, 2`},
		{
			`` +
				`{{range .Messages}}{{.}}{{"\n\n"}}{{end}}` +
				`{{range .Links}}{{.Title}} {{.URL}}{{"\n"}}{{end}}`,
			"" +
				"msg1 aaa/1/3\n\n" +
				"msg2 aaa/1/3\n\n" +
				"title 1 aaa/1/3 https://1/aaa/1/3\n" +
				"title 2 aaa/1/3 https://2/aaa/1/3\n",
		},
		{
			`` +
				`{{range .Messages}}{{.}}{{"\n\n"}}{{end}}` +
				`{{range .Links}}[{{.Title}}]({{.URL}})  {{"\n"}}{{end}}`,
			"" +
				"msg1 aaa/1/3\n\n" +
				"msg2 aaa/1/3\n\n" +
				"[title 1 aaa/1/3](https://1/aaa/1/3)  \n" +
				"[title 2 aaa/1/3](https://2/aaa/1/3)  \n",
		},
		{`bump-{{.Name}}-{{.Latest}}`, `bump-aaa-3`},
	}
	for _, tC := range testCases {
		t.Run(tC.template, func(t *testing.T) {
			actual, err := tf(tC.template)
			if err != nil {
				t.Error(err)
			}
			if tC.expected != actual {
				t.Errorf("expected %q, got %q", tC.expected, actual)
			}
		})
	}
}
