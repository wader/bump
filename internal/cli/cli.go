package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wader/bump/internal/bump"
	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/all"
	"github.com/wader/bump/internal/naivediff"
	"github.com/wader/bump/internal/pipeline"
)

// BumpfileName default Bumpfile name
const BumpfileName = "Bumpfile"

func flagWasPassed(flags *flag.FlagSet, name string) bool {
	passed := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			passed = true
		}
	})
	return passed
}

// Command is a command based interface to bump packages
type Command struct {
	Version string
	Env     bump.Env
}

func (cmd Command) filters() []filter.NamedFilter {
	return all.Filters()
}

func (cmd Command) help(flags *flag.FlagSet) string {
	text := `
Usage: {{ARGV0}} [OPTIONS] COMMAND
OPTIONS:
{{OPTIONS_HELP}}

COMMANDS:
  version               Show version of bump itself ({{VERSION}})
  help [FILTER]         Show help or help for a filter
  list [FILE...]        Show bump configurations
  current [FILE...]     Show current versions
  check [FILE...]       Check for possible version updates
  update [FILE...]      Update versions
  diff [FILE...]        Show diff of what an update would change
  pipeline PIPELINE     Run a filter pipeline

BUMPFILE is a file with CONFIG:s or glob patterns of FILE:s
FILE is file with EMBEDCONFIG:s or versions to be checked or updated
CONFIG is "NAME /REGEXP/ PIPELINE"
EMBEDCONFIG is "bump: CONFIG"
PIPELINE is a filter pipeline: FILTER|FILTER|...
FILTER
{{FILTER_HELP}}
NAME is a configuration name
REGEXP is a regexp with one submatch to find current version
`[1:]

	var optionsHelps []string
	flags.VisitAll(func(f *flag.Flag) {
		var ss []string
		ss = append(ss, fmt.Sprintf("  -%-20s", f.Name))
		ss = append(ss, f.Usage)
		if f.DefValue != "" {
			ss = append(ss, fmt.Sprintf("(%s)", f.DefValue))
		}
		optionsHelps = append(optionsHelps, strings.Join(ss, " "))
	})
	optionHelp := strings.Join(optionsHelps, "\n")

	var filterHelps []string
	for _, nf := range cmd.filters() {
		syntax, _, _ := filter.ParseHelp(nf.Help)
		filterHelps = append(filterHelps, "  "+strings.Join(syntax, " | "))
	}
	filterHelp := strings.Join(filterHelps, "\n")

	return strings.NewReplacer(
		"{{ARGV0}}", cmd.Env.Args()[0],
		"{{VERSION}}", cmd.Version,
		"{{OPTIONS_HELP}}", optionHelp,
		"{{FILTER_HELP}}", filterHelp,
	).Replace(text)
}

func (cmd Command) helpFilter(nf filter.NamedFilter) string {
	syntax, description, examples := filter.ParseHelp(nf.Help)

	var examplesRuns []string
	for _, e := range examples {
		if strings.HasPrefix(e, "#") {
			examplesRuns = append(examplesRuns, e)
			continue
		}
		examplesRuns = append(examplesRuns, fmt.Sprintf("bump pipeline '%s'", e))
	}

	return fmt.Sprintf(`
Syntax:
%s

%s

Examples:
%s
`[1:],
		strings.Join(syntax, ", "),
		description,
		strings.Join(examplesRuns, "\n"),
	)
}

func (cmd Command) formatDiff(a, b string, patch string) string {
	return fmt.Sprintf(`
--- %s
+++ %s
%s`[1:],
		a, b, patch)
}

// Run bump command
func (cmd Command) Run() []error {
	errs := cmd.run()
	for _, err := range errs {
		fmt.Fprintln(cmd.Env.Stderr(), err)
	}
	return errs
}

