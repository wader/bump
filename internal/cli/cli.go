package cli

import (
	"errors"
	"flag"
	"fmt"
	"strings"

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

func csvToSlice(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ',' })
}

// Command is a command based interface to bump packages
type Command struct {
	Version string
	OS      bump.OS
}

func (cmd Command) filters() []filter.NamedFilter {
	return all.Filters()
}

func (c Command) help(flags *flag.FlagSet) string {
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

EXIT CODE:
  0: All went fine
  1: Something went wrong
  3: Check found new versions

BUMPFILE is a file with CONFIG:s or glob patterns of FILE:s
FILE is a file with EMBEDCONFIG:s or versions to be checked and updated
EMBEDCONFIG is "bump: CONFIG"
CONFIG is
  NAME /REGEXP/ PIPELINE |
  NAME command COMMAND |
  NAME after COMMAND |
  NAME message MESSAGE |
  NAME link TITLE URL
NAME is a configuration name
REGEXP is a regexp with one submatch to find current version
PIPELINE is a filter pipeline: FILTER|FILTER|...
FILTER
{{FILTER_HELP}}
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
	for _, nf := range c.filters() {
		syntax, _, _ := filter.ParseHelp(nf.Help)
		filterHelps = append(filterHelps, "  "+strings.Join(syntax, " | "))
	}
	filterHelp := strings.Join(filterHelps, "\n")

	return strings.NewReplacer(
		"{{ARGV0}}", c.OS.Args()[0],
		"{{VERSION}}", c.Version,
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

// Run bump command
func (c Command) Run() ([]error, int) {
	errs, ec := c.run()
	for _, err := range errs {
		fmt.Fprintln(c.OS.Stderr(), err)
	}
	return errs, ec
}

func (c Command) run() ([]error, int) {
	var bumpfile string
	var include string
	var exclude string
	var verbose bool
	var runCommands bool

	flags := flag.NewFlagSet(c.OS.Args()[0], flag.ContinueOnError)
	flags.StringVar(&bumpfile, "f", BumpfileName, "Bumpfile to read")
	flags.StringVar(&include, "i", "", "Comma separated names to include")
	flags.StringVar(&exclude, "e", "", "Comma separated names to exclude")
	flags.BoolVar(&verbose, "v", false, "Verbose")
	flags.BoolVar(&runCommands, "r", false, "Run update commands")
	flags.SetOutput(c.OS.Stderr())
	flags.Usage = func() {
		fmt.Fprint(flags.Output(), c.help(flags))
	}
	parseFlags := func(args []string) ([]error, bool) {
		err := flags.Parse(args)
		if errors.Is(err, flag.ErrHelp) {
			flags.Usage()
			return nil, false
		} else if err != nil {
			return []error{err}, false
		}
		return nil, true
	}

	if err, ok := parseFlags(c.OS.Args()[1:]); err != nil || !ok {
		return err, 1
	}
	if len(flags.Args()) == 0 {
		flags.Usage()
		return nil, 0
	}
	command := flags.Arg(0)
	if errs, ok := parseFlags(flags.Args()[1:]); errs != nil || !ok {
		return errs, 1
	}

	if command == "version" {
		fmt.Fprintf(c.OS.Stdout(), "%s\n", c.Version)
		return nil, 0
	} else if command == "help" {
		filterName := flags.Arg(0)
		if filterName == "" {
			flags.Usage()
			return nil, 0
		}
		for _, nf := range c.filters() {
			if filterName == nf.Name {
				fmt.Fprint(c.OS.Stdout(), c.helpFilter(nf))
				return nil, 0
			}
		}
		fmt.Fprintf(c.OS.Stdout(), "Filter not found\n")
		return nil, 0
	}

	files := flags.Args()
	includes := map[string]bool{}
	excludes := map[string]bool{}
	var bfs *bump.FileSet
	var errs []error

	for _, n := range csvToSlice(include) {
		includes[n] = true
	}
	for _, n := range csvToSlice(exclude) {
		excludes[n] = true
	}

	switch command {
	case "list", "current", "check", "diff", "update":
		bumpfilePassed := flagWasPassed(flags, "f")
		if bumpfilePassed && len(files) > 0 {
			return []error{errors.New("both bumpfile and file arguments can't be specified")}, 1
		}

		bfs, errs = bump.NewBumpFileSet(c.OS, c.filters(), bumpfile, files)
		if errs != nil {
			return errs, 1
		}
	}

	if bfs != nil && (len(includes) > 0 || len(excludes) > 0) {
		names := map[string]bool{}
		for _, c := range bfs.Checks {
			names[c.Name] = true
		}
		for n := range includes {
			if _, found := names[n]; !found {
				return []error{fmt.Errorf("include name %q not found", n)}, 1
			}
		}
		for n := range excludes {
			if _, found := names[n]; !found {
				return []error{fmt.Errorf("exclude name %q not found", n)}, 1
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
				fmt.Fprintf(c.OS.Stdout(), "%s:%d: %s\n", check.File.Name, check.PipelineLineNr, check)
				for _, cs := range check.CommandShells {
					fmt.Fprintf(c.OS.Stdout(), "%s:%d: %s command %s\n", cs.File.Name, cs.LineNr, check.Name, cs.Cmd)
				}
				for _, ca := range check.AfterShells {
					fmt.Fprintf(c.OS.Stdout(), "%s:%d: %s after %s\n", ca.File.Name, ca.LineNr, check.Name, ca.Cmd)
				}
				for _, m := range check.Messages {
					fmt.Fprintf(c.OS.Stdout(), "%s:%d: %s message %s\n", m.File.Name, m.LineNr, check.Name, m.Message)
				}
				for _, l := range check.Links {
					fmt.Fprintf(c.OS.Stdout(), "%s:%d: %s link %q %s\n", l.File.Name, l.LineNr, check.Name, l.Title, l.URL)
				}
			} else {
				fmt.Fprintf(c.OS.Stdout(), "%s\n", check.Name)
			}
		}
	case "current":
		for _, check := range bfs.SelectedChecks() {
			for _, current := range check.Currents {
				fmt.Fprintf(c.OS.Stdout(), "%s:%d: %s %s\n", current.File.Name, current.LineNr, check.Name, current.Version)
			}
		}
	case "check", "diff", "update":
		ua, errs := bfs.UpdateActions()
		if errs != nil {
			return errs, 1
		}

		switch command {
		case "check":
			if verbose {
				for _, check := range bfs.Checks {
					for _, current := range check.Currents {
						fmt.Fprintf(c.OS.Stdout(), "%s:%d: %s %s -> %s %.3fs\n",
							current.File.Name, current.LineNr, check.Name, current.Version, check.Latest,
							float32(check.PipelineDuration.Milliseconds())/1000.0)
					}
				}
			} else {
				for _, vs := range ua.VersionChanges {
					fmt.Fprintf(c.OS.Stdout(), "%s %s\n", vs.Check.Name, vs.Check.Latest)
				}
			}
			ec := 0
			if len(ua.VersionChanges) > 0 {
				// first non-special exit code to make it distinguishable from other errors
				ec = 3
			}
			return nil, ec
		case "diff":
			for _, fc := range ua.FileChanges {
				fmt.Fprint(c.OS.Stdout(), fc.Diff)
			}
		case "update":
			for _, fc := range ua.FileChanges {
				if err := c.OS.WriteFile(fc.File.Name, []byte(fc.NewText)); err != nil {
					return []error{err}, 1
				}
			}
			if runCommands {
				for _, rs := range ua.RunShells {
					if verbose {
						fmt.Fprintf(c.OS.Stdout(), "%s: shell: %s %s\n", rs.Check.Name, strings.Join(rs.Env, " "), rs.Cmd)
					}
					if err := c.OS.Shell(rs.Cmd, rs.Env); err != nil {
						return []error{fmt.Errorf("%s: shell: %s: %w", rs.Check.Name, rs.Cmd, err)}, 1
					}
				}
			} else if len(ua.RunShells) > 0 {
				for _, rs := range ua.RunShells {
					fmt.Fprintf(c.OS.Stdout(), "skipping %s: shell: %s %s\n", rs.Check.Name, strings.Join(rs.Env, " "), rs.Cmd)
				}
			}
		}
	case "pipeline":
		plStr := flags.Arg(0)
		pl, err := pipeline.New(c.filters(), plStr)
		if err != nil {
			return []error{err}, 1
		}
		logFn := func(format string, v ...interface{}) {}
		if verbose {
			logFn = func(format string, v ...interface{}) {
				fmt.Fprintf(c.OS.Stderr(), format+"\n", v...)
			}
		}
		logFn("Parsed pipeline: %s", pl)
		v, err := pl.Value(logFn)
		if err != nil {
			return []error{err}, 1
		}
		fmt.Fprintf(c.OS.Stdout(), "%s\n", v)
	default:
		flags.Usage()
		return []error{fmt.Errorf("unknown command: %s", command)}, 1
	}

	return nil, 0
}
