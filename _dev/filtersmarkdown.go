// Convert filter help texts into markdown
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/all"
	"github.com/wader/bump/internal/pipeline"
)

func main() {
	for _, nf := range all.Filters() {
		syntax, description, examples := filter.ParseHelp(nf.Help)

		var syntaxMDParts []string
		for i, s := range syntax {
			delim := ""
			if i < len(syntax)-2 {
				delim = ", "
			} else if i < len(syntax)-1 {
				delim = " or "
			}
			syntaxMDParts = append(syntaxMDParts, fmt.Sprintf("`%s`%s", s, delim))
		}
		var examplesMDParts []string
		for _, e := range examples {
			if strings.HasPrefix(e, "#") {
				examplesMDParts = append(examplesMDParts, e)
				continue
			}

			examplesMDParts = append(examplesMDParts, fmt.Sprintf("$ bump pipeline '%s'", e))

			p, err := pipeline.New(all.Filters(), e)
			if err != nil {
				panic(err.Error() + ":" + e)
			}

			v, err := p.Value(nil)
			if err != nil {
				examplesMDParts = append(examplesMDParts, err.Error())
			} else {
				examplesMDParts = append(examplesMDParts, v)
			}
		}

		fmt.Fprintf(os.Stdout,
			strings.NewReplacer(
				"{{name}}", nf.Name,
				"{{syntax}}", strings.Join(syntaxMDParts, ""),
				"{{desc}}", description,
				"{{examples}}", strings.Join(examplesMDParts, "\n"),
				"{{block}}", "```",
			).Replace(`
### {{name}}

{{syntax}}

{{desc}}

{{block}}sh
{{examples}}
{{block}}

`[1:]))
	}
}