func (cmd Command) run() []error {
	var bumpfile string
	var include string
	var exclude string
	var verbose bool

	flags := flag.NewFlagSet(cmd.Env.Args()[0], flag.ContinueOnError)
	flags.StringVar(&bumpfile, "c", BumpfileName, "Bumpfile to read")
	flags.StringVar(&include, "i", "", "Comma separated names to include")
	flags.StringVar(&exclude, "e", "", "Comma separated names to exclude")
	flags.BoolVar(&verbose, "v", false, "Verbose")
	flags.SetOutput(cmd.Env.Stderr())
	flags.Usage = func() {
		fmt.Fprint(flags.Output(), cmd.help(flags))
	}

	err := flags.Parse(cmd.Env.Args()[1:])
	if err == flag.ErrHelp {
		flags.Usage()
		return nil
	} else if err != nil {
		return []error{err}
	}
	bumpfilePassed := flagWasPassed(flags, "c")

	if len(flags.Args()) == 0 {
		flags.Usage()
		return nil
	}

	command := flags.Arg(0)

	if command == "version" {
		fmt.Fprintf(cmd.Env.Stdout(), "%s\n", cmd.Version)
		return nil
	} else if command == "help" {
		filterName := flags.Arg(1)
		if filterName == "" {
			flags.Usage()
			return nil
		}
		for _, nf := range cmd.filters() {
			if filterName == nf.Name {
				fmt.Fprint(cmd.Env.Stdout(), cmd.helpFilter(nf))
				return nil
			}
		}
		fmt.Fprint(cmd.Env.Stdout(), "Filter not found\n")
		return nil
	}

	files := flags.Args()[1:]
	includes := map[string]bool{}
	excludes := map[string]bool{}
	var bfs *bump.FileSet
	var errs []error

	include = strings.Replace(strings.TrimSpace(include), ",", " ", -1)
	if include != "" {
		for _, n := range strings.Fields(include) {
			includes[n] = true
		}
	}
	exclude = strings.Replace(strings.TrimSpace(exclude), ",", " ", -1)
	if exclude != "" {
		for _, n := range strings.Fields(exclude) {
			excludes[n] = true
		}
	}

	switch command {
	case "list", "current", "check", "diff", "update":

		if bumpfilePassed && len(files) > 0 {
			return []error{errors.New("both bumpfile and file arguments can't be specified")}
		}

		bfs, errs = bump.NewBumpFileSet(cmd.Env, cmd.filters(), bumpfile, files)
		if errs != nil {
			return errs
		}
	}

	if bfs != nil && (len(includes) > 0 || len(excludes) > 0) {
		names := map[string]bool{}
		for _, c := range bfs.Checks {
			names[c.Name] = true
		}
		for n := range includes {
			if _, found := names[n]; !found {
				return []error{fmt.Errorf("include name %q not found", n)}
			}
		}
		for n := range excludes {
			if _, found := names[n]; !found {
				return []error{fmt.Errorf("exclude name %q not found", n)}
			}
		}
		bfs.SkipCheckFn = func(c *bump.Check) bool {
			includeFound := true
			if len(include) > 0 {
				_, includeFound = includes[c.Name]
			}
			_, excludeFound := excludes[c.Name]
			return excludeFound || !includeFound
		}
	}

	switch command {
	case "list":
		for _, c := range bfs.SelectedChecks() {
			if verbose {
				fmt.Fprintf(cmd.Env.Stdout(), "%s:%d: %s\n", c.File.Name, c.LineNr, c)
			} else {
				fmt.Fprintf(cmd.Env.Stdout(), "%s\n", c.Name)
			}
		}
	case "current":
		for _, check := range bfs.SelectedChecks() {
			for _, c := range check.Currents {
				fmt.Fprintf(cmd.Env.Stdout(), "%s:%d: %s %s\n", c.File.Name, c.LineNr, check.Name, c.Version)
			}
		}
	case "check", "diff", "update":
		type change struct {
			file    string
			line    int
			version string
		}
		type update struct {
			name    string
			version string
			changes []change
		}
		type file struct {
			name string
			text string
		}
		type result struct {
			diff        string
			updates     []update
			fileChanges []file
		}

		var resultFn func(check *bump.Check, err error, duration time.Duration)
		if verbose {
			var resultFnMU sync.Mutex
			resultFn = func(check *bump.Check, err error, duration time.Duration) {
				resultFnMU.Lock()
				defer resultFnMU.Unlock()
				var result string
				if err == nil {
					result = check.Latest
				} else {
					result = err.Error()
				}
				fmt.Fprintf(cmd.Env.Stdout(), "%s %s %s\n", check.Name, result, duration)
			}
		}

		if errs := bfs.Latest(resultFn); errs != nil {
			return errs
		}

		var r result

		for _, check := range bfs.SelectedChecks() {
			var changes []change

			for _, c := range check.Currents {
				if c.Version == check.Latest {
					continue
				}

				changes = append(changes, change{
					file:    c.File.Name,
					line:    c.LineNr,
					version: c.Version,
				})
			}

			if len(changes) == 0 {
				continue
			}

			r.updates = append(r.updates, update{
				name:    check.Name,
				version: check.Latest,
				changes: changes,
			})
		}

		var diffs []string
		for _, f := range bfs.Files {
			newTextBuf := bfs.Replace(f)
			if bytes.Equal(f.Text, newTextBuf) {
				continue
			}
			newText := string(newTextBuf)

			diffs = append(diffs, cmd.formatDiff(
				f.Name,
				f.Name,
				naivediff.Diff(string(f.Text), newText, 3),
			))

			r.fileChanges = append(r.fileChanges, file{
				name: f.Name,
				text: newText,
			})
		}
		r.diff = strings.Join(diffs, "")

		switch command {
		case "check":
			for _, u := range r.updates {
				if verbose {
					for _, c := range u.changes {
						fmt.Fprintf(cmd.Env.Stdout(), "%s:%d: %s %s -> %s\n", c.file, c.line, u.name, c.version, u.version)
					}
				} else {
					fmt.Fprintf(cmd.Env.Stdout(), "%s %s\n", u.name, u.version)
				}
			}
		case "diff":
			fmt.Fprint(cmd.Env.Stdout(), r.diff)
		case "update":
			for _, f := range r.fileChanges {
				if err := cmd.Env.WriteFile(f.name, []byte(f.text)); err != nil {
					return []error{err}
				}
			}
		}
	case "pipeline":
		plStr := flags.Arg(1)
		pl, err := pipeline.New(cmd.filters(), plStr)
		if err != nil {
			return []error{err}
		}
		logFn := func(format string, v ...interface{}) {}
		if verbose {
			logFn = func(format string, v ...interface{}) {
				fmt.Fprintf(cmd.Env.Stderr(), format+"\n", v...)
			}
		}
		logFn("Parsed pipeline: %s", pl)
		v, err := pl.Value(logFn)
		if err != nil {
			return []error{err}
		}
		fmt.Fprintf(cmd.Env.Stdout(), "%s\n", v)
	default:
		flags.Usage()
		return []error{fmt.Errorf("unknown command: %s", command)}
	}

	return nil
}
