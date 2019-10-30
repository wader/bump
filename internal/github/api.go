// Package github implements part of the GitHub REST API and Action functionality
package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DEFAULT_BASE_URL to GitHub REST API
const DEFAULT_BASE_URL = "https://api.github.com"

// StrR creates a string ref
func StrR(s string) *string {
	return &s
}

// BoolR creates a bool ref
func BoolR(b bool) *bool {
	return &b
}

// PullRequest is a GitHub Pull request fetched from the API
// from api docs converted using https://mholt.github.io/json-to-go/
type PullRequest struct {
	URL                 string    `json:"url"`
	ID                  int       `json:"id"`
	NodeID              string    `json:"node_id"`
	HTMLURL             string    `json:"html_url"`
	DiffURL             string    `json:"diff_url"`
	PatchURL            string    `json:"patch_url"`
	IssueURL            string    `json:"issue_url"`
	CommitsURL          string    `json:"commits_url"`
	ReviewCommentsURL   string    `json:"review_comments_url"`
	ReviewCommentURL    string    `json:"review_comment_url"`
	CommentsURL         string    `json:"comments_url"`
	StatusesURL         string    `json:"statuses_url"`
	Number              int       `json:"number"`
	State               string    `json:"state"`
	Locked              bool      `json:"locked"`
	Title               string    `json:"title"`
	User                User      `json:"user"`
	Body                string    `json:"body"`
	Labels              []Label   `json:"labels"`
	Milestone           Milestone `json:"milestone"`
	ActiveLockReason    string    `json:"active_lock_reason"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	ClosedAt            time.Time `json:"closed_at"`
	MergedAt            time.Time `json:"merged_at"`
	MergeCommitSha      string    `json:"merge_commit_sha"`
	Assignee            User      `json:"assignee"`
	Assignees           []User    `json:"assignees"`
	RequestedReviewers  []User    `json:"requested_reviewers"`
	RequestedTeams      []Team    `json:"requested_teams"`
	Head                Ref       `json:"head"`
	Base                Ref       `json:"base"`
	Links               Links     `json:"_links"`
	AuthorAssociation   string    `json:"author_association"`
	Draft               bool      `json:"draft"`
	Merged              bool      `json:"merged"`
	Mergeable           bool      `json:"mergeable"`
	Rebaseable          bool      `json:"rebaseable"`
	MergeableState      string    `json:"mergeable_state"`
	MergedBy            User      `json:"merged_by"`
	Comments            int       `json:"comments"`
	ReviewComments      int       `json:"review_comments"`
	MaintainerCanModify bool      `json:"maintainer_can_modify"`
	Commits             int       `json:"commits"`
	Additions           int       `json:"additions"`
	Deletions           int       `json:"deletions"`
	ChangedFiles        int       `json:"changed_files"`
}

type User struct {
	Login             string `json:"login"`
	ID                int    `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
}

