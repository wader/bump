package lexer_test

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/wader/bump/internal/lexer"
)

func TestScan(t *testing.T) {
	makeStr := func() *string {
		var s string
		return &s
	}
	type input struct {
		s        string
		expected map[string]string
	}
	testCases := []struct {
		makeScanFn func(vars map[string]*string) lexer.ScanFn
		vars       map[string]*string
		inputs     []input
	}{
		{
			makeScanFn: func(vars map[string]*string) lexer.ScanFn {
				return lexer.Concat(
					lexer.Var("title", vars["title"], lexer.Or(
						lexer.Quoted(`"`),
						lexer.Re(regexp.MustCompile(`\w`)),
					)),
					lexer.Re(regexp.MustCompile(`\s`)),
					lexer.Var("URL", vars["URL"], lexer.Rest(1)),
				)
			},
			vars: map[string]*string{
				"title": makeStr(),
				"URL":   makeStr(),
			},
			inputs: []input{
				{
					s: `aaa bbb`,
					expected: map[string]string{
						"title": "aaa",
						"URL":   "bbb",
					},
				},
				{
					s: `"aaa aaa" bbb`,
					expected: map[string]string{
						"title": "aaa aaa",
						"URL":   "bbb",
					},
				},
			},
		},
		{
			makeScanFn: func(vars map[string]*string) lexer.ScanFn {
				return lexer.Concat(
					lexer.Var("name", vars["name"], lexer.Or(
						lexer.Quoted(`"`),
						lexer.Re(regexp.MustCompile(`\w`)),
					)),
					lexer.Re(regexp.MustCompile(`\s`)),
					lexer.Var("title", vars["title"], lexer.Or(
						lexer.Quoted(`"`),
						lexer.Re(regexp.MustCompile(`\w`)),
					)),
					lexer.Re(regexp.MustCompile(`\s`)),
					lexer.Var("rest", vars["rest"], lexer.Or(
						lexer.Quoted(`"`),
						lexer.Rest(1),
					)),
				)
			},
			vars: map[string]*string{
				"name":  makeStr(),
				"title": makeStr(),
				"rest":  makeStr(),
			},
			inputs: []input{
				{
					s: `aaa bbb ccc ccc`,
					expected: map[string]string{
						"name":  "aaa",
						"title": "bbb",
						"rest":  "ccc ccc",
					},
				},
				{
					s: `"aaa aaa" bbb ccc ccc`,
					expected: map[string]string{
						"name":  "aaa aaa",
						"title": "bbb",
						"rest":  "ccc ccc",
					},
				},
				{
					s: `aaa "bbb bbb" ccc ccc`,
					expected: map[string]string{
						"name":  "aaa",
						"title": "bbb bbb",
						"rest":  "ccc ccc",
					},
				},
				{
					s: `"aaa aaa" "bbb bbb" ccc ccc`,
					expected: map[string]string{
						"name":  "aaa aaa",
						"title": "bbb bbb",
						"rest":  "ccc ccc",
					},
				},
				{
					s: `"aaa aaa" "bbb bbb" "ccc ccc"`,
					expected: map[string]string{
						"name":  "aaa aaa",
						"title": "bbb bbb",
						"rest":  "ccc ccc",
					},
				},
			},
		},
	}
	for _, tC := range testCases {
		for _, i := range tC.inputs {
			t.Run(i.s, func(t *testing.T) {
				_, err := lexer.Scan(i.s, tC.makeScanFn(tC.vars))
				if err != nil {
					t.Fatal(err)
				}

				actual := map[string]string{}
				for k, v := range tC.vars {
					actual[k] = *v
				}
				if !reflect.DeepEqual(actual, i.expected) {
					t.Errorf("expected %v, actual %v", i.expected, actual)
				}
			})
		}
	}
}
