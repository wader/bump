// testing is done thru cli tests

package bump

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/lexer"
	"github.com/wader/bump/internal/locline"
	"github.com/wader/bump/internal/pipeline"
	"github.com/wader/bump/internal/rereplacer"
)

var bumpRe = regexp.MustCompile(`bump:\s*(\w.*)`)

type CheckLink struct {
	Name string
	URL  string
}

type CheckShell struct {
	Cmd    string
	File   *File
	LineNr int
}

// Check is a bump config line
type Check struct {
	File *File
	Name string

	// bump: <name> /<re>/ <pipeline>
	PipelineLineNr   int
	CurrentRE        *regexp.Regexp
	Pipeline         pipeline.Pipeline
	PipelineDuration time.Duration

	// bump: <name> command ...
	CommandShells []CheckShell
	// bump: <name> after ...
	AfterShells []CheckShell
	// bump: <name> link <title> <url>
	Links []CheckLink

	Latest   string
	Currents []Current
}

// HasUpdate returns true if any current version does not match Latest
func (c *Check) HasUpdate() bool {
	for _, cur := range c.Currents {
		if cur.Version != c.Latest {
			return true
		}
	}
	return false
}

// Current version found in a file
type Current struct {
	File    *File
	LineNr  int
	Range   [2]int
	Version string
}

func (c *Check) String() string {
	return fmt.Sprintf("%s /%s/ %s", c.Name, c.CurrentRE, c.Pipeline)
}

// FileSet is a set of File:s, filters and checks found in files
type FileSet struct {
	Files       []*File
	Filters     []filter.NamedFilter
	Checks      []*Check
	SkipCheckFn func(c *Check) bool
}

// File is file with config or versions
type File struct {
	Name          string
	Text          []byte
	HasConfig     bool
	HasCurrents   bool
	HasNoVersions bool // for Bumpfile
}

func rangeOverlap(x1, x2, y1, y2 int) bool {
	return x1 < y2 && y1 < x2
}

// NewBumpFileSet creates a new BumpFileSet
func NewBumpFileSet(
	os OS,
	filters []filter.NamedFilter,
	bumpfile string,
	filenames []string) (*FileSet, []error) {

	b := &FileSet{
		Filters: filters,
	}

	if len(filenames) > 0 {
		for _, f := range filenames {
			if err := b.addFile(os, f); err != nil {
				return nil, []error{err}
			}
		}
	} else {
		if err := b.addBumpfile(os, bumpfile); err != nil {
			return nil, []error{err}
		}
	}

	b.findCurrent()

	if errs := b.Lint(); errs != nil {
		return nil, errs
	}

	return b, nil
}

func (fs *FileSet) addBumpfile(os OS, name string) error {
	text, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	file := &File{Name: name, Text: text, HasNoVersions: true}
	fs.Files = append(fs.Files, file)

	lineNr := 0
	for _, l := range strings.Split(string(text), "\n") {
		lineNr++
		if strings.HasPrefix(l, "#") || strings.TrimSpace(l) == "" {
			continue
		}

		file.HasConfig = true

		matches, _ := os.Glob(l)
		if len(matches) > 0 {
			for _, m := range matches {
				if err := fs.addFile(os, m); err != nil {
					return err
				}
			}
			continue
		}

		err := fs.parseCheckLine(file, lineNr, l, fs.Filters)
		if err != nil {
			return fmt.Errorf("%s:%d: %w", file.Name, lineNr, err)
		}
	}

	return nil
}

func (fs *FileSet) addFile(os OS, name string) error {
	text, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	file := &File{Name: name, Text: text}
	fs.Files = append(fs.Files, file)

	err = fs.parseFile(file, fs.Filters)
	if err != nil {
		return err
	}

	return nil
}

// SelectedChecks returns selected checks based on SkipCheckFn
func (fs *FileSet) SelectedChecks() []*Check {
	if fs.SkipCheckFn == nil {
		return fs.Checks
	}

	var filteredChecks []*Check
	for _, c := range fs.Checks {
		if fs.SkipCheckFn(c) {
			continue
		}
		filteredChecks = append(filteredChecks, c)
	}

	return filteredChecks
}

