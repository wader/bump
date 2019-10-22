package bump

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/all"
	"github.com/wader/bump/internal/naivediff"
	"github.com/wader/bump/internal/pipeline"
)

// Enver is a environment the command is running in
type Enver interface {
	Args() []string
	Stdout() io.Writer
	Stderr() io.Writer
	WriteFile(filename string, data []byte) error
	ReadFile(filename string) ([]byte, error)
}

// Command is a command based interface to bump packages
type Command struct {
	Version string
	Env     Enver
}

func (cmd Command) filters() []filter.NamedFilter {
	return all.Filters()
}

func (cmd Command) help() string {
	text := `
Usage: $argv0 [OPTIONS] COMMAND
OPTIONS:
  -e string             Exclude specified names (space or comma separated)
  -i string             Include specified names (space or comma separated)
  -v                    Verbose

COMMANDS:
  version               Show version of bump itself ($version)
  help [FILTER]         Show help or filter help
  list FILES...         Show bump configurations
  current FILES...      Show current versions
  check FILES...        Check for possible version updates
  update FILES...       Update versions
  diff FILES...         Show diff of what an update would change
  pipeline PIPELINE     Run a filter pipeline

FILES is files with CONFIGURATION or versions to be checked or updated
PIPELINE is a filter pipeline: FILTER|FILTER|...
FILTER
$filterhelp
CONFIGURATION lines looks like this: bump: NAME /REGEXP/ PIPELINE
NAME is a configuration identifier
REGEXP is a regexp with one submatch to find current version
`[1:]

	var filterHelps []string
	for _, nf := range cmd.filters() {
		parts := strings.SplitN(nf.Help, "\n\n", 3)
		filterHelps = append(filterHelps, "  "+parts[0])
	}
	filterhelp := strings.Join(filterHelps, "\n")

	return strings.NewReplacer(
		"$argv0", cmd.Env.Args()[0],
		"$version", cmd.Version,
		"$filterhelp", filterhelp,
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
	var verbose bool
	var include = ""
	var exclude = ""

	f := flag.NewFlagSet(cmd.Env.Args()[0], flag.ContinueOnError)
	f.SetOutput(cmd.Env.Stderr())
	f.Usage = func() {
		fmt.Fprint(f.Output(), cmd.help())
	}
	f.StringVar(&include, "i", "", "Include specified names (space or comma separated)")
	f.StringVar(&exclude, "e", "", "Exclude specified names (space or comma separated)")
	f.BoolVar(&verbose, "v", false, "Verbose")
	err := f.Parse(cmd.Env.Args()[1:])
	if err == flag.ErrHelp {
		f.Usage()
		return nil
	} else if err != nil {
		return []error{err}
	}

	if len(f.Args()) == 0 {
		f.Usage()
		return nil
	}

	command := f.Arg(0)

	if command == "version" {
		fmt.Fprintf(cmd.Env.Stdout(), "%s\n", cmd.Version)
		return nil
	} else if command == "help" {
		filterName := f.Arg(1)
		if filterName == "" {
			f.Usage()
			return nil
		}
		for _, nf := range cmd.filters() {
			if filterName == nf.Name {
				fmt.Fprintf(cmd.Env.Stdout(), cmd.helpFilter(nf))
				return nil
			}
		}
		fmt.Fprintf(cmd.Env.Stdout(), "Filter not found\n")
		return nil
	}

	files := f.Args()[1:]
	includes := map[string]bool{}
	excludes := map[string]bool{}
	var bfs *FileSet
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
		bfs, errs = NewBumpFileSet(cmd.filters(), cmd.Env.ReadFile, files)
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
		bfs.SkipCheckFn = func(c *Check) bool {
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

		if errs := bfs.Latest(); errs != nil {
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
			newTextBuf := bfs.Replace(f.Text)
			if bytes.Compare(f.Text, newTextBuf) == 0 {
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
		plStr := f.Arg(1)
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
		f.Usage()
		return []error{fmt.Errorf("unknown command: %s", command)}
	}

	return nil
}
