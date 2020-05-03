package github_test

import (
	"testing"

	"github.com/wader/bump/internal/github"
)

func createGetEnvFn(env map[string]string) github.GetenvFn {
	return func(name string) string {
		return env[name]
	}
}

func expect(t *testing.T, actual, expected string) {
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

func TestIsActionEnv(t *testing.T) {
	if !github.IsActionEnv(createGetEnvFn(map[string]string{"GITHUB_ACTION": "action"})) {
		t.Fatal("should be action env")
	}
	if github.IsActionEnv(createGetEnvFn(map[string]string{})) {
		t.Fatal("should not be action env")
	}
}

func TestNewActionEnv(t *testing.T) {
	ae, err := github.NewActionEnv(createGetEnvFn(map[string]string{
		"GITHUB_TOKEN":      "token",
		"GITHUB_WORKFLOW":   "workflow",
		"GITHUB_ACTION":     "action",
		"GITHUB_ACTOR":      "actor",
		"GITHUB_EVENT_NAME": "event name",
		"GITHUB_EVENT_PATH": "event path",
		"GITHUB_WORKSPACE":  "workspace",
		"GITHUB_SHA":        "sha",
		"GITHUB_REF":        "refs/heads/master",
		"GITHUB_HEAD_REF":   "head ref",
		"GITHUB_BASE_REF":   "base ref",
		"GITHUB_REPOSITORY": "user/repo",
		"INPUT_A":           "a",
	}), "test")

	if err != nil {
		t.Fatal(err)
	}

	t.Run("Workflow", func(t *testing.T) { expect(t, ae.Workflow, "workflow") })
	t.Run("Action", func(t *testing.T) { expect(t, ae.Action, "action") })
	t.Run("Actor", func(t *testing.T) { expect(t, ae.Actor, "actor") })
	t.Run("EventName", func(t *testing.T) { expect(t, ae.EventName, "event name") })
	t.Run("EventPath", func(t *testing.T) { expect(t, ae.EventPath, "event path") })
	t.Run("Workspace", func(t *testing.T) { expect(t, ae.Workspace, "workspace") })
	t.Run("SHA", func(t *testing.T) { expect(t, ae.SHA, "sha") })
	t.Run("Ref", func(t *testing.T) { expect(t, ae.Ref, "refs/heads/master") })
	t.Run("HeadRef", func(t *testing.T) { expect(t, ae.HeadRef, "head ref") })
	t.Run("BaseRef", func(t *testing.T) { expect(t, ae.BaseRef, "base ref") })
	t.Run("Repository", func(t *testing.T) { expect(t, ae.Repository, "user/repo") })
	t.Run("Owner", func(t *testing.T) { expect(t, ae.Owner, "user") })
	t.Run("RepoName", func(t *testing.T) { expect(t, ae.RepoName, "repo") })
	t.Run("Input lowercase", func(t *testing.T) {
		actual, err := ae.Input("a")
		if err != nil {
			t.Error(err)
		}
		expect(t, actual, "a")
	})
	t.Run("Input uppercase", func(t *testing.T) {
		actual, err := ae.Input("A")
		if err != nil {
			t.Error(err)
		}
		expect(t, actual, "a")
	})
}
