package githubaction

import (
	"fmt"
	"strings"

	"github.com/wader/bump/internal/bump"
	"github.com/wader/bump/internal/filter/all"
	"github.com/wader/bump/internal/github"
)

// CheckTemplateReplaceFn buils a function for doing template replacing for check
func CheckTemplateReplaceFn(c *bump.Check) func(s string) string {
	var currentVersions []string
	for _, c := range c.Currents {
		currentVersions = append(currentVersions, c.Version)
	}

	r := strings.NewReplacer(
		"$NAME", c.Name,
		"$LATEST", c.Latest,
		"$CURRENT", strings.Join(currentVersions, ", "),
	)

	return func(s string) string {
		return r.Replace(s)
	}
}

// Command is a github action interface to bump packages
type Command struct {
	Version string
	OS      bump.OS
}

// Run bump in a github action environment
func (cmd Command) Run() []error {
	errs := cmd.run()
	for _, err := range errs {
		fmt.Fprintln(cmd.OS.Stderr(), err)
	}

	return errs
}

func (cmd Command) runExecs(argss [][]string) error {
	for _, args := range argss {
		fmt.Printf("> %s\n", strings.Join(args, " "))
		if err := cmd.OS.Exec(args, nil); err != nil {
			return err
		}
	}
	return nil
}

func (cmd Command) run() []error {
	ae, err := github.NewActionEnv(cmd.OS.Getenv, cmd.Version)
	if err != nil {
		return []error{err}
	}
	// TODO: used in tests
	ae.Client.BaseURL = cmd.OS.Getenv("GITHUB_API_URL")

	if ae.SHA == "" {
		return []error{fmt.Errorf("GITHUB_SHA not set")}
	}

	bumpfile, err := ae.Input("bumpfile")
	if err != nil {
		return []error{err}
	}
	files, _ := ae.Input("bump_files")
	titleTemplate, err := ae.Input("title_template")
	if err != nil {
		return []error{err}
	}
	branchTemplate, err := ae.Input("branch_template")
	if err != nil {
		return []error{err}
	}
	userName, err := ae.Input("user_name")
	if err != nil {
		return []error{err}
	}
	userEmail, err := ae.Input("user_email")
	if err != nil {
		return []error{err}
	}

	pushURL := fmt.Sprintf("https://%s:%s@github.com/%s.git", ae.Actor, ae.Client.Token, ae.Repository)
	err = cmd.runExecs([][]string{
		{"git", "config", "--global", "user.name", userName},
		{"git", "config", "--global", "user.email", userEmail},
		{"git", "remote", "set-url", "--push", "origin", pushURL},
	})
	if err != nil {
		return []error{err}
	}

	// TODO: whitespace in filenames
	filesParts := strings.Fields(files)
	bfs, errs := bump.NewBumpFileSet(cmd.OS, all.Filters(), bumpfile, filesParts)
	if errs != nil {
		return errs
	}

	for _, c := range bfs.Checks {
		// only concider this check for update actions
		bfs.SkipCheckFn = func(skipC *bump.Check) bool {
			return skipC.Name != c.Name
		}

		ua, errs := bfs.UpdateActions()
		if errs != nil {
			return errs
		}

		fmt.Printf("Checking %s\n", c.Name)

		if !c.HasUpdate() {
			fmt.Printf("  No updates\n")

			// TODO: close if PR is open?
			continue
		}

		fmt.Printf("  Updateable to %s\n", c.Latest)

		templateReplacerFn := CheckTemplateReplaceFn(c)

		branchName := templateReplacerFn(branchTemplate)
		if err := github.IsValidBranchName(branchName); err != nil {
			return []error{fmt.Errorf("branch name %q is invalid: %w", branchName, err)}
		}

		prs, err := ae.RepoRef.ListPullRequest("state", "all", "head", ae.Owner+":"+branchName)
		if err != nil {
			return []error{err}
		}

		// there is already an open or closed PR for this update
		if len(prs) > 0 {
			fmt.Printf("  Open or closed PR %d %s already exists\n",
				prs[0].Number, ae.Owner+":"+branchName)

			// TODO: do get pull request and check for mergable and rerun/close if needed?
			continue
		}

		// reset HEAD back to triggering commit before each PR
		err = cmd.runExecs([][]string{{"git", "reset", "--hard", ae.SHA}})
		if err != nil {
			return []error{err}
		}

		for _, fc := range ua.FileChanges {
			if err := cmd.OS.WriteFile(fc.File.Name, []byte(fc.NewText)); err != nil {
				return []error{err}
			}

			fmt.Printf("  Wrote change to %s\n", fc.File.Name)
		}

		for _, rs := range ua.RunShells {
			if err := cmd.OS.Shell(rs.Cmd, rs.Env); err != nil {
				return []error{fmt.Errorf("%s: shell: %s: %w", rs.Check.Name, rs.Cmd, err)}
			}
		}

		title := templateReplacerFn(titleTemplate)
		err = cmd.runExecs([][]string{
			{"git", "diff"},
			{"git", "add", "--update"},
			{"git", "commit", "--message", title},
			// force so if for some reason there was an existing closed update PR with the same name
			{"git", "push", "--force", "origin", "HEAD:refs/heads/" + branchName},
		})
		if err != nil {
			return []error{err}
		}

		fmt.Printf("  Commited and pushed\n")

		newPr, err := ae.RepoRef.CreatePullRequest(github.NewPullRequest{
			Base:  ae.Ref,
			Head:  ae.Owner + ":" + branchName,
			Title: title,
		})
		if err != nil {
			return []error{err}
		}

		fmt.Printf("  Created PR %s\n", newPr.URL)
	}

	return nil
}