// Latest run all pipelines to get latest version
func (fs *FileSet) Latest() []error {
	type result struct {
		i        int
		latest   string
		err      error
		duration time.Duration
	}

	selectedChecks := fs.SelectedChecks()
	resultCh := make(chan result, len(selectedChecks))

	wg := sync.WaitGroup{}
	wg.Add(len(selectedChecks))
	for i, c := range selectedChecks {
		go func(i int, c *Check) {
			defer wg.Done()
			start := time.Now()
			v, err := c.Pipeline.Value(nil)
			resultCh <- result{i: i, latest: v, err: err, duration: time.Since(start)}
		}(i, c)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var errs []error
	for r := range resultCh {
		c := selectedChecks[r.i]
		c.PipelineDuration = r.duration
		c.Latest = r.latest
		if r.err != nil {
			errs = append(errs, fmt.Errorf("%s:%d: %s: %w", c.File.Name, c.PipelineLineNr, c.Name, r.err))
		}
	}

	return errs
}

func (fs *FileSet) findCurrent() {
	for _, c := range fs.SelectedChecks() {
		for _, f := range fs.Files {
			if f.HasNoVersions {
				continue
			}

			locLine := locline.New(f.Text)
			checkLineSet := map[int]bool{}
			for _, sm := range bumpRe.FindAllSubmatchIndex(f.Text, -1) {
				lineNr := locLine.Line(sm[0])
				checkLineSet[lineNr] = true
			}

			for _, sm := range c.CurrentRE.FindAllSubmatchIndex(f.Text, -1) {
				lineNr := locLine.Line(sm[0])
				if _, ok := checkLineSet[lineNr]; ok {
					continue
				}

				f.HasCurrents = true

				version := string(f.Text[sm[2]:sm[3]])
				c.Currents = append(c.Currents, Current{
					File:    f,
					LineNr:  lineNr,
					Range:   [2]int{sm[2], sm[3]},
					Version: version,
				})
			}
		}
	}
}

// Lint configuration
func (fs *FileSet) Lint() []error {
	var errs []error

	for _, c := range fs.Checks {
		if len(c.Currents) != 0 {
			continue
		}
		errs = append(errs, fmt.Errorf("%s:%d: %s has no current version matches", c.File.Name, c.PipelineLineNr, c.Name))
	}

	for _, f := range fs.Files {
		if f.HasNoVersions {
			if f.HasConfig {
				continue
			}
			errs = append(errs, fmt.Errorf("%s: has no configuration", f.Name))
		} else {
			if f.HasConfig || f.HasCurrents {
				continue
			}
			errs = append(errs, fmt.Errorf("%s: has no configuration or current version matches", f.Name))
		}
	}

	for _, ca := range fs.Checks {
		for _, cca := range ca.Currents {
			for _, cb := range fs.Checks {
				if ca == cb {
					continue
				}

				for _, ccb := range cb.Currents {
					if cca.File.Name != ccb.File.Name ||
						!rangeOverlap(cca.Range[0], cca.Range[1], ccb.Range[0], ccb.Range[1]) {
						continue
					}

					errs = append(errs, fmt.Errorf("%s:%d:%s has overlapping matches with %s:%d:%s at %s:%d",
						ca.File.Name, ca.PipelineLineNr, ca.Name,
						cb.File.Name, cb.PipelineLineNr, cb.Name,
						cca.File.Name, cca.LineNr))
				}
			}
		}
	}

	return errs
}

// Replace current with latest versions in text
func (fs *FileSet) Replace(file *File) []byte {
	if file.HasNoVersions {
		return file.Text
	}

	locLine := locline.New(file.Text)
	checkLineSet := map[int]bool{}
	for _, sm := range bumpRe.FindAllSubmatchIndex(file.Text, -1) {
		lineNr := locLine.Line(sm[0])
		checkLineSet[lineNr] = true
	}

	selectedChecks := fs.SelectedChecks()
	var replacers []rereplacer.Replace
	for _, c := range selectedChecks {
		// skip if check has run commands
		if len(c.CommandShells) > 0 {
			continue
		}

		// new variable for the replacer fn closure
		c := c
		replacers = append(replacers, rereplacer.Replace{
			Re: c.CurrentRE,
			Fn: func(b []byte, sm []int) []byte {
				matchLine := locLine.Line(sm[0])
				if _, ok := checkLineSet[matchLine]; ok {
					return b[sm[0]:sm[1]]
				}

				l := []byte{}
				l = append(l, b[sm[0]:sm[2]]...)
				l = append(l, []byte(c.Latest)...)
				l = append(l, b[sm[3]:sm[1]]...)

				return l
			},
		})
	}

	return rereplacer.Replacer(replacers).Replace(file.Text)
}

func (fs *FileSet) CommandEnv(check *Check) []string {
	return []string{
		fmt.Sprintf("NAME=%s", check.Name),
		fmt.Sprintf("LATEST=%s", check.Latest),
	}
}

func (fs *FileSet) parseFile(file *File, filters []filter.NamedFilter) error {
	locLine := locline.New(file.Text)

	for _, sm := range bumpRe.FindAllSubmatchIndex(file.Text, -1) {
		lineNr := locLine.Line(sm[0])
		checkLine := strings.TrimSpace(string(file.Text[sm[2]:sm[3]]))
		err := fs.parseCheckLine(file, lineNr, checkLine, filters)
		if err != nil {
			return fmt.Errorf("%s:%d: %w", file.Name, lineNr, err)
		}
	}

	return nil
}

func (fs *FileSet) findCheckByName(name string) *Check {
	for _, c := range fs.Checks {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func (fs *FileSet) parseCheckLine(file *File, lineNr int, line string, filters []filter.NamedFilter) error {
	file.HasConfig = true

	nameRest := strings.SplitN(line, " ", 2)
	if len(nameRest) != 2 {
		return fmt.Errorf("invalid name and arguments: %q", line)
	}

	name, rest := strings.TrimSpace(nameRest[0]), nameRest[1]
	if name == "" {
		return fmt.Errorf("name is empty")
	}

	if strings.HasPrefix(rest, "/") {
		// bump: <name> /<re>/ <pipeline>
		var currentReStr string
		var pipelineStr string

		tokens := []lexer.Token{
			{Name: "re", Dest: &currentReStr, Fn: lexer.Quoted(`/`)},
			{Fn: lexer.Re(regexp.MustCompile(`\s`))},
			{Name: "pipeline", Dest: &pipelineStr, Fn: lexer.Rest(1)},
		}

		if _, err := lexer.Tokenize(rest, tokens); err != nil {
			return err
		}
		pl, err := pipeline.New(filters, pipelineStr)
		if err != nil {
			return fmt.Errorf("%s: %w", pipelineStr, err)
		}
		currentRe, err := regexp.Compile(currentReStr)
		if err != nil {
			return fmt.Errorf("invalid current version regexp: %q", currentReStr)
		}
		if currentRe.NumSubexp() != 1 {
			return fmt.Errorf("regexp must have one submatch: %q", currentReStr)
		}

		check := &Check{
			File:           file,
			Name:           name,
			CurrentRE:      currentRe,
			PipelineLineNr: lineNr,
			Pipeline:       pl,
		}

		for _, bc := range fs.Checks {
			if check.Name == bc.Name {
				return fmt.Errorf("%s already used at %s:%d",
					check.Name, bc.File.Name, bc.PipelineLineNr)
			}
		}

		fs.Checks = append(fs.Checks, check)

		return nil
	} else {
		check := fs.findCheckByName(name)
		if check == nil {
			return fmt.Errorf("%s has not been defined yet", name)
		}

		kindReset := strings.SplitN(rest, " ", 2)
		if len(kindReset) != 2 {
			return fmt.Errorf("invalid kind and arguments: %q", line)
		}
		kind, rest := strings.TrimSpace(kindReset[0]), kindReset[1]

		switch kind {
		case "command":
			// bump: <name> command ...
			check.CommandShells = append(check.CommandShells, CheckShell{
				Cmd:    rest,
				File:   file,
				LineNr: lineNr,
			})
		case "after":
			// bump: <name> after ...
			check.AfterShells = append(check.AfterShells, CheckShell{
				Cmd:    rest,
				File:   file,
				LineNr: lineNr,
			})
		case "link":
			// bump: <name> link <link-name> <link>

			var linkName string
			var linkURL string
			tokens := []lexer.Token{
				{Name: "name", Dest: &linkName, Fn: lexer.Quoted(`"`)},
				{Fn: lexer.Re(regexp.MustCompile(`\s`))},
				{Name: "URL", Dest: &linkURL, Fn: lexer.Rest(1)},
			}

			if _, err := lexer.Tokenize(rest, tokens); err != nil {
				return err
			}

			check.Links = append(check.Links, CheckLink{Name: linkName, URL: linkURL})
		default:
			return fmt.Errorf("expected command, after or link: %q", line)
		}
	}

	return nil
}
