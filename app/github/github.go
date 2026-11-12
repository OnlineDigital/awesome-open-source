// Package github fetches repository metadata and the activity signals used by
// the scoring engine. With a token it uses GraphQL to see every branch; without
// one it falls back to the REST API (default branch + pushed_at).
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"awesomeopensource.dev/app/scoring"
)

var (
	ErrNotFound    = errors.New("github: repository not found")
	ErrRateLimited = errors.New("github: rate limited")
)

type Client struct {
	Token   string
	BaseURL string // REST base, default https://api.github.com
	HTTP    *http.Client
}

func New(token string) *Client {
	return &Client{Token: token, BaseURL: "https://api.github.com", HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// Repo is the metadata persisted on a project plus the scoring signals.
type Repo struct {
	Owner, Name   string
	Description   string
	Homepage      string
	License       string
	Language      string
	Topics        []string
	AvatarURL     string
	Stars, Forks  int
	OpenIssues    int
	Archived      bool
	DefaultBranch string
	CreatedAt     time.Time
	PushedAt      time.Time
	LastRelease   *scoring.Release
	Signals       scoring.Signals
	GraphQL       bool // whether all branches were inspected
}

var repoRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9._-]*)$`)

// ParseRepo accepts https://github.com/o/r[.git][/...], github.com/o/r, git@github.com:o/r.git and o/r.
func ParseRepo(input string) (owner, name string, ok bool) {
	s := strings.TrimSpace(input)
	s = strings.TrimPrefix(s, "git@github.com:")
	for _, p := range []string{"https://", "http://", "www."} {
		s = strings.TrimPrefix(s, p)
	}
	s = strings.TrimPrefix(s, "github.com/")
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(strings.Trim(s, "/"), "/")
	if len(parts) < 2 {
		return "", "", false
	}
	owner, name = parts[0], strings.TrimSuffix(parts[1], ".git")
	if !repoRe.MatchString(owner) || !repoRe.MatchString(name) || name == "." || name == ".." {
		return "", "", false
	}
	return owner, name, true
}

type restRepo struct {
	Description   *string  `json:"description"`
	Homepage      *string  `json:"homepage"`
	Language      *string  `json:"language"`
	Topics        []string `json:"topics"`
	Stars         int      `json:"stargazers_count"`
	Forks         int      `json:"forks_count"`
	OpenIssues    int      `json:"open_issues_count"`
	Archived      bool     `json:"archived"`
	Disabled      bool     `json:"disabled"`
	DefaultBranch string   `json:"default_branch"`
	CreatedAt     string   `json:"created_at"`
	PushedAt      string   `json:"pushed_at"`
	License       *struct {
		SPDX string `json:"spdx_id"`
		Name string `json:"name"`
	} `json:"license"`
	Owner struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"owner"`
	Name string `json:"name"`
}

