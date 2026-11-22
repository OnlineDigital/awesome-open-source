// Package forge abstracts code hosting platforms (GitHub, GitLab, ...) behind
// drivers. A driver parses repository references for its host and fetches the
// metadata and activity signals used by the scoring engine.
package forge

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"awesomeopensource.dev/app/scoring"
)

var (
	ErrNotFound    = errors.New("forge: repository not found")
	ErrRateLimited = errors.New("forge: rate limited")
)

// Repo is the metadata persisted on a project plus the scoring signals.
type Repo struct {
	Owner, Name   string // Owner may contain "/" on hosts with nested groups (GitLab)
	WebURL        string
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
	AllBranches   bool // whether every branch was inspected (vs. default branch only)
}

// Driver is one code hosting platform.
type Driver interface {
	ID() string    // stable key stored in the database, e.g. "github"
	Label() string // display name, e.g. "GitHub"
	Host() string  // web host matched when parsing URLs, e.g. "github.com"
	// ParsePath splits the URL path after the host into owner and name.
	ParsePath(path string) (owner, name string, ok bool)
	URL(owner, name string) string
	Fetch(ctx context.Context, owner, name string) (*Repo, error)
}

// Ref identifies a repository on a specific forge.
type Ref struct{ Forge, Owner, Name string }

func (r Ref) FullName() string { return r.Owner + "/" + r.Name }

// Registry holds the enabled drivers; the first one is the default for bare "owner/repo" input.
type Registry struct{ drivers []Driver }

func NewRegistry(drivers ...Driver) *Registry { return &Registry{drivers: drivers} }

func (r *Registry) Get(id string) (Driver, bool) {
	for _, d := range r.drivers {
		if d.ID() == id {
			return d, true
		}
	}
	return nil, false
}

func (r *Registry) Drivers() []Driver { return r.drivers }

// Labels lists display names, e.g. "GitHub, GitLab".
func (r *Registry) Labels() string {
	names := make([]string, len(r.drivers))
	for i, d := range r.drivers {
		names[i] = d.Label()
	}
	return strings.Join(names, ", ")
}

// URL returns the web URL of a stored repository, falling back to the default driver's format.
func (r *Registry) URL(forgeID, owner, name string) string {
	if d, ok := r.Get(forgeID); ok {
		return d.URL(owner, name)
	}
	return ""
}

// Parse accepts https://host/o/r[.git][/...], host/o/r, git@host:o/r.git and bare o/r (default driver).
func (r *Registry) Parse(input string) (Ref, bool) {
	s := strings.TrimSpace(input)
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	if strings.HasPrefix(s, "git@") {
		s = strings.Replace(strings.TrimPrefix(s, "git@"), ":", "/", 1)
	}
	for _, p := range []string{"https://", "http://", "ssh://", "www."} {
		s = strings.TrimPrefix(s, p)
	}
	s = strings.Trim(s, "/")
	host, rest, _ := strings.Cut(s, "/")
	for _, d := range r.drivers {
		if strings.EqualFold(host, d.Host()) {
			if o, n, ok := d.ParsePath(rest); ok {
				return Ref{Forge: d.ID(), Owner: o, Name: n}, true
			}
			return Ref{}, false
		}
	}
	if strings.Contains(host, ".") || len(r.drivers) == 0 { // an unsupported host
		return Ref{}, false
	}
	d := r.drivers[0]
	if o, n, ok := d.ParsePath(s); ok {
		return Ref{Forge: d.ID(), Owner: o, Name: n}, true
	}
	return Ref{}, false
}

var segmentRe = regexp.MustCompile(`^[A-Za-z0-9_](?:[A-Za-z0-9._-]*)$`)

// ValidSegment reports whether s is a safe owner/group/repository path segment.
func ValidSegment(s string) bool {
	return segmentRe.MatchString(s) && s != "." && s != ".."
}
