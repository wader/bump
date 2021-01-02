package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/wader/bump/internal/bump"
	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/all"
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
	OS      bump.OS
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
		"{{ARGV0}}", cmd.OS.Args()[0],
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
		fmt.Fprintln(cmd.OS.Stderr(), err)
	}
	return errs
}

func (cmd Command) run() []error {
	var bumpfile string
	var include string
	var exclude string
	var verbose bool

	flags := flag.NewFlagSet(cmd.OS.Args()[0], flag.ContinueOnError)
	flags.StringVar(&bumpfile, "f", BumpfileName, "Bumpfile to read")
	flags.StringVar(&include, "i", "", "Comma separated names to include")
	flags.StringVar(&exclude, "e", "", "Comma separated names to exclude")
	flags.BoolVar(&verbose, "v", false, "Verbose")
	flags.SetOutput(cmd.OS.Stderr())
	flags.Usage = func() {
		fmt.Fprint(flags.Output(), cmd.help(flags))
	}

	err := flags.Parse(cmd.OS.Args()[1:])
	if err == flag.ErrHelp {
		flags.Usage()
		return nil
	} else if err != nil {
		return []error{err}
	}
	bumpfilePassed := flagWasPassed(flags, "f")

	if len(flags.Args()) == 0 {
		flags.Usage()
		return nil
	}

	command := flags.Arg(0)

	if command == "version" {
		fmt.Fprintf(cmd.OS.Stdout(), "%s\n", cmd.Version)
		return nil
	} else if command == "help" {
		filterName := flags.Arg(1)
		if filterName == "" {
			flags.Usage()
			return nil
		}
		for _, nf := range cmd.filters() {
			if filterName == nf.Name {
				fmt.Fprint(cmd.OS.Stdout(), cmd.helpFilter(nf))
				return nil
			}
		}
		fmt.Fprintf(cmd.OS.Stdout(), "Filter not found\n")
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

		bfs, errs = bump.NewBumpFileSet(cmd.OS, cmd.filters(), bumpfile, files)
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
		for _, check := range bfs.SelectedChecks() {
			if verbose {
				fmt.Fprintf(cmd.OS.Stdout(), "%s:%d: %s\n", check.File.Name, check.PipelineLineNr, check)
				for _, cs := range check.CommandShells {
					fmt.Fprintf(cmd.OS.Stdout(), "%s:%d: %s command %s\n", cs.File.Name, cs.LineNr, check.Name, cs.Cmd)
				}
				for _, ca := range check.AfterShells {
					fmt.Fprintf(cmd.OS.Stdout(), "%s:%d: %s after %s\n", ca.File.Name, ca.LineNr, check.Name, ca.Cmd)
				}
			} else {
				fmt.Fprintf(cmd.OS.Stdout(), "%s\n", check.Name)
			}
		}
	case "current":
		for _, check := range bfs.SelectedChecks() {
			for _, c := range check.Currents {
				fmt.Fprintf(cmd.OS.Stdout(), "%s:%d: %s %s\n", c.File.Name, c.LineNr, check.Name, c.Version)
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
		type run struct {
			cmd string
			env []string
		}
		type result struct {
			diff        string
			updates     []update
			fileChanges []file
			commandRuns []run
			afterRuns   []run
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
				fmt.Fprintf(cmd.OS.Stdout(), "%s %s %s\n", check.Name, result, duration)
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

			env := bfs.CommandEnv(check)
			// TODO: refactor, bfs.Replace skip if there are commands
			for _, cr := range check.CommandRuns {
				r.commandRuns = append(r.commandRuns, run{cmd: cr, env: env})
			}
			for _, cr := range check.AfterRuns {
				r.commandRuns = append(r.commandRuns, run{cmd: cr, env: env})
			}
		}

		var diffs []string
		for _, f := range bfs.Files {
			newTextBuf := bfs.Replace(f)
			// might return equal even if version has changed if checks has run commands
			if bytes.Equal(f.Text, newTextBuf) {
				continue
			}
			newText := string(newTextBuf)

			udiff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
				A:        difflib.SplitLines(string(f.Text)),
				B:        difflib.SplitLines(newText),
				FromFile: f.Name,
				ToFile:   f.Name,
				Context:  3,
			})
			if err != nil {
				return []error{err}
			}

			diffs = append(diffs, udiff)

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
						fmt.Fprintf(cmd.OS.Stdout(), "%s:%d: %s %s -> %s\n", c.file, c.line, u.name, c.version, u.version)
					}
				} else {
					fmt.Fprintf(cmd.OS.Stdout(), "%s %s\n", u.name, u.version)
				}
			}
		case "diff":
			fmt.Fprint(cmd.OS.Stdout(), r.diff)
		case "update":
			for _, f := range r.fileChanges {
				if err := cmd.OS.WriteFile(f.name, []byte(f.text)); err != nil {
					return []error{err}
				}
			}
			for _, r := range r.commandRuns {
				if verbose {
					fmt.Fprintf(cmd.OS.Stdout(), "command: %s %s\n", strings.Join(r.env, " "), r.cmd)
				}
				if err := cmd.OS.Shell(r.cmd, r.env); err != nil {
					return []error{fmt.Errorf("command: %s: %w", r.cmd, err)}
				}
			}
			for _, r := range r.afterRuns {
				if verbose {
					fmt.Fprintf(cmd.OS.Stdout(), "after: %s %s\n", strings.Join(r.env, " "), r.cmd)
				}
				if err := cmd.OS.Shell(r.cmd, r.env); err != nil {
					return []error{fmt.Errorf("after: %s: %w", r.cmd, err)}
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
				fmt.Fprintf(cmd.OS.Stderr(), format+"\n", v...)
			}
		}
		logFn("Parsed pipeline: %s", pl)
		v, err := pl.Value(logFn)
		if err != nil {
			return []error{err}
		}
		fmt.Fprintf(cmd.OS.Stdout(), "%s\n", v)
	default:
		flags.Usage()
		return []error{fmt.Errorf("unknown command: %s", command)}
	}

	return nil
}
