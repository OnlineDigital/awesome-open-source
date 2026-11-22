package forge_test

import (
	"testing"

	"awesomeopensource.dev/app/forge"
	"awesomeopensource.dev/app/forge/github"
	"awesomeopensource.dev/app/forge/gitlab"
)

func TestRegistryParse(t *testing.T) {
	reg := forge.NewRegistry(github.New(""), gitlab.New(""))
	ok := map[string]forge.Ref{
		"https://github.com/vercel/next.js":                       {"github", "vercel", "next.js"},
		"https://github.com/golang/go.git":                        {"github", "golang", "go"},
		"github.com/a/b/tree/main/docs?x=1":                       {"github", "a", "b"},
		"git@github.com:rust-lang/rust.git":                       {"github", "rust-lang", "rust"},
		"  AndreiTelteu/inertia-go-solid-template ":               {"github", "AndreiTelteu", "inertia-go-solid-template"},
		"https://gitlab.com/glitchtip/glitchtip-backend":          {"gitlab", "glitchtip", "glitchtip-backend"},
		"https://gitlab.com/glitchtip/glitchtip-backend/-/issues": {"gitlab", "glitchtip", "glitchtip-backend"},
		"git@gitlab.com:gitlab-org/gitlab-runner.git":             {"gitlab", "gitlab-org", "gitlab-runner"},
		"gitlab.com/gnome/world/podcasts/":                        {"gitlab", "gnome/world", "podcasts"},
		"https://www.gitlab.com/inkscape/inkscape#readme":         {"gitlab", "inkscape", "inkscape"},
	}
	for in, want := range ok {
		got, good := reg.Parse(in)
		if !good || got != want {
			t.Errorf("Parse(%q) = %+v,%v; want %+v", in, got, good, want)
		}
	}
	for _, in := range []string{"", "justone", "https://gitlab.com", "https://gitlab.com/solo", "a/..", "a/b c", "/", "https://bitbucket.org/a/b", "gitlab.com/a/../b"} {
		if r, good := reg.Parse(in); good {
			t.Errorf("Parse(%q) = %+v, should fail", in, r)
		}
	}
	if u := reg.URL("gitlab", "gnome/world", "podcasts"); u != "https://gitlab.com/gnome/world/podcasts" {
		t.Errorf("URL = %q", u)
	}
}
