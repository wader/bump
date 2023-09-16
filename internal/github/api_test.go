package github_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/wader/bump/internal/github"
)

// TODO: better tests?

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (r RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}

func responseClient(fn func(*http.Request) (interface{}, int)) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			v, code := fn(req)
			var b []byte
			if v != nil {
				var err error
				b, err = json.Marshal(v)
				if err != nil {
					panic(err)
				}
			}
			return &http.Response{
				StatusCode: code,
				Body:       io.NopCloser(bytes.NewReader(b)),
			}, nil
		}),
	}
}

func TestHeaders(t *testing.T) {
	gotCalled := false
	c := &github.Client{
		Token:   "abc",
		Version: "test",
		HTTPClient: responseClient(func(req *http.Request) (interface{}, int) {
			gotCalled = true
			type c struct {
				UserAgent     string
				Accept        string
				Authorization string
				ContentType   string
			}

			expectedC := c{
				UserAgent:     "https://github.com/wader/bump test",
				Accept:        "application/vnd.github.v3+json",
				Authorization: "token abc",
				ContentType:   "application/json",
			}
			actualC := c{
				UserAgent:     req.UserAgent(),
				Accept:        req.Header.Get("Accept"),
				Authorization: req.Header.Get("Authorization"),
				ContentType:   req.Header.Get("Content-Type"),
			}

			if expectedC != actualC {
				t.Errorf("expected %#v, got %#v", expectedC, actualC)
			}

			return nil, 200
		}),
	}
	_, _ = c.NewRepoRef("user/repo").CreatePullRequest(github.NewPullRequest{})
	if !gotCalled {
		t.Error("did not get called")
	}
}

func TestListPullRequest(t *testing.T) {
	expectedPRs := []github.PullRequest{
		{ID: 123, Number: 1, Title: "PR title 1"},
		{ID: 456, Number: 2, Title: "PR title 2"},
	}

	c := &github.Client{
		Token: "abc",
		HTTPClient: responseClient(func(req *http.Request) (interface{}, int) {
			type c struct {
				Method     string
				ParamState string
				Path       string
			}

			expectedC := c{
				Method:     "GET",
				ParamState: "closed",
				Path:       "/repos/user/repo/pulls",
			}
			actualC := c{
				Method:     req.Method,
				ParamState: req.URL.Query().Get("state"),
				Path:       req.URL.Path,
			}

			if expectedC != actualC {
				t.Errorf("expected %#v, got %#v", expectedC, actualC)
			}

			return expectedPRs, 200
		}),
	}

	actualPRs, err := c.NewRepoRef("user/repo").ListPullRequest("state", "closed")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(expectedPRs, actualPRs) {
		t.Errorf("expected PRs %#v, got %#v", expectedPRs, actualPRs)
	}
}

func TestCreatePullRequest(t *testing.T) {
	expectedNewPR := github.NewPullRequest{
		Title:               "a",
		Head:                "b",
		Base:                "c",
		Body:                github.StrR("d"),
		MaintainerCanModify: github.BoolR(true),
		Draft:               github.BoolR(true),
	}
	expectedPR := github.PullRequest{
		Title:               "a",
		Head:                github.Ref{Ref: "b"},
		Base:                github.Ref{Ref: "c"},
		Body:                "d",
		MaintainerCanModify: true,
		Draft:               true,
	}

	c := &github.Client{
		Token: "abc",
		HTTPClient: responseClient(func(req *http.Request) (interface{}, int) {
			type r struct {
				Method string
				Path   string
				NewPR  github.NewPullRequest
			}
			expectedR := r{
				Method: "POST",
				Path:   "/repos/user/repo/pulls",
				NewPR:  expectedNewPR,
			}
			actualR := r{
				Method: req.Method,
				Path:   req.URL.Path,
			}
			_ = json.NewDecoder(req.Body).Decode(&actualR.NewPR)

			if !reflect.DeepEqual(expectedR, actualR) {
				t.Errorf("expected:\n%#v\ngot:\n%#v\n", expectedR, actualR)
			}

			return expectedPR, 200
		}),
	}

	actualPR, err := c.NewRepoRef("user/repo").CreatePullRequest(expectedNewPR)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(expectedPR, actualPR) {
		t.Errorf("expected PR %#v, got %#v", expectedPR, actualPR)
	}
}

func TestUpdatePullRequest(t *testing.T) {
	expectedUpdatePR := github.UpdatePullRequest{
		Title:               github.StrR("a"),
		Base:                github.StrR("c"),
		Body:                github.StrR("d"),
		State:               github.StrR("closed"),
		MaintainerCanModify: github.BoolR(true),
	}
	expectedPR := github.PullRequest{
		Title:               "a",
		Head:                github.Ref{Ref: "b"},
		Base:                github.Ref{Ref: "c"},
		Body:                "d",
		MaintainerCanModify: true,
		Draft:               true,
	}

	c := &github.Client{
		Token: "abc",
		HTTPClient: responseClient(func(req *http.Request) (interface{}, int) {
			type r struct {
				Method   string
				Path     string
				UpdatePR github.UpdatePullRequest
			}
			expectedR := r{
				Method:   "PATCH",
				Path:     "/repos/user/repo/pulls/123",
				UpdatePR: expectedUpdatePR,
			}
			actualR := r{
				Method: req.Method,
				Path:   req.URL.Path,
			}
			_ = json.NewDecoder(req.Body).Decode(&actualR.UpdatePR)

			if !reflect.DeepEqual(expectedR, actualR) {
				t.Errorf("expected:\n%#v\ngot:\n%#v\n", expectedR, actualR)
			}

			return expectedPR, 200
		}),
	}

	actualPR, err := c.NewRepoRef("user/repo").UpdatePullRequest(123, expectedUpdatePR)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(expectedPR, actualPR) {
		t.Errorf("expected PR %#v, got %#v", expectedPR, actualPR)
	}
}