type restRelease struct {
	TagName     string `json:"tag_name"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	PublishedAt string `json:"published_at"`
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// Fetch gathers everything needed to refresh a project.
func (c *Client) Fetch(ctx context.Context, owner, name string) (*Repo, error) {
	var meta restRepo
	if err := c.rest(ctx, fmt.Sprintf("/repos/%s/%s", url.PathEscape(owner), url.PathEscape(name)), &meta); err != nil {
		return nil, err
	}
	r := &Repo{
		Owner: meta.Owner.Login, Name: meta.Name, AvatarURL: meta.Owner.AvatarURL,
		Topics: meta.Topics, Stars: meta.Stars, Forks: meta.Forks, OpenIssues: meta.OpenIssues,
		Archived: meta.Archived, DefaultBranch: meta.DefaultBranch,
		CreatedAt: parseTime(meta.CreatedAt), PushedAt: parseTime(meta.PushedAt),
	}
	if meta.Description != nil {
		r.Description = *meta.Description
	}
	if meta.Homepage != nil {
		r.Homepage = *meta.Homepage
	}
	if meta.Language != nil {
		r.Language = *meta.Language
	}
	if meta.License != nil {
		r.License = meta.License.SPDX
		if r.License == "" || r.License == "NOASSERTION" {
			r.License = meta.License.Name
		}
	}

	now := time.Now()
	sig := scoring.Signals{
		Now: now, Archived: meta.Archived, Disabled: meta.Disabled,
		CreatedAt: r.CreatedAt, PushedAt: r.PushedAt, LastCommitAt: r.PushedAt,
	}

	var rels []restRelease
	// The score only needs recency plus up to six releases in the last year; requesting
	// fewer releases keeps huge release bodies (100+ releases) from breaking the fetch.
	if err := c.rest(ctx, fmt.Sprintf("/repos/%s/%s/releases?per_page=10", url.PathEscape(owner), url.PathEscape(name)), &rels); err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	for _, rel := range rels {
		if rel.Draft {
			continue
		}
		sr := scoring.Release{Tag: rel.TagName, PublishedAt: parseTime(rel.PublishedAt), Prerelease: rel.Prerelease}
		sig.Releases = append(sig.Releases, sr)
		if !sr.Prerelease && (r.LastRelease == nil || sr.PublishedAt.After(r.LastRelease.PublishedAt)) {
			cp := sr
			r.LastRelease = &cp
		}
	}

	if c.Token != "" {
		if err := c.graphqlCommits(ctx, r, &sig); err == nil {
			r.GraphQL = true
		} else if errors.Is(err, ErrRateLimited) {
			return nil, err
		}
	}
	if !r.GraphQL {
		if err := c.restCommits(ctx, owner, name, meta.DefaultBranch, &sig); err != nil {
			return nil, err
		}
	}
	for _, m := range sig.RecentCommitMessages {
		sig.SampledCommits++
		if scoring.IsFixMessage(m) {
			sig.FixCommits++
		}
	}
	r.Signals = sig
	return r, nil
}

const graphqlQuery = `query($owner:String!,$name:String!,$s30:GitTimestamp!,$s90:GitTimestamp!,$s365:GitTimestamp!){
 repository(owner:$owner,name:$name){
  defaultBranchRef{ name target{ ... on Commit{
    c30:history(since:$s30){totalCount}
    c90:history(since:$s90){totalCount}
    c365:history(since:$s365){totalCount}
    recent:history(first:100){nodes{messageHeadline}}
  }}}
  refs(refPrefix:"refs/heads/",first:100,orderBy:{field:TAG_COMMIT_DATE,direction:DESC}){
    totalCount
    nodes{ name target{ ... on Commit{ committedDate } } }
  }
 }
}`

func (c *Client) graphqlCommits(ctx context.Context, r *Repo, sig *scoring.Signals) error {
	now := sig.Now
	vars := map[string]any{
		"owner": r.Owner, "name": r.Name,
		"s30": now.AddDate(0, 0, -30).Format(time.RFC3339), "s90": now.AddDate(0, 0, -90).Format(time.RFC3339), "s365": now.AddDate(0, 0, -365).Format(time.RFC3339),
	}
	body, _ := json.Marshal(map[string]any{"query": graphqlQuery, "variables": vars})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(c.BaseURL, "/")+"/graphql", bytes.NewReader(body))
	c.headers(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return err
	}
	var out struct {
		Errors []struct{ Message string } `json:"errors"`
		Data   struct {
			Repository *struct {
				DefaultBranchRef *struct {
					Target struct {
						C30    struct{ TotalCount int } `json:"c30"`
						C90    struct{ TotalCount int } `json:"c90"`
						C365   struct{ TotalCount int } `json:"c365"`
						Recent struct {
							Nodes []struct {
								MessageHeadline string `json:"messageHeadline"`
							} `json:"nodes"`
						} `json:"recent"`
					} `json:"target"`
				} `json:"defaultBranchRef"`
				Refs struct {
					TotalCount int `json:"totalCount"`
					Nodes      []struct {
						Target struct {
							CommittedDate string `json:"committedDate"`
						} `json:"target"`
					} `json:"nodes"`
				} `json:"refs"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&out); err != nil {
		return err
	}
	if out.Data.Repository == nil {
		if len(out.Errors) > 0 {
			return fmt.Errorf("github graphql: %s", out.Errors[0].Message)
		}
		return ErrNotFound
	}
	repo := out.Data.Repository
	if d := repo.DefaultBranchRef; d != nil {
		sig.Commits30, sig.Commits90, sig.Commits365 = d.Target.C30.TotalCount, d.Target.C90.TotalCount, d.Target.C365.TotalCount
		for _, n := range d.Target.Recent.Nodes {
			sig.RecentCommitMessages = append(sig.RecentCommitMessages, n.MessageHeadline)
		}
	}
	sig.TotalBranches = repo.Refs.TotalCount
	cutoff := now.AddDate(0, 0, -90)
	for _, n := range repo.Refs.Nodes {
		t := parseTime(n.Target.CommittedDate)
		if t.After(sig.LastCommitAt) {
			sig.LastCommitAt = t
		}
		if t.After(cutoff) {
			sig.ActiveBranches90++
		}
	}
	return nil
}

// restCommits is the unauthenticated fallback: it samples the default branch only.
func (c *Client) restCommits(ctx context.Context, owner, name, branch string, sig *scoring.Signals) error {
	since := sig.Now.AddDate(0, 0, -365).Format(time.RFC3339)
	var commits []struct {
		Commit struct {
			Message   string `json:"message"`
			Committer struct {
				Date string `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	path := fmt.Sprintf("/repos/%s/%s/commits?per_page=100&since=%s", url.PathEscape(owner), url.PathEscape(name), url.QueryEscape(since))
	if branch != "" {
		path += "&sha=" + url.QueryEscape(branch)
	}
	if err := c.rest(ctx, path, &commits); err != nil {
		if errors.Is(err, ErrNotFound) { // empty repository
			return nil
		}
		return err
	}
	for i, cm := range commits {
		t := parseTime(cm.Commit.Committer.Date)
		if i == 0 && t.After(sig.LastCommitAt) {
			sig.LastCommitAt = t
		}
		d := sig.Now.Sub(t).Hours() / 24
		sig.Commits365++
		if d <= 90 {
			sig.Commits90++
		}
		if d <= 30 {
			sig.Commits30++
		}
		if len(sig.RecentCommitMessages) < 100 {
			line, _, _ := strings.Cut(cm.Commit.Message, "\n")
			sig.RecentCommitMessages = append(sig.RecentCommitMessages, line)
		}
	}
	return nil
}

func (c *Client) headers(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "awesomeopensource.dev-sync")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

func checkStatus(resp *http.Response) error {
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests,
		resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0":
		reset, _ := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64)
		return fmt.Errorf("%w (resets %s)", ErrRateLimited, time.Unix(reset, 0).Format(time.RFC3339))
	case resp.StatusCode >= 300:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("github: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *Client) rest(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(c.BaseURL, "/")+path, nil)
	if err != nil {
		return err
	}
	c.headers(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return err
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(out)
}
