package githubaction

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/wader/bump/internal/bump"
	"github.com/wader/bump/internal/filter/all"
	"github.com/wader/bump/internal/github"
)

func runCmds(argss [][]string) error {
	for _, args := range argss {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = os.Environ()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Printf("> %s\n", cmd.String())
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// Run bump in a github action environment
func Run(version string) []error {
	ae, err := github.NewActionEnv(os.Getenv, version)
	if err != nil {
		return []error{err}
	}
	// TODO: used in tests
	ae.Client.BaseURL = os.Getenv("GITHUB_API_URL")

	if _, err := exec.LookPath("git"); err != nil {
		return []error{err}
	}

	if ae.SHA == "" {
		return []error{fmt.Errorf("GITHUB_SHA not set")}
	}

	bumpFiles, err := ae.Input("bump_files")
	if err != nil {
		return []error{err}
	}
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
	err = runCmds([][]string{
		{"git", "config", "--global", "user.name", userName},
		{"git", "config", "--global", "user.email", userEmail},
		{"git", "remote", "set-url", "--push", "origin", pushURL},
	})
	if err != nil {
		return []error{err}
	}

	// TODO: whitespace in filenames
	bumpFilesParts := strings.Fields(bumpFiles)
	bfs, errs := bump.NewBumpFileSet(all.Filters(), ioutil.ReadFile, bumpFilesParts)
	if errs != nil {
		return errs
	}

	errs = bfs.Latest()
	if errs != nil {
		return errs
	}

	for _, c := range bfs.Checks {
		fmt.Printf("Checking %s\n", c.Name)

		if !c.HasUpdate() {
			fmt.Printf("  No updates\n")

			// TODO: close if PR is open?
			continue
		}

		fmt.Printf("  Updateable to %s\n", c.Latest)

		templateReplacer := strings.NewReplacer(
			"$name", c.Name,
			"$version", c.Latest,
		)

		branchName := templateReplacer.Replace(branchTemplate)
		if err := github.IsValidBranchName(branchName); err != nil {
			return []error{fmt.Errorf("branch name %q is invalid: %w", branchName, err)}
		}

		prs, err := ae.RepoRef.ListPullRequest("state", "open", "head", ae.Owner+":"+branchName)
		if err != nil {
			return []error{err}
		}

		// there is already an open PR for this update
		if len(prs) > 0 {
			fmt.Printf("  Open PR %d already exists\n", prs[0].Number)

			// TODO: do get pull request and check for mergable and rerun/close if needed?
			continue
		}

		// reset HEAD back to triggering commit before each PR
		err = runCmds([][]string{{"git", "reset", "--hard", ae.SHA}})
		if err != nil {
			return []error{err}
		}

		// only concider this check when replacing
		bfs.SkipCheckFn = func(skipC *bump.Check) bool {
			return skipC.Name != c.Name
		}

		for _, f := range bfs.Files {
			newTextBuf := bfs.Replace(f.Text)
			if bytes.Compare(f.Text, newTextBuf) == 0 {
				continue
			}
			if err := ioutil.WriteFile(f.Name, []byte(newTextBuf), 0644); err != nil {
				return []error{err}
			}

			fmt.Printf("  Wrote change to %s\n", f.Name)
		}

		title := templateReplacer.Replace(titleTemplate)
		err = runCmds([][]string{
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
