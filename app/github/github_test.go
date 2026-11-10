package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseRepo(t *testing.T) {
	ok := map[string][2]string{
		"https://github.com/vercel/next.js":         {"vercel", "next.js"},
		"https://github.com/golang/go.git":          {"golang", "go"},
		"github.com/a/b/tree/main/docs?x=1":         {"a", "b"},
		"git@github.com:rust-lang/rust.git":         {"rust-lang", "rust"},
		"  AndreiTelteu/inertia-go-solid-template ": {"AndreiTelteu", "inertia-go-solid-template"},
	}
	for in, want := range ok {
		o, n, good := ParseRepo(in)
		if !good || o != want[0] || n != want[1] {
			t.Errorf("ParseRepo(%q) = %q,%q,%v", in, o, n, good)
		}
	}
	for _, in := range []string{"", "justone", "https://gitlab.com", "a/..", "a/b c", "/"} {
		if _, _, good := ParseRepo(in); good && in != "https://gitlab.com" {
			t.Errorf("ParseRepo(%q) should fail", in)
		}
	}
}

func TestFetchGraphQL(t *testing.T) {
	now := time.Now().UTC()
	ts := func(d int) string { return now.AddDate(0, 0, -d).Format(time.RFC3339) }
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"name":"r","owner":{"login":"o","avatar_url":"a"},"description":"d","stargazers_count":7,"default_branch":"main","pushed_at":%q,"created_at":%q,"topics":["x"],"license":{"spdx_id":"MIT"}}`, ts(40), ts(900))
	})
	mux.HandleFunc("/repos/o/r/releases", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"tag_name":"v1.2.3","published_at":%q},{"tag_name":"v1.3.0-rc1","prerelease":true,"published_at":%q},{"tag_name":"draft","draft":true}]`, ts(30), ts(5))
	})
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Error("missing token")
		}
		fmt.Fprintf(w, `{"data":{"repository":{"defaultBranchRef":{"target":{"c30":{"totalCount":3},"c90":{"totalCount":9},"c365":{"totalCount":30},"recent":{"nodes":[{"messageHeadline":"fix: crash"},{"messageHeadline":"docs"}]}}},
		"refs":{"totalCount":2,"nodes":[{"target":{"committedDate":%q}},{"target":{"committedDate":%q}}]}}}}`, ts(2), ts(200))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New("tok")
	c.BaseURL = srv.URL
	repo, err := c.Fetch(context.Background(), "o", "r")
	if err != nil {
		t.Fatal(err)
	}
	if !repo.GraphQL || repo.Stars != 7 || repo.License != "MIT" {
		t.Fatalf("unexpected repo %+v", repo)
	}
	s := repo.Signals
	if s.Commits90 != 9 || s.ActiveBranches90 != 1 || s.TotalBranches != 2 || s.FixCommits != 1 || s.SampledCommits != 2 || len(s.Releases) != 2 {
		t.Fatalf("unexpected signals %+v", s)
	}
	if d := now.Sub(s.LastCommitAt).Hours() / 24; d < 1.9 || d > 2.1 {
		t.Fatalf("last commit should come from the newest branch, got %.1f days", d)
	}
	if repo.LastRelease == nil || repo.LastRelease.Tag != "v1.2.3" {
		t.Fatalf("last stable release wrong: %+v", repo.LastRelease)
	}
}

func TestFetchNotFoundAndRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "missing") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1900000000")
		w.WriteHeader(403)
	}))
	defer srv.Close()
	c := New("")
	c.BaseURL = srv.URL
	if _, err := c.Fetch(context.Background(), "o", "missing"); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if _, err := c.Fetch(context.Background(), "o", "busy"); err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("want rate limit, got %v", err)
	}
}
