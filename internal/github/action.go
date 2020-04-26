package github

import (
	"fmt"
	"strings"
)

// GetenvFn function to return environment values
type GetenvFn func(name string) string

// ActionEnv is a GitHub action environment
// https://help.github.com/en/articles/virtual-environments-for-github-actions#default-environment-variables
type ActionEnv struct {
	getenv     GetenvFn
	Client     *Client
	Workflow   string   // GITHUB_WORKFLOW The name of the workflow.
	Action     string   // GITHUB_ACTION The name of the action.
	Actor      string   // GITHUB_ACTOR The name of the person or app that initiated the workflow. For example, octocat.
	EventName  string   // GITHUB_EVENT_NAME The name of the webhook event that triggered the workflow.
	EventPath  string   // GITHUB_EVENT_PATH The path of the file with the complete webhook event payload. For example, /github/workflow/event.json.
	Workspace  string   // GITHUB_WORKSPACE The GitHub workspace directory path. The workspace directory contains a subdirectory with a copy of your repository if your workflow uses the actions/checkout action. If you don't use the actions/checkout action, the directory will be empty. For example, /home/runner/work/my-repo-name/my-repo-name.
	SHA        string   // GITHUB_SHA The commit SHA that triggered the workflow. For example, ffac537e6cbbf934b08745a378932722df287a53.
	Ref        string   // GITHUB_REF The branch or tag ref that triggered the workflow. For example, refs/heads/feature-branch-1. If neither a branch or tag is available for the event type, the variable will not exist.
	HeadRef    string   // GITHUB_HEAD_REF Only set for forked repositories. The branch of the head repository.
	BaseRef    string   // GITHUB_BASE_REF Only set for forked repositories. The branch of the base repository.
	Repository string   // GITHUB_REPOSITORY user/repo
	Owner      string   // user (extracted from Repository)
	RepoName   string   // repo (extracted from Repository)
	RepoRef    *RepoRef // *RepoRef variant of Repository
}

// NewActionEnv creates a new ActionEnv
func NewActionEnv(getenv GetenvFn, version string) (*ActionEnv, error) {
	getenvOrErr := func(name string) (string, error) {
		v := getenv(name)
		if v == "" {
			return "", fmt.Errorf("%s not set", name)
		}
		return v, nil
	}

	token, err := getenvOrErr("GITHUB_TOKEN")
	if err != nil {
		return nil, err
	}
	workflow, err := getenvOrErr("GITHUB_WORKFLOW")
	if err != nil {
		return nil, err
	}
	action, err := getenvOrErr("GITHUB_ACTION")
	if err != nil {
		return nil, err
	}
	actor, err := getenvOrErr("GITHUB_ACTOR")
	if err != nil {
		return nil, err
	}
	repository, err := getenvOrErr("GITHUB_REPOSITORY")
	if err != nil {
		return nil, err
	}
	repositoryParts := strings.SplitN(repository, "/", 2)

	client := &Client{
		Token:   token,
		Version: version,
	}

	return &ActionEnv{
		getenv:     getenv,
		Client:     client,
		Workflow:   workflow,
		Action:     action,
		Actor:      actor,
		EventName:  getenv("GITHUB_EVENT_NAME"),
		EventPath:  getenv("GITHUB_EVENT_PATH"),
		Workspace:  getenv("GITHUB_WORKSPACE"),
		SHA:        getenv("GITHUB_SHA"),
		Ref:        getenv("GITHUB_REF"),
		HeadRef:    getenv("GITHUB_HEAD_REF"),
		BaseRef:    getenv("GITHUB_BASE_REF"),
		Repository: repository,
		Owner:      repositoryParts[0],
		RepoName:   repositoryParts[1],
		RepoRef:    client.NewRepoRef(repository),
	}, nil
}

// Input returns value of input variable as defined in action.yml
func (a *ActionEnv) Input(name string) (string, error) {
	envName := "INPUT_" + strings.ToUpper(name)
	v := a.getenv(envName)
	if v == "" {
		return "", fmt.Errorf("%s (%s) is empty", name, envName)
	}
	return v, nil
}
