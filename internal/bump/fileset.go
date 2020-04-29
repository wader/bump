// testing is currently done thru cli tests

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

// Check is a bump config line
type Check struct {
	File      *File
	Line      string
	LineNr    int
	Name      string
	Pipeline  pipeline.Pipeline
	CurrentRE *regexp.Regexp
	Latest    string
	Currents  []Current
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
	env Env,
	filters []filter.NamedFilter,
	bumpfile string,
	filenames []string) (*FileSet, []error) {

	b := &FileSet{
		Filters: filters,
	}

	if len(filenames) > 0 {
		for _, f := range filenames {
			if err := b.addFile(env, f); err != nil {
				return nil, []error{err}
			}
		}
	} else {
		if err := b.addBumpfile(env, bumpfile); err != nil {
			return nil, []error{err}
		}
	}

	b.findCurrent()

	if errs := b.Lint(); errs != nil {
		return nil, errs
	}

	return b, nil
}

func (b *FileSet) addBumpfile(env Env, name string) error {
	text, err := env.ReadFile(name)
	if err != nil {
		return err
	}
	file := &File{Name: name, Text: text, HasNoVersions: true}

	lineNr := 0
	var checks []*Check
	for _, l := range strings.Split(string(text), "\n") {
		lineNr++
		if strings.HasPrefix(l, "#") || strings.TrimSpace(l) == "" {
			continue
		}

		file.HasConfig = true

		matches, _ := env.Glob(l)
		if len(matches) > 0 {
			for _, m := range matches {
				if err := b.addFile(env, m); err != nil {
					return err
				}
			}
			continue
		}

		check, err := parseCheckLine(l, b.Filters)
		if err != nil {
			return fmt.Errorf("%s:%d: %w", file.Name, lineNr, err)
		}
		check.File = file
		check.LineNr = lineNr
		checks = append(checks, check)
	}

	return b.addChecks(file, checks)
}

func (b *FileSet) addFile(env Env, name string) error {
	text, err := env.ReadFile(name)
	if err != nil {
		return err
	}
	file := &File{Name: name, Text: text}

	checks, err := parseFile(file, b.Filters)
	if err != nil {
		return err
	}

	return b.addChecks(file, checks)
}

func (b *FileSet) addChecks(file *File, checks []*Check) error {
	for _, c := range checks {
		file.HasConfig = true

		for _, bc := range b.Checks {
			if c.Name == bc.Name {
				return fmt.Errorf("%s:%d: %s already used at %s:%d",
					c.File.Name, c.LineNr, c.Name, bc.File.Name, bc.LineNr)
			}
		}
		b.Checks = append(b.Checks, c)
	}

	b.Files = append(b.Files, file)

	return nil
}

// SelectedChecks returns selected checks based on SkipCheckFn
func (b *FileSet) SelectedChecks() []*Check {
	if b.SkipCheckFn == nil {
		return b.Checks
	}

	var filteredChecks []*Check
	for _, c := range b.Checks {
		if b.SkipCheckFn(c) {
			continue
		}
		filteredChecks = append(filteredChecks, c)
	}

	return filteredChecks
}

// Latest run all pipelines to populate latest versions
func (b *FileSet) Latest(resultFn func(check *Check, err error, duration time.Duration)) []error {
	type result struct {
		i        int
		latest   string
		err      error
		duration time.Duration
	}

	selectedChecks := b.SelectedChecks()
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
		c.Latest = r.latest
		if r.err != nil {
			errs = append(errs, fmt.Errorf("%s:%d: %s: %w", c.File.Name, c.LineNr, c.Name, r.err))
		}

		if resultFn != nil {
			resultFn(c, r.err, r.duration)
		}
	}

	return errs
}

func (b *FileSet) findCurrent() {
	for _, c := range b.SelectedChecks() {
		for _, f := range b.Files {
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
func (b *FileSet) Lint() []error {
	var errs []error

	for _, c := range b.Checks {
		if len(c.Currents) != 0 {
			continue
		}
		errs = append(errs, fmt.Errorf("%s:%d: %s has no current version matches", c.File.Name, c.LineNr, c.Name))
	}

	for _, f := range b.Files {
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

	for _, ca := range b.Checks {
		for _, cca := range ca.Currents {
			for _, cb := range b.Checks {
				if ca == cb {
					continue
				}

				for _, ccb := range cb.Currents {
					if cca.File.Name != ccb.File.Name ||
						!rangeOverlap(cca.Range[0], cca.Range[1], ccb.Range[0], ccb.Range[1]) {
						continue
					}

					errs = append(errs, fmt.Errorf("%s:%d:%s has overlapping matches with %s:%d:%s at %s:%d",
						ca.File.Name, ca.LineNr, ca.Name,
						cb.File.Name, cb.LineNr, cb.Name,
						cca.File.Name, cca.LineNr))
				}
			}
		}
	}

	return errs
}

// Replace current with latest versions in text
func (b *FileSet) Replace(file *File) []byte {
	if file.HasNoVersions {
		return file.Text
	}

	locLine := locline.New(file.Text)
	checkLineSet := map[int]bool{}
	for _, sm := range bumpRe.FindAllSubmatchIndex(file.Text, -1) {
		lineNr := locLine.Line(sm[0])
		checkLineSet[lineNr] = true
	}

	selectedChecks := b.SelectedChecks()
	var replacers []rereplacer.Replace
	for i := range selectedChecks {
		c := selectedChecks[i]
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

func parseFile(file *File, filters []filter.NamedFilter) ([]*Check, error) {
	var checks []*Check
	locLine := locline.New(file.Text)

	for _, sm := range bumpRe.FindAllSubmatchIndex(file.Text, -1) {
		lineNr := locLine.Line(sm[0])
		checkLine := strings.TrimSpace(string(file.Text[sm[2]:sm[3]]))
		check, err := parseCheckLine(checkLine, filters)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", file.Name, lineNr, err)
		}

		check.File = file
		check.LineNr = lineNr
		checks = append(checks, check)
	}

	return checks, nil
}

func parseCheckLine(line string, filters []filter.NamedFilter) (*Check, error) {
	var name,
		currentReStr,
		pipelineStr string
	var pl pipeline.Pipeline
	var err error

	tokens := []lexer.Token{
		{Name: "name", Dest: &name, Fn: lexer.Re(regexp.MustCompile(`[[:alnum:]-_+.]`))},
		{Fn: lexer.Re(regexp.MustCompile(`\s`))},
		{Name: "re", Dest: &currentReStr, Fn: lexer.Quoted(`/`)},
		{Fn: lexer.Re(regexp.MustCompile(`\s`))},
		{Name: "pipeline", Dest: &pipelineStr, Fn: lexer.Rest(1)},
	}

	if _, err := lexer.Tokenize(line, tokens); err != nil {
		return nil, err
	}
	pl, err = pipeline.New(filters, pipelineStr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", pipelineStr, err)
	}
	currentRe, err := regexp.Compile(currentReStr)
	if err != nil {
		return nil, fmt.Errorf("invalid current version regexp: %q", currentReStr)
	}
	if currentRe.NumSubexp() != 1 {
		return nil, fmt.Errorf("regexp must have one submatch: %q", currentReStr)
	}

	return &Check{
		Line:      line,
		Name:      name,
		CurrentRE: currentRe,
		Pipeline:  pl,
	}, nil
}
