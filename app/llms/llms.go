// Package llms renders the AI-agent facing documents:
//
//	/llms.txt                        index with links to categories
//	/llms/category/{slug}.txt        category with links to every project
//	/llms/{project-slug}.md          project details with links back and to similar projects
//
// All cross-links are relative, so an agent can walk the whole site from
// /llms.txt without ever constructing a URL (and the tree works under /{locale}/ too).
package llms

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"awesomeopensource.dev/app/scoring"
)

type T func(key string, kv ...any) string

type Cat struct {
	Slug, Name, Description string
	Count                   int
}

type Proj struct {
	Slug, FullName, Tagline, Language string
	Score                             *float64
	TierLabel                         string
	Stars                             int
}

type IndexInput struct {
	T                              T
	SiteName, SiteURL, Description string
	Categories                     []Cat
	SitemapURL, HomeURL            string
}

func link(text, url string) string {
	text = strings.NewReplacer("[", "(", "]", ")", "\n", " ").Replace(text)
	return "[" + text + "](" + url + ")"
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func BuildIndex(in IndexInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n> %s\n\n", in.SiteName, oneLine(in.Description))
	b.WriteString(in.T("llms.intro") + "\n\n")
	fmt.Fprintf(&b, "## %s\n\n", in.T("nav.categories"))
	for _, c := range in.Categories {
		fmt.Fprintf(&b, "- %s: %s (%s)\n", link(c.Name, "llms/category/"+c.Slug+".txt"), oneLine(c.Description), in.T("common.projects", "n", c.Count))
	}
	fmt.Fprintf(&b, "\n## Optional\n\n- %s\n- %s\n", link("Sitemap", in.SitemapURL), link(in.SiteName+" (HTML)", in.HomeURL))
	return b.String()
}

type CategoryInput struct {
	T                          T
	Name, Description, PageURL string
	Projects                   []Proj
}

func scoreText(t T, p Proj) string {
	if p.Score == nil {
		return ""
	}
	return fmt.Sprintf("%s %.1f/5 (%s)", t("score.title"), *p.Score, p.TierLabel)
}

func BuildCategory(in CategoryInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", in.Name)
	if d := oneLine(in.Description); d != "" {
		fmt.Fprintf(&b, "> %s\n\n", d)
	}
	fmt.Fprintf(&b, "%s: %s · %s\n\n", in.T("llms.navigation"), link(in.T("llms.index"), "../../llms.txt"), link("HTML", in.PageURL))
	fmt.Fprintf(&b, "## %s\n\n", in.T("common.projects", "n", len(in.Projects)))
	for _, p := range in.Projects {
		parts := []string{oneLine(p.Tagline), scoreText(in.T, p)}
		if p.Language != "" {
			parts = append(parts, p.Language)
		}
		if p.Stars > 0 {
			parts = append(parts, "★ "+strconv.Itoa(p.Stars))
		}
		fmt.Fprintf(&b, "- %s: %s\n", link(p.FullName, "../"+p.Slug+".md"), joinNonEmpty(parts, " · "))
	}
	return b.String()
}

type Shot struct{ URL, Alt string }

type ProjectInput struct {
	T                                     T
	Slug, FullName, Tagline, Description  string
	CategorySlug, CategoryName            string
	PageURL, GitHubURL, Homepage, License string
	Language                              string
	Topics                                []string
	Stars                                 int
	Archived                              bool
	Score                                 *float64
	TierLabel, Cap                        string
	Components                            []scoring.Component
	LastCommit, LastRelease               *time.Time
	LastReleaseTag                        string
	Commits90, Commits365                 int
	VideoURL                              string
	Screenshots                           []Shot
	Related                               []Proj
	Machine                               bool
}

func date(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}

func BuildProject(in ProjectInput) string {
	t := in.T
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", in.FullName)
	if tg := oneLine(in.Tagline); tg != "" {
		fmt.Fprintf(&b, "> %s\n\n", tg)
	}
	nav := []string{link(t("llms.index"), "../llms.txt")}
	if in.CategorySlug != "" {
		nav = append([]string{link(in.CategoryName, "category/"+in.CategorySlug+".txt")}, nav...)
	}
	fmt.Fprintf(&b, "%s: %s\n\n", t("llms.navigation"), strings.Join(nav, " · "))

	b.WriteString("## " + t("llms.facts") + "\n\n")
	fact := func(k, v string) {
		if v != "" {
			fmt.Fprintf(&b, "- %s: %s\n", k, v)
		}
	}
	fact("GitHub", in.GitHubURL)
	fact("HTML", in.PageURL)
	fact(t("project.homepage"), in.Homepage)
	fact(t("project.license"), in.License)
	fact(t("project.language"), in.Language)
	if in.Stars > 0 {
		fact(t("common.stars"), strconv.Itoa(in.Stars))
	}
	fact(t("project.topics"), strings.Join(in.Topics, ", "))
	if in.Archived {
		fact("Status", "archived")
	}
	if in.Score != nil {
		fact(t("score.title"), fmt.Sprintf("%.1f/5 (%s)", *in.Score, in.TierLabel))
	}
	fact(t("project.last_commit"), date(in.LastCommit))
	if in.LastRelease != nil {
		fact(t("project.last_release"), strings.TrimSpace(in.LastReleaseTag+" "+date(in.LastRelease)))
	}
	b.WriteString("\n")

	if d := strings.TrimSpace(in.Description); d != "" {
		b.WriteString("## " + t("llms.description") + "\n\n" + d + "\n\n")
		if in.Machine {
			b.WriteString("_" + t("project.machine_translated") + "_\n\n")
		}
	}
	if in.Score != nil && len(in.Components) > 0 {
		b.WriteString("## " + t("project.activity") + "\n\n")
		for _, c := range in.Components {
			if !c.Used {
				continue
			}
			fmt.Fprintf(&b, "- %s: %.0f%% (weight %.0f%%)", t("score."+c.Name), c.Value*100, c.Weight*100)
			if c.Evidence != "" {
				b.WriteString(" — " + c.Evidence)
			}
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "- Commits: %d / 90d, %d / 365d\n", in.Commits90, in.Commits365)
		if in.Cap != "" {
			fmt.Fprintf(&b, "- Cap applied: %s\n", in.Cap)
		}
		b.WriteString("\n")
	}
	if in.VideoURL != "" {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", t("project.demo"), in.VideoURL)
	}
	if len(in.Screenshots) > 0 {
		b.WriteString("## " + t("project.screenshots") + "\n\n")
		for _, s := range in.Screenshots {
			alt := s.Alt
			if alt == "" {
				alt = in.FullName
			}
			fmt.Fprintf(&b, "- %s\n", link(alt, s.URL))
		}
		b.WriteString("\n")
	}
	if len(in.Related) > 0 {
		b.WriteString("## " + t("project.related") + "\n\n")
		for _, r := range in.Related {
			fmt.Fprintf(&b, "- %s: %s\n", link(r.FullName, r.Slug+".md"), joinNonEmpty([]string{oneLine(r.Tagline), scoreText(t, r)}, " · "))
		}
	}
	return b.String()
}

func joinNonEmpty(parts []string, sep string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}
