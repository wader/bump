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
	var runCommands bool

	flags := flag.NewFlagSet(cmd.OS.Args()[0], flag.ContinueOnError)
	flags.StringVar(&bumpfile, "f", BumpfileName, "Bumpfile to read")
	flags.StringVar(&include, "i", "", "Comma separated names to include")
	flags.StringVar(&exclude, "e", "", "Comma separated names to exclude")
	flags.BoolVar(&verbose, "v", false, "Verbose")
	flags.BoolVar(&runCommands, "r", false, "Run update commands")
	flags.SetOutput(cmd.OS.Stderr())
	flags.Usage = func() {
		fmt.Fprint(flags.Output(), cmd.help(flags))
	}
	parseFlags := func(args []string) ([]error, bool) {
		err := flags.Parse(args)
		if err == flag.ErrHelp {
			flags.Usage()
			return nil, false
		} else if err != nil {
			return []error{err}, false
		}
		return nil, true
	}

	if err, ok := parseFlags(cmd.OS.Args()[1:]); err != nil || !ok {
		return err
	}
	if len(flags.Args()) == 0 {
		flags.Usage()
		return nil
	}
	command := flags.Arg(0)
	if err, ok := parseFlags(flags.Args()[1:]); err != nil || !ok {
		return err
	}

	if command == "version" {
		fmt.Fprintf(cmd.OS.Stdout(), "%s\n", cmd.Version)
		return nil
	} else if command == "help" {
		filterName := flags.Arg(0)
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
				for _, m := range check.Messages {
					fmt.Fprintf(cmd.OS.Stdout(), "%s:%d: %s message %s\n", m.File.Name, m.LineNr, check.Name, m.Message)
				}
				for _, l := range check.Links {
					fmt.Fprintf(cmd.OS.Stdout(), "%s:%d: %s link %q %s\n", l.File.Name, l.LineNr, check.Name, l.Title, l.URL)
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
		ua, errs := bfs.UpdateActions()
		if errs != nil {
			return errs
		}

		switch command {
		case "check":
			if verbose {
				for _, check := range bfs.Checks {
					for _, c := range check.Currents {
						fmt.Fprintf(cmd.OS.Stdout(), "%s:%d: %s %s -> %s %.3fs\n",
							c.File.Name, c.LineNr, check.Name, c.Version, check.Latest,
							float32(check.PipelineDuration.Milliseconds())/1000.0)
					}
				}
			} else {
				for _, vs := range ua.VersionChanges {
					fmt.Fprintf(cmd.OS.Stdout(), "%s %s\n", vs.Check.Name, vs.Check.Latest)
				}
			}
		case "diff":
			for _, fc := range ua.FileChanges {
				fmt.Fprint(cmd.OS.Stdout(), fc.Diff)
			}
		case "update":
			for _, fc := range ua.FileChanges {
				if err := cmd.OS.WriteFile(fc.File.Name, []byte(fc.NewText)); err != nil {
					return []error{err}
				}
			}
			if runCommands {
				for _, rs := range ua.RunShells {
					if verbose {
						fmt.Fprintf(cmd.OS.Stdout(), "%s: shell: %s %s\n", rs.Check.Name, strings.Join(rs.Env, " "), rs.Cmd)
					}
					if err := cmd.OS.Shell(rs.Cmd, rs.Env); err != nil {
						return []error{fmt.Errorf("%s: shell: %s: %w", rs.Check.Name, rs.Cmd, err)}
					}
				}
			} else if len(ua.RunShells) > 0 {
				for _, rs := range ua.RunShells {
					fmt.Fprintf(cmd.OS.Stdout(), "skipping %s: shell: %s %s\n", rs.Check.Name, strings.Join(rs.Env, " "), rs.Cmd)
				}
			}
		}
	case "pipeline":
		plStr := flags.Arg(0)
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