func TestCreateComment(t *testing.T) {
	expectedNewComment := github.NewComment{
		Body: "a",
	}
	expectedComment := github.Comment{
		Body: "a",
	}

	c := &github.Client{
		Token: "abc",
		HTTPClient: responseClient(func(req *http.Request) (interface{}, int) {
			type r struct {
				Method     string
				Path       string
				NewComment github.NewComment
			}
			expectedR := r{
				Method:     "POST",
				Path:       "/repos/user/repo/issues/123/comments",
				NewComment: expectedNewComment,
			}
			actualR := r{
				Method: req.Method,
				Path:   req.URL.Path,
			}
			_ = json.NewDecoder(req.Body).Decode(&actualR.NewComment)

			if !reflect.DeepEqual(expectedR, actualR) {
				t.Errorf("expected:\n%#v\ngot:\n%#v\n", expectedR, actualR)
			}

			return expectedComment, 200
		}),
	}

	actualComment, err := c.NewRepoRef("user/repo").CreateComment(123, expectedNewComment)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(expectedComment, actualComment) {
		t.Errorf("expected PR %#v, got %#v", expectedComment, actualComment)
	}
}

func TestIsValidBranchName(t *testing.T) {
	testCases := []struct {
		s string
		e string
	}{
		{``, "can't be empty"},
		{`.a`, "can't start with '.'"},
		{`a/`, "can't end with '/'"},
		{`a.lock`, "can't end with '.lock'"},
		{`/a/`, "can't end with '/'"},
		{`~`, `can't include any of '~^: \@?*{}[]'`},
		{`^`, `can't include any of '~^: \@?*{}[]'`},
		{`:`, `can't include any of '~^: \@?*{}[]'`},
		{` `, `can't include any of '~^: \@?*{}[]'`},
		{`\`, `can't include any of '~^: \@?*{}[]'`},
		{`@`, `can't include any of '~^: \@?*{}[]'`},
		{`?`, `can't include any of '~^: \@?*{}[]'`},
		{`*`, `can't include any of '~^: \@?*{}[]'`},
		{`{`, `can't include any of '~^: \@?*{}[]'`},
		{`}`, `can't include any of '~^: \@?*{}[]'`},
		{`[`, `can't include any of '~^: \@?*{}[]'`},
		{`]`, `can't include any of '~^: \@?*{}[]'`},
		{"\x00", "can't include control characters"},
		{"\x01", "can't include control characters"},
		{"\x02", "can't include control characters"},
		{"\x03", "can't include control characters"},
		{"\x04", "can't include control characters"},
		{"\x05", "can't include control characters"},
		{"\x06", "can't include control characters"},
		{"\x07", "can't include control characters"},
		{"\x08", "can't include control characters"},
		{"\x09", "can't include control characters"},
		{"\x0a", "can't include control characters"},
		{"\x0b", "can't include control characters"},
		{"\x0c", "can't include control characters"},
		{"\x0d", "can't include control characters"},
		{"\x0e", "can't include control characters"},
		{"\x0f", "can't include control characters"},
		{"\x10", "can't include control characters"},
		{"\x11", "can't include control characters"},
		{"\x12", "can't include control characters"},
		{"\x13", "can't include control characters"},
		{"\x14", "can't include control characters"},
		{"\x15", "can't include control characters"},
		{"\x16", "can't include control characters"},
		{"\x17", "can't include control characters"},
		{"\x18", "can't include control characters"},
		{"\x19", "can't include control characters"},
		{"\x1a", "can't include control characters"},
		{"\x1b", "can't include control characters"},
		{"\x1c", "can't include control characters"},
		{"\x1d", "can't include control characters"},
		{"\x1e", "can't include control characters"},
		{"\x1f", "can't include control characters"},
		{"\x7f", "can't include control characters"},
		{`a`, ""},
		{`ab/cd/ab-cd-ab.cd_ab_cd`, ""},
	}
	for _, tC := range testCases {
		t.Run(tC.s, func(t *testing.T) {
			actual := github.IsValidBranchName(tC.s)
			expected := tC.e

			if expected == "" {
				if actual != nil {
					t.Errorf("expected nil got %s", actual.Error())

				}
			} else {
				if actual == nil {
					t.Errorf("expected %s got nil", expected)
				} else if expected != actual.Error() {
					t.Errorf("expected %s got %s", expected, actual.Error())
				}
			}
		})
	}
}

// // user to do test requests durinv dev
// func TestRealAPI(t *testing.T) {
// 	c := &Client{
// 		Token:   "",
// 		Version: "test",
// 	}
// 	prs, err := c.NewRepoRef("user/repo").ListPullRequest()
// 	log.Printf("err: %#+v\n", err)
// 	log.Printf("prs: %#+v\n", prs)
// }
