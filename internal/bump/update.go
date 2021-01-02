package bump

import (
	"bytes"

	"github.com/pmezard/go-difflib/difflib"
)

type VersionChange struct {
	Check    *Check
	Currents []Current
}

type FileChange struct {
	File    *File
	NewText string
	Diff    string
}

type RunShell struct {
	Check *Check
	Cmd   string
	Env   []string
}

type Actions struct {
	VersionChanges []VersionChange
	FileChanges    []FileChange
	RunShells      []RunShell
}

func (fs *FileSet) UpdateActions() (Actions, []error) {
	if errs := fs.Latest(); errs != nil {
		return Actions{}, errs
	}

	a := Actions{}

	for _, check := range fs.SelectedChecks() {
		var currentChanges []Current
		for _, c := range check.Currents {
			if c.Version == check.Latest {
				continue
			}
			currentChanges = append(currentChanges, c)
		}

		if len(currentChanges) == 0 {
			continue
		}

		a.VersionChanges = append(a.VersionChanges, VersionChange{
			Check:    check,
			Currents: currentChanges,
		})

		env := fs.CommandEnv(check)
		// TODO: refactor, currently Replace skips if there are CommandRuns
		for _, cr := range check.CommandShells {
			a.RunShells = append(a.RunShells, RunShell{Check: check, Cmd: cr.Cmd, Env: env})
		}
		for _, cr := range check.AfterShells {
			a.RunShells = append(a.RunShells, RunShell{Check: check, Cmd: cr.Cmd, Env: env})
		}
	}

	for _, f := range fs.Files {
		newTextBuf := fs.Replace(f)
		// might return equal even if version has changed if checks has CommandRuns
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
			return Actions{}, []error{err}
		}

		a.FileChanges = append(a.FileChanges, FileChange{
			File:    f,
			NewText: newText,
			Diff:    udiff,
		})
	}

	return a, nil
}