type Repo struct {
	ID               int         `json:"id"`
	NodeID           string      `json:"node_id"`
	Name             string      `json:"name"`
	FullName         string      `json:"full_name"`
	Owner            User        `json:"owner"`
	Private          bool        `json:"private"`
	HTMLURL          string      `json:"html_url"`
	Description      string      `json:"description"`
	Fork             bool        `json:"fork"`
	URL              string      `json:"url"`
	ArchiveURL       string      `json:"archive_url"`
	AssigneesURL     string      `json:"assignees_url"`
	BlobsURL         string      `json:"blobs_url"`
	BranchesURL      string      `json:"branches_url"`
	CollaboratorsURL string      `json:"collaborators_url"`
	CommentsURL      string      `json:"comments_url"`
	CommitsURL       string      `json:"commits_url"`
	CompareURL       string      `json:"compare_url"`
	ContentsURL      string      `json:"contents_url"`
	ContributorsURL  string      `json:"contributors_url"`
	DeploymentsURL   string      `json:"deployments_url"`
	DownloadsURL     string      `json:"downloads_url"`
	EventsURL        string      `json:"events_url"`
	ForksURL         string      `json:"forks_url"`
	GitCommitsURL    string      `json:"git_commits_url"`
	GitRefsURL       string      `json:"git_refs_url"`
	GitTagsURL       string      `json:"git_tags_url"`
	GitURL           string      `json:"git_url"`
	IssueCommentURL  string      `json:"issue_comment_url"`
	IssueEventsURL   string      `json:"issue_events_url"`
	IssuesURL        string      `json:"issues_url"`
	KeysURL          string      `json:"keys_url"`
	LabelsURL        string      `json:"labels_url"`
	LanguagesURL     string      `json:"languages_url"`
	MergesURL        string      `json:"merges_url"`
	MilestonesURL    string      `json:"milestones_url"`
	NotificationsURL string      `json:"notifications_url"`
	PullsURL         string      `json:"pulls_url"`
	ReleasesURL      string      `json:"releases_url"`
	SSHURL           string      `json:"ssh_url"`
	StargazersURL    string      `json:"stargazers_url"`
	StatusesURL      string      `json:"statuses_url"`
	SubscribersURL   string      `json:"subscribers_url"`
	SubscriptionURL  string      `json:"subscription_url"`
	TagsURL          string      `json:"tags_url"`
	TeamsURL         string      `json:"teams_url"`
	TreesURL         string      `json:"trees_url"`
	CloneURL         string      `json:"clone_url"`
	MirrorURL        string      `json:"mirror_url"`
	HooksURL         string      `json:"hooks_url"`
	SvnURL           string      `json:"svn_url"`
	Homepage         string      `json:"homepage"`
	Language         interface{} `json:"language"`
	ForksCount       int         `json:"forks_count"`
	StargazersCount  int         `json:"stargazers_count"`
	WatchersCount    int         `json:"watchers_count"`
	Size             int         `json:"size"`
	DefaultBranch    string      `json:"default_branch"`
	OpenIssuesCount  int         `json:"open_issues_count"`
	IsTemplate       bool        `json:"is_template"`
	Topics           []string    `json:"topics"`
	HasIssues        bool        `json:"has_issues"`
	HasProjects      bool        `json:"has_projects"`
	HasWiki          bool        `json:"has_wiki"`
	HasPages         bool        `json:"has_pages"`
	HasDownloads     bool        `json:"has_downloads"`
	Archived         bool        `json:"archived"`
	Disabled         bool        `json:"disabled"`
	PushedAt         time.Time   `json:"pushed_at"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
	Permissions      struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
		Pull  bool `json:"pull"`
	} `json:"permissions"`
	AllowRebaseMerge   bool        `json:"allow_rebase_merge"`
	TemplateRepository interface{} `json:"template_repository"`
	AllowSquashMerge   bool        `json:"allow_squash_merge"`
	AllowMergeCommit   bool        `json:"allow_merge_commit"`
	SubscribersCount   int         `json:"subscribers_count"`
	NetworkCount       int         `json:"network_count"`
}

type Ref struct {
	Label string `json:"label"`
	Ref   string `json:"ref"`
	Sha   string `json:"sha"`
	User  User   `json:"user"`
	Repo  Repo   `json:"repo"`
}

type Links struct {
	Self struct {
		Href string `json:"href"`
	} `json:"self"`
	HTML struct {
		Href string `json:"href"`
	} `json:"html"`
	Issue struct {
		Href string `json:"href"`
	} `json:"issue"`
	Comments struct {
		Href string `json:"href"`
	} `json:"comments"`
	ReviewComments struct {
		Href string `json:"href"`
	} `json:"review_comments"`
	ReviewComment struct {
		Href string `json:"href"`
	} `json:"review_comment"`
	Commits struct {
		Href string `json:"href"`
	} `json:"commits"`
	Statuses struct {
		Href string `json:"href"`
	} `json:"statuses"`
}

type Team struct {
	ID              int         `json:"id"`
	NodeID          string      `json:"node_id"`
	URL             string      `json:"url"`
	HTMLURL         string      `json:"html_url"`
	Name            string      `json:"name"`
	Slug            string      `json:"slug"`
	Description     string      `json:"description"`
	Privacy         string      `json:"privacy"`
	Permission      string      `json:"permission"`
	MembersURL      string      `json:"members_url"`
	RepositoriesURL string      `json:"repositories_url"`
	Parent          interface{} `json:"parent"`
}

type Label struct {
	ID          int    `json:"id"`
	NodeID      string `json:"node_id"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Default     bool   `json:"default"`
}

