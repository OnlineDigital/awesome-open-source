// Package gitlab is the GitLab forge driver (gitlab.com or a self-managed
// instance). It uses the REST v4 API; a token is optional for public projects.
package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"awesomeopensource.dev/app/forge"
	"awesomeopensource.dev/app/scoring"
)

// commitPages caps how many 100-commit pages of the last year are walked.
const commitPages = 10

type Client struct {
	Token   string
	WebURL  string // default https://gitlab.com
	BaseURL string // API base, default WebURL + /api/v4
	HTTP    *http.Client
}

// New returns the gitlab.com driver.
func New(token string) *Client { return NewInstance("https://gitlab.com", token) }

// NewInstance targets any GitLab instance. Self-managed instances get their own
// driver ID ("gitlab:host") so they coexist with gitlab.com.
func NewInstance(webURL, token string) *Client {
	webURL = strings.TrimRight(webURL, "/")
	return &Client{Token: token, WebURL: webURL, BaseURL: webURL + "/api/v4", HTTP: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) hosted() bool { return strings.EqualFold(c.Host(), "gitlab.com") }

func (c *Client) ID() string {
	if c.hosted() {
		return "gitlab"
	}
	return "gitlab:" + strings.ToLower(c.Host())
}

func (c *Client) Label() string {
	if c.hosted() {
		return "GitLab"
	}
	return "GitLab (" + c.Host() + ")"
}

func (c *Client) Host() string {
	u, err := url.Parse(c.WebURL)
	if err != nil {
		return ""
	}
	return u.Host
}

func (c *Client) URL(owner, name string) string {
	return strings.TrimSuffix(c.WebURL, "/") + "/" + owner + "/" + name
}

// ParsePath accepts "group[/subgroup...]/project[.git][/-/...]".
func (c *Client) ParsePath(path string) (owner, name string, ok bool) {
	path = strings.Trim(path, "/")
	if i := strings.Index(path, "/-/"); i >= 0 {
		path = path[:i]
	}
	parts := strings.Split(strings.TrimSuffix(path, "/-"), "/")
	// Legacy URLs without "/-/" (e.g. /group/project/tree/main): cut at the first route keyword.
	for i := 2; i < len(parts); i++ {
		if legacyRoutes[parts[i]] {
			parts = parts[:i]
			break
		}
	}
	if len(parts) < 2 {
		return "", "", false
	}
	parts[len(parts)-1] = strings.TrimSuffix(parts[len(parts)-1], ".git")
	for _, p := range parts {
		if !forge.ValidSegment(p) {
			return "", "", false
		}
	}
	return strings.Join(parts[:len(parts)-1], "/"), parts[len(parts)-1], true
}

// legacyRoutes are GitLab project sub-pages that older URLs put directly after the project path.
var legacyRoutes = map[string]bool{"tree": true, "blob": true, "raw": true, "blame": true, "commits": true, "commit": true,
	"issues": true, "merge_requests": true, "tags": true, "releases": true, "branches": true, "wikis": true, "pipelines": true,
	"jobs": true, "compare": true, "graphs": true, "network": true, "activity": true, "snippets": true, "environments": true}

type project struct {
	Path           string   `json:"path"`
	Description    *string  `json:"description"`
	WebURL         string   `json:"web_url"`
	AvatarURL      *string  `json:"avatar_url"`
	Topics         []string `json:"topics"`
	TagList        []string `json:"tag_list"`
	Stars          int      `json:"star_count"`
	Forks          int      `json:"forks_count"`
	OpenIssues     int      `json:"open_issues_count"`
	Archived       bool     `json:"archived"`
	DefaultBranch  string   `json:"default_branch"`
	CreatedAt      string   `json:"created_at"`
	LastActivityAt string   `json:"last_activity_at"`
	Namespace      struct {
		FullPath  string  `json:"full_path"`
		AvatarURL *string `json:"avatar_url"`
	} `json:"namespace"`
	License *struct {
		Key      string `json:"key"`
		Name     string `json:"name"`
		Nickname string `json:"nickname"`
	} `json:"license"`
}

type release struct {
	TagName    string `json:"tag_name"`
	ReleasedAt string `json:"released_at"`
	Upcoming   bool   `json:"upcoming_release"`
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// Fetch gathers everything needed to refresh a project.
func (c *Client) Fetch(ctx context.Context, owner, name string) (*forge.Repo, error) {
	base := "/projects/" + strings.ReplaceAll(url.PathEscape(owner+"/"+name), "/", "%2F")
	var p project
	if _, err := c.get(ctx, base+"?license=true", &p); err != nil {
		return nil, err
	}
	r := &forge.Repo{
		Owner: p.Namespace.FullPath, Name: p.Path, WebURL: p.WebURL,
		Topics: p.Topics, Stars: p.Stars, Forks: p.Forks, OpenIssues: p.OpenIssues,
		Archived: p.Archived, DefaultBranch: p.DefaultBranch,
		CreatedAt: parseTime(p.CreatedAt), PushedAt: parseTime(p.LastActivityAt),
	}
	if r.Owner == "" || r.Name == "" {
		r.Owner, r.Name = owner, name
	}
	if len(r.Topics) == 0 {
		r.Topics = p.TagList
	}
	if p.Description != nil {
		r.Description = strings.TrimSpace(*p.Description)
	}
	switch {
	case p.AvatarURL != nil && *p.AvatarURL != "":
		r.AvatarURL = *p.AvatarURL
	case p.Namespace.AvatarURL != nil:
		r.AvatarURL = *p.Namespace.AvatarURL
	}
	if strings.HasPrefix(r.AvatarURL, "/") {
		r.AvatarURL = strings.TrimSuffix(c.WebURL, "/") + r.AvatarURL
	}
	if l := p.License; l != nil {
		r.License = firstNonEmpty(l.Nickname, strings.ToUpper(l.Key), l.Name)
	}

	var langs map[string]float64
	if _, err := c.get(ctx, base+"/languages", &langs); err == nil {
		best := 0.0
		for l, pct := range langs {
			if pct > best || (pct == best && l < r.Language) {
				r.Language, best = l, pct
			}
		}
	} else if errors.Is(err, forge.ErrRateLimited) {
		return nil, err
	}

	now := time.Now()
	sig := scoring.Signals{Now: now, Archived: p.Archived, CreatedAt: r.CreatedAt, PushedAt: r.PushedAt}

	var rels []release
	if _, err := c.get(ctx, base+"/releases?per_page=10", &rels); err != nil && !errors.Is(err, forge.ErrNotFound) {
		return nil, err
	}
	for _, rel := range rels {
		if rel.Upcoming {
			continue
		}
		sr := scoring.Release{Tag: rel.TagName, PublishedAt: parseTime(rel.ReleasedAt), Prerelease: isPrerelease(rel.TagName)}
		sig.Releases = append(sig.Releases, sr)
		if !sr.Prerelease && (r.LastRelease == nil || sr.PublishedAt.After(r.LastRelease.PublishedAt)) {
			cp := sr
			r.LastRelease = &cp
		}
	}

	if err := c.commits(ctx, base, p.DefaultBranch, &sig); err != nil {
		return nil, err
	}
	all, err := c.branches(ctx, base, &sig)
	if err != nil {
		return nil, err
	}
	r.AllBranches = all
	for _, m := range sig.RecentCommitMessages {
		sig.SampledCommits++
		if scoring.IsFixMessage(m) {
			sig.FixCommits++
		}
	}
	r.Signals = sig
	return r, nil
}

// commits counts default-branch commits of the last year and samples their titles.
func (c *Client) commits(ctx context.Context, base, branch string, sig *scoring.Signals) error {
	q := url.Values{"per_page": {"100"}, "since": {sig.Now.AddDate(0, 0, -365).Format(time.RFC3339)}}
	if branch != "" {
		q.Set("ref_name", branch)
	}
	for page := 1; page <= commitPages; page++ {
		q.Set("page", strconv.Itoa(page))
		var list []struct {
			Title         string `json:"title"`
			CommittedDate string `json:"committed_date"`
		}
		h, err := c.get(ctx, base+"/repository/commits?"+q.Encode(), &list)
		if errors.Is(err, forge.ErrNotFound) { // empty repository
			return nil
		} else if err != nil {
			return err
		}
		for _, cm := range list {
			t := parseTime(cm.CommittedDate)
			if t.After(sig.LastCommitAt) {
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
				sig.RecentCommitMessages = append(sig.RecentCommitMessages, cm.Title)
			}
		}
		if total, err := strconv.Atoi(h.Get("X-Total")); err == nil && total > sig.Commits365 && page == commitPages {
			sig.Commits365 = total
		}
		if len(list) < 100 || h.Get("X-Next-Page") == "" {
			break
		}
	}
	return nil
}

// branches finds the newest commit across branches and counts recently active ones.
// It reports whether every branch fit in the inspected page.
func (c *Client) branches(ctx context.Context, base string, sig *scoring.Signals) (bool, error) {
	var list []struct {
		Commit struct {
			CommittedDate string `json:"committed_date"`
		} `json:"commit"`
	}
	h, err := c.get(ctx, base+"/repository/branches?per_page=100", &list)
	if errors.Is(err, forge.ErrNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	sig.TotalBranches = len(list)
	if total, err := strconv.Atoi(h.Get("X-Total")); err == nil && total > len(list) {
		sig.TotalBranches = total
	}
	cutoff := sig.Now.AddDate(0, 0, -90)
	for _, b := range list {
		t := parseTime(b.Commit.CommittedDate)
		if t.After(sig.LastCommitAt) {
			sig.LastCommitAt = t
		}
		if t.After(cutoff) {
			sig.ActiveBranches90++
		}
	}
	return sig.TotalBranches == len(list), nil
}

// isPrerelease flags semver-style pre-release tags, since GitLab releases have no such flag.
func isPrerelease(tag string) bool {
	t := strings.ToLower(tag)
	for _, m := range []string{"-rc", "-alpha", "-beta", "-pre", "-dev", ".rc", "rc."} {
		if strings.Contains(t, m) {
			return true
		}
	}
	return false
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func (c *Client) get(ctx context.Context, path string, out any) (http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(c.BaseURL, "/")+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "awesomeopensource.dev-sync")
	if c.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return resp.Header, forge.ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests:
		reset, _ := strconv.ParseInt(resp.Header.Get("RateLimit-Reset"), 10, 64)
		return resp.Header, fmt.Errorf("%w (resets %s)", forge.ErrRateLimited, time.Unix(reset, 0).Format(time.RFC3339))
	case resp.StatusCode >= 300:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return resp.Header, fmt.Errorf("gitlab: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return resp.Header, json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(out)
}
