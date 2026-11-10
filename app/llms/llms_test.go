package llms

import (
	"strings"
	"testing"
	"time"
)

func tr(key string, kv ...any) string { return key }

func TestRelativeLinkTree(t *testing.T) {
	idx := BuildIndex(IndexInput{T: tr, SiteName: "S", Description: "d", Categories: []Cat{{Slug: "db", Name: "DB", Count: 2}}, SitemapURL: "https://x/sitemap.xml", HomeURL: "https://x"})
	if !strings.Contains(idx, "[DB](llms/category/db.txt)") {
		t.Fatalf("index must link to category relatively:\n%s", idx)
	}
	sc := 4.6
	cat := BuildCategory(CategoryInput{T: tr, Name: "DB", PageURL: "https://x/category/db", Projects: []Proj{{Slug: "a-b", FullName: "a/b", Tagline: "fast", Score: &sc, TierLabel: "Thriving", Stars: 5}}})
	for _, want := range []string{"[a/b](../a-b.md)", "(../../llms.txt)", "4.6/5"} {
		if !strings.Contains(cat, want) {
			t.Errorf("category lacks %q:\n%s", want, cat)
		}
	}
	now := time.Now()
	proj := BuildProject(ProjectInput{T: tr, Slug: "a-b", FullName: "a/b", Tagline: "fast", Description: "long text", CategorySlug: "db", CategoryName: "DB",
		Score: &sc, TierLabel: "Thriving", LastCommit: &now, VideoURL: "https://youtu.be/x", Related: []Proj{{Slug: "c-d", FullName: "c/d"}}, Screenshots: []Shot{{URL: "https://x/u/1.png"}}})
	for _, want := range []string{"[DB](category/db.txt)", "(../llms.txt)", "[c/d](c-d.md)", "https://youtu.be/x", "long text"} {
		if !strings.Contains(proj, want) {
			t.Errorf("project lacks %q:\n%s", want, proj)
		}
	}
}