type Milestone struct {
	URL          string    `json:"url"`
	HTMLURL      string    `json:"html_url"`
	LabelsURL    string    `json:"labels_url"`
	ID           int       `json:"id"`
	NodeID       string    `json:"node_id"`
	Number       int       `json:"number"`
	State        string    `json:"state"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Creator      User      `json:"creator"`
	OpenIssues   int       `json:"open_issues"`
	ClosedIssues int       `json:"closed_issues"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ClosedAt     time.Time `json:"closed_at"`
	DueOn        time.Time `json:"due_on"`
}

type Comment struct {
	ID        int       `json:"id"`
	NodeID    string    `json:"node_id"`
	URL       string    `json:"url"`
	HTMLURL   string    `json:"html_url"`
	Body      string    `json:"body"`
	User      User      `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Client struct {
	BaseURL    string
	Token      string
	Version    string // used in user-agent
	HTTPClient *http.Client
}

func (c *Client) URL(path string, params []string) (*url.URL, error) {
	rawBaseURL := c.BaseURL
	if rawBaseURL == "" {
		rawBaseURL = DEFAULT_BASE_URL
	}
	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, err
	}
	if len(params)%2 != 0 {
		return nil, fmt.Errorf("params should be pairs")
	}

	v := url.Values{}
	for i := 0; i < len(params)/2; i++ {
		v[params[i*2]] = []string{params[i*2+1]}
	}

	return baseURL.ResolveReference(&url.URL{
		Path:     path,
		RawQuery: v.Encode(),
	}), nil
}

func (c *Client) Do(method, path string, params []string, body interface{}, out interface{}) error {
	if c.Token == "" {
		return fmt.Errorf("token not set")
	}

	u, err := c.URL(path, params)
	if err != nil {
		return err
	}

	var bodyR io.Reader
	if body != nil {
		bodyBuf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyR = bytes.NewReader(bodyBuf)
	}

	req, err := http.NewRequest(method, u.String(), bodyR)
	if err != nil {
		return err
	}

	// https://developer.github.com/v3/#user-agent-required
	req.Header.Set("User-Agent", "https://github.com/wader/bump "+c.Version)
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	req.Header.Add("Authorization", "token "+c.Token)

	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	hc := http.DefaultClient
	if c.HTTPClient != nil {
		hc = c.HTTPClient
	}

	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s", resp.Status)
	}

	if out != nil {
		err = json.NewDecoder(resp.Body).Decode(out)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) NewRepoRef(name string) *RepoRef {
	return &RepoRef{c: c, Name: name}
}

type RepoRef struct {
	c    *Client
	Name string
}

type NewPullRequest struct {
	Title               string  `json:"title"`
	Head                string  `json:"head"`
	Base                string  `json:"base"`
	Body                *string `json:"body,omitempty"`
	MaintainerCanModify *bool   `json:"maintainer_can_modify,omitempty"`
	Draft               *bool   `json:"draft,omitempty"`
}

func (repo *RepoRef) CreatePullRequest(pr NewPullRequest) (PullRequest, error) {
	var newPr PullRequest
	err := repo.c.Do("POST", fmt.Sprintf("repos/%s/pulls", repo.Name), nil, pr, &newPr)
	return newPr, err
}

type UpdatePullRequest struct {
	Title               *string `json:"title,omitempty"`
	Base                *string `json:"base,omitempty"`
	Body                *string `json:"body,omitempty"`
	State               *string `json:"state,omitempty"`
	MaintainerCanModify *bool   `json:"maintainer_can_modify,omitempty"`
}

func (repo *RepoRef) UpdatePullRequest(prNumber int, pr UpdatePullRequest) (PullRequest, error) {
	var outPr PullRequest
	err := repo.c.Do("PATCH", fmt.Sprintf("repos/%s/pulls/%d", repo.Name, prNumber), nil, pr, &outPr)
	return outPr, err
}

func (repo *RepoRef) ListPullRequest(params ...string) ([]PullRequest, error) {
	var outPrs []PullRequest
	err := repo.c.Do("GET", fmt.Sprintf("repos/%s/pulls", repo.Name), params, nil, &outPrs)
	return outPrs, err
}

type NewComment struct {
	Body string `json:"body"`
}

func (repo *RepoRef) CreateComment(prNumber int, com NewComment) (Comment, error) {
	var outCom Comment
	err := repo.c.Do("POST", fmt.Sprintf("repos/%s/issues/%d/comments", repo.Name, prNumber), nil, com, &outCom)
	return outCom, err
}
