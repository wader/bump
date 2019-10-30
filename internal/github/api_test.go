package github

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
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
				Body:       ioutil.NopCloser(bytes.NewReader(b)),
			}, nil
		}),
	}
}

func TestHeaders(t *testing.T) {
	gotCalled := false
	c := &Client{
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
	c.NewRepoRef("user/repo").CreatePullRequest(NewPullRequest{})
	if !gotCalled {
		t.Error("did not get called")
	}
}

func TestListPullRequest(t *testing.T) {
	expectedPRs := []PullRequest{
		{ID: 123, Number: 1, Title: "PR title 1"},
		{ID: 456, Number: 2, Title: "PR title 2"},
	}

	c := &Client{
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
	expectedNewPR := NewPullRequest{
		Title:               "a",
		Head:                "b",
		Base:                "c",
		Body:                StrR("d"),
		MaintainerCanModify: BoolR(true),
		Draft:               BoolR(true),
	}
	expectedPR := PullRequest{
		Title:               "a",
		Head:                Ref{Ref: "b"},
		Base:                Ref{Ref: "c"},
		Body:                "d",
		MaintainerCanModify: true,
		Draft:               true,
	}

	c := &Client{
		Token: "abc",
		HTTPClient: responseClient(func(req *http.Request) (interface{}, int) {
			type r struct {
				Method string
				Path   string
				NewPR  NewPullRequest
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
			json.NewDecoder(req.Body).Decode(&actualR.NewPR)

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
	expectedUpdatePR := UpdatePullRequest{
		Title:               StrR("a"),
		Base:                StrR("c"),
		Body:                StrR("d"),
		State:               StrR("closed"),
		MaintainerCanModify: BoolR(true),
	}
	expectedPR := PullRequest{
		Title:               "a",
		Head:                Ref{Ref: "b"},
		Base:                Ref{Ref: "c"},
		Body:                "d",
		MaintainerCanModify: true,
		Draft:               true,
	}

	c := &Client{
		Token: "abc",
		HTTPClient: responseClient(func(req *http.Request) (interface{}, int) {
			type r struct {
				Method   string
				Path     string
				UpdatePR UpdatePullRequest
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
			json.NewDecoder(req.Body).Decode(&actualR.UpdatePR)

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
	expectedNewComment := NewComment{
		Body: "a",
	}
	expectedComment := Comment{
		Body: "a",
	}

	c := &Client{
		Token: "abc",
		HTTPClient: responseClient(func(req *http.Request) (interface{}, int) {
			type r struct {
				Method     string
				Path       string
				NewComment NewComment
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
			json.NewDecoder(req.Body).Decode(&actualR.NewComment)

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
