package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"awesomeopensource.dev/app/forge"
)

func TestFetch(t *testing.T) {
	now := time.Now().UTC()
	ts := func(d int) string { return now.AddDate(0, 0, -d).Format(time.RFC3339) }
	mux := http.NewServeMux()
	const base = "/api/v4/projects/grp%2Fsub%2Fproj"
	mux.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "tok" {
			t.Error("missing token")
		}
		fmt.Fprintf(w, `{"path":"proj","web_url":"https://gl.example/grp/sub/proj","description":"Error tracking","star_count":12,"forks_count":3,"open_issues_count":4,
		"default_branch":"main","created_at":%q,"last_activity_at":%q,"topics":["sentry"],"avatar_url":"/uploads/a.png",
		"namespace":{"full_path":"grp/sub"},"license":{"key":"mit","name":"MIT License","nickname":""}}`, ts(900), ts(1))
	})
	mux.HandleFunc(base+"/languages", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"Python":80.5,"HTML":19.5}`) })
	mux.HandleFunc(base+"/releases", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"tag_name":"v4.2.0-rc1","released_at":%q},{"tag_name":"v4.1.0","released_at":%q},{"tag_name":"v9","released_at":%q,"upcoming_release":true}]`, ts(3), ts(20), ts(-5))
	})
	mux.HandleFunc(base+"/repository/commits", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ref_name") != "main" {
			t.Error("commits not scoped to default branch")
		}
		fmt.Fprintf(w, `[{"title":"fix: crash","committed_date":%q},{"title":"docs","committed_date":%q},{"title":"feat","committed_date":%q}]`, ts(2), ts(50), ts(200))
	})
	mux.HandleFunc(base+"/repository/branches", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"commit":{"committed_date":%q}},{"commit":{"committed_date":%q}}]`, ts(1), ts(300))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New("tok")
	c.WebURL, c.BaseURL = srv.URL, srv.URL+"/api/v4"
	r, err := c.Fetch(context.Background(), "grp/sub", "proj")
	if err != nil {
		t.Fatal(err)
	}
	s := r.Signals
	if r.Owner != "grp/sub" || r.Name != "proj" || r.Stars != 12 || r.License != "MIT" || r.Language != "Python" || r.AvatarURL != srv.URL+"/uploads/a.png" || r.WebURL == "" {
		t.Fatalf("unexpected repo %+v", r)
	}
	if r.LastRelease == nil || r.LastRelease.Tag != "v4.1.0" || len(s.Releases) != 2 {
		t.Fatalf("releases wrong: %+v %+v", r.LastRelease, s.Releases)
	}
	if s.Commits30 != 1 || s.Commits90 != 2 || s.Commits365 != 3 || s.FixCommits != 1 || s.SampledCommits != 3 {
		t.Fatalf("commit signals wrong: %+v", s)
	}
	if s.TotalBranches != 2 || s.ActiveBranches90 != 1 || !r.AllBranches || now.Sub(s.LastCommitAt) > 25*time.Hour {
		t.Fatalf("branch signals wrong: %+v", s)
	}
}

func TestFetchErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() == "/projects/a%2Fmissing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("RateLimit-Reset", "1700000000")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	c := New("")
	c.BaseURL = srv.URL
	if _, err := c.Fetch(context.Background(), "a", "missing"); err != forge.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if _, err := c.Fetch(context.Background(), "a", "busy"); err == nil || !errors.Is(err, forge.ErrRateLimited) {
		t.Fatalf("want ErrRateLimited, got %v", err)
	}
}

func TestParsePathAndInstances(t *testing.T) {
	c := New("")
	for in, want := range map[string][2]string{
		"glitchtip/glitchtip-backend/tree/master":       {"glitchtip", "glitchtip-backend"},
		"grp/sub/proj/blob/main/README.md":              {"grp/sub", "proj"},
		"grp/proj/merge_requests/12":                    {"grp", "proj"},
		"glitchtip/glitchtip-backend/-/releases/v7.0.0": {"glitchtip", "glitchtip-backend"},
	} {
		if o, n, ok := c.ParsePath(in); !ok || o != want[0] || n != want[1] {
			t.Errorf("ParsePath(%q) = %q,%q,%v", in, o, n, ok)
		}
	}
	self := NewInstance("https://git.example.org/", "")
	if c.ID() != "gitlab" || self.ID() != "gitlab:git.example.org" || self.Label() != "GitLab (git.example.org)" || self.URL("a", "b") != "https://git.example.org/a/b" {
		t.Errorf("ids/labels wrong: %s %s %s", c.ID(), self.ID(), self.Label())
	}
}
