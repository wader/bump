package githubaction

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/wader/bump/internal/bump"
	"github.com/wader/bump/internal/filter/all"
	"github.com/wader/bump/internal/github"
	sx "github.com/wader/bump/internal/slicex"
)

// CheckTemplateReplaceFn builds a function for doing template replacing for check
func CheckTemplateReplaceFn(c *bump.Check) func(s string) (string, error) {
	varReplacer := strings.NewReplacer(
		"$NAME", c.Name,
		"$LATEST", c.Latest,
		// TODO: this might be wrong if there are multiple current versions
		"$CURRENT", c.Currents[0].Version,
	)

	currentVersions := sx.Unique(sx.Map(c.Currents, func(c bump.Current) string {
		return c.Version
	}))
	messages := sx.Map(c.Messages, func(m bump.CheckMessage) string {
		return varReplacer.Replace(m.Message)
	})
	type link struct {
		Title string
		URL   string
	}
	links := sx.Map(c.Links, func(l bump.CheckLink) link {
		return link{
			Title: varReplacer.Replace(l.Title),
			URL:   varReplacer.Replace(l.URL),
		}
	})

	tmplData := struct {
		Name     string
		Current  []string
		Messages []string
		Latest   string
		Links    []link
	}{
		Name:     c.Name,
		Current:  currentVersions,
		Messages: messages,
		Latest:   c.Latest,
		Links:    links,
	}

	return func(s string) (string, error) {
		tmpl := template.New("")
		tmpl = tmpl.Funcs(template.FuncMap{
			"join": strings.Join,
		})
		tmpl, err := tmpl.Parse(s)
		if err != nil {
			return "", err
		}

		execBuf := &bytes.Buffer{}
		err = tmpl.Execute(execBuf, tmplData)
		if err != nil {
			return "", err
		}

		return execBuf.String(), nil
	}
}

// Command is a github action interface to bump packages
type Command struct {
	Version string
	OS      bump.OS
}

// Run bump in a github action environment
func (c Command) Run() []error {
	errs := c.run()
	for _, err := range errs {
		fmt.Fprintln(c.OS.Stderr(), err)
	}

	return errs
}

func (c Command) execs(argss [][]string) error {
	for _, args := range argss {
		fmt.Printf("exec> %s\n", strings.Join(args, " "))
		if err := c.OS.Exec(args, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c Command) shell(cmd string, env []string) error {
	fmt.Printf("shell> %s %s\n", strings.Join(env, " "), cmd)
	if err := c.OS.Shell(cmd, env); err != nil {
		return err
	}
	return nil
}

func (c Command) run() []error {
	ae, err := github.NewActionEnv(c.OS.Getenv, c.Version)
	if err != nil {
		return []error{err}
	}
	// TODO: used in tests
	ae.Client.BaseURL = c.OS.Getenv("GITHUB_API_URL")

	if ae.SHA == "" {
		return []error{fmt.Errorf("GITHUB_SHA not set")}
	}

	// support "bump_files" for backward compatibility
	bumpFiles, _ := ae.Input("bump_files")
	files, _ := ae.Input("files")
	var bumpfile,
		titleTemplate,
		commitBodyTemplate,
		prBodyTemplate,
		branchTemplate,
		userName,
		userEmail string
	for _, v := range []struct {
		s *string
		n string
	}{
		{&bumpfile, "bumpfile"},
		{&titleTemplate, "title_template"},
		{&commitBodyTemplate, "commit_body_template"},
		{&prBodyTemplate, "pr_body_template"},
		{&branchTemplate, "branch_template"},
		{&userName, "user_name"},
		{&userEmail, "user_email"},
	} {
		s, err := ae.Input(v.n)
		if err != nil {
			return []error{err}
		}
		*v.s = s
	}

	pushURL := fmt.Sprintf("https://%s:%s@github.com/%s.git", ae.Actor, ae.Client.Token, ae.Repository)
	err = c.execs([][]string{
		// safe.directory workaround for CVE-2022-24765
		// https://github.blog/2022-04-12-git-security-vulnerability-announced/
		{"git", "config", "--global", "--add", "safe.directory", ae.Workspace},
		{"git", "config", "--global", "user.name", userName},
		{"git", "config", "--global", "user.email", userEmail},
		{"git", "remote", "set-url", "--push", "origin", pushURL},
	})
	if err != nil {
		return []error{err}
	}

	// TODO: whitespace in filenames
	var filenames []string
	filenames = append(filenames, strings.Fields(bumpFiles)...)
	filenames = append(filenames, strings.Fields(files)...)
	bfs, errs := bump.NewBumpFileSet(c.OS, all.Filters(), bumpfile, filenames)
	if errs != nil {
		return errs
	}

	for _, check := range bfs.Checks {
		// only consider this check for update actions
		bfs.SkipCheckFn = func(skipC *bump.Check) bool {
			return skipC.Name != check.Name
		}

		ua, errs := bfs.UpdateActions()
		if errs != nil {
			return errs
		}

		fmt.Printf("Checking %s\n", check.Name)

		if !check.HasUpdate() {
			fmt.Printf("  No updates\n")

			// TODO: close if PR is open?
			continue
		}

		fmt.Printf("  Updatable to %s\n", check.Latest)

		templateReplacerFn := CheckTemplateReplaceFn(check)

		branchName, err := templateReplacerFn(branchTemplate)
		if err != nil {
			return []error{fmt.Errorf("branch template error: %w", err)}
		}
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
		err = c.execs([][]string{{"git", "reset", "--hard", ae.SHA}})
		if err != nil {
			return []error{err}
		}

		for _, fc := range ua.FileChanges {
			if err := c.OS.WriteFile(fc.File.Name, []byte(fc.NewText)); err != nil {
				return []error{err}
			}

			fmt.Printf("  Wrote change to %s\n", fc.File.Name)
		}

		for _, rs := range ua.RunShells {
			if err := c.shell(rs.Cmd, rs.Env); err != nil {
				return []error{fmt.Errorf("%s: shell: %s: %w", rs.Check.Name, rs.Cmd, err)}
			}
		}

		title, err := templateReplacerFn(titleTemplate)
		if err != nil {
			return []error{fmt.Errorf("title template error: %w", err)}
		}
		commitBody, err := templateReplacerFn(commitBodyTemplate)
		if err != nil {
			return []error{fmt.Errorf("title template error: %w", err)}
		}
		prBody, err := templateReplacerFn(prBodyTemplate)
		if err != nil {
			return []error{fmt.Errorf("title template error: %w", err)}
		}

		err = c.execs([][]string{
			{"git", "diff"},
			{"git", "add", "--all"},
			{"git", "commit", "--message", title, "--message", commitBody},
			// force so if for some reason there was an existing closed update PR with the same name
			{"git", "push", "--force", "origin", "HEAD:refs/heads/" + branchName},
		})
		if err != nil {
			return []error{err}
		}

		fmt.Printf("  Committed and pushed\n")

		newPr, err := ae.RepoRef.CreatePullRequest(github.NewPullRequest{
			Base:  ae.Ref,
			Head:  ae.Owner + ":" + branchName,
			Title: title,
			Body:  &prBody,
		})
		if err != nil {
			return []error{err}
		}

		fmt.Printf("  Created PR %s\n", newPr.URL)
	}

	return nil
}
