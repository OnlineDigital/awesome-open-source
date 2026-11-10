package application

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"awesomeopensource.dev/app/scoring"
	"awesomeopensource.dev/app/search"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
)

// Card is the compact project representation used in lists, search and the API.
type Card struct {
	Slug         string     `json:"slug"`
	Name         string     `json:"name"`
	FullName     string     `json:"fullName"`
	Tagline      string     `json:"tagline"`
	Category     string     `json:"category"`
	CategorySlug string     `json:"categorySlug"`
	Score        *float64   `json:"score"`
	Tier         *int       `json:"tier"`
	Stars        int        `json:"stars"`
	Language     string     `json:"language"`
	Avatar       string     `json:"avatar"`
	HasVideo     bool       `json:"hasVideo"`
	Archived     bool       `json:"archived"`
	LastCommitAt *time.Time `json:"lastCommitAt"`
	Path         string     `json:"path"`
	URL          string     `json:"url"`
}

func (a *Application) CardOf(p *store.Project, locale string) Card {
	return Card{
		Slug: p.Slug, Name: p.Repo, FullName: p.FullName, Tagline: p.Tagline, Category: p.CategoryName, CategorySlug: p.CategorySlug,
		Score: p.Score, Tier: p.Tier, Stars: p.Stars, Language: p.Language, Avatar: p.AvatarURL, HasVideo: p.YouTubeID != "", Archived: p.Archived,
		LastCommitAt: p.LastCommitAt, Path: a.Path(locale, "/project/"+p.Slug), URL: a.Abs(locale, "/project/"+p.Slug),
	}
}

func (a *Application) Cards(ps []store.Project, locale string) []Card {
	out := make([]Card, 0, len(ps))
	for i := range ps {
		out = append(out, a.CardOf(&ps[i], locale))
	}
	return out
}

type CategoryCard struct {
	Slug         string `json:"slug"`
	Icon         string `json:"icon"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ProjectCount int    `json:"projectCount"`
	Path         string `json:"path"`
	URL          string `json:"url"`
}

func (a *Application) CategoryCards(cs []store.Category, locale string) []CategoryCard {
	out := make([]CategoryCard, 0, len(cs))
	for _, c := range cs {
		out = append(out, CategoryCard{Slug: c.Slug, Icon: c.Icon, Name: c.Name, Description: c.Description, ProjectCount: c.ProjectCount,
			Path: a.Path(locale, "/category/"+c.Slug), URL: a.Abs(locale, "/category/"+c.Slug)})
	}
	return out
}

// Activity is the parsed score explanation stored with a project.
type Activity struct {
	Score          float64             `json:"score"`
	Tier           int                 `json:"tier"`
	Label          string              `json:"label"`
	Cap            string              `json:"cap,omitempty"`
	Components     []scoring.Component `json:"components"`
	Commits30      int                 `json:"commits30"`
	Commits90      int                 `json:"commits90"`
	Commits365     int                 `json:"commits365"`
	ActiveBranches int                 `json:"activeBranches90"`
	TotalBranches  int                 `json:"totalBranches"`
	Releases12m    int                 `json:"releases12m"`
	AllBranches    bool                `json:"allBranches"`
	LastCommitAt   *time.Time          `json:"lastCommitAt"`
	LastReleaseAt  *time.Time          `json:"lastReleaseAt"`
	LastReleaseTag string              `json:"lastReleaseTag"`
}

// ActivityOf returns nil for projects that were never synced.
func ActivityOf(p *store.Project) *Activity {
	if p.Score == nil || len(p.Breakdown) == 0 {
		return nil
	}
	var raw struct {
		Result  scoring.Result  `json:"result"`
		Signals scoring.Signals `json:"signals"`
		GraphQL bool            `json:"graphql"`
	}
	if err := json.Unmarshal(p.Breakdown, &raw); err != nil {
		return nil
	}
	n12 := 0
	cut := time.Now().AddDate(-1, 0, 0)
	for _, r := range raw.Signals.Releases {
		if r.PublishedAt.After(cut) && !r.Prerelease {
			n12++
		}
	}
	return &Activity{
		Score: *p.Score, Tier: raw.Result.Tier, Label: raw.Result.Label, Cap: raw.Result.Cap, Components: raw.Result.Components,
		Commits30: raw.Signals.Commits30, Commits90: raw.Signals.Commits90, Commits365: raw.Signals.Commits365,
		ActiveBranches: raw.Signals.ActiveBranches90, TotalBranches: raw.Signals.TotalBranches, Releases12m: n12, AllBranches: raw.GraphQL,
		LastCommitAt: p.LastCommitAt, LastReleaseAt: p.LastReleaseAt, LastReleaseTag: p.LastReleaseTag,
	}
}

// YouTube helpers.
func YouTubeThumb(id string) string { return "https://i.ytimg.com/vi/" + id + "/hqdefault.jpg" }
func YouTubeEmbed(id string) string { return "https://www.youtube-nocookie.com/embed/" + id }
func YouTubeWatch(id string) string { return "https://www.youtube.com/watch?v=" + id }

// ---- search ----

type SearchResult struct {
	Found    int                       `json:"found"`
	Page     int                       `json:"page"`
	PerPage  int                       `json:"perPage"`
	Projects []Card                    `json:"projects"`
	Facets   map[string][]search.Facet `json:"facets"`
}

func SearchParamsFrom(c fiber.Ctx) search.Params {
	p := search.Params{Query: strings.TrimSpace(c.Query("q")), Category: c.Query("category"), Language: c.Query("language"), Sort: c.Query("sort"), Page: QueryInt(c, "page", 1), PerPage: 24}
	if len(p.Query) > 200 {
		p.Query = p.Query[:200]
	}
	if v, err := strconv.ParseFloat(c.Query("min_score", c.Query("min")), 64); err == nil && v >= 1 && v <= 5 {
		p.MinScore = v
	}
	if l := QueryInt(c, "limit", 0); l > 0 {
		p.PerPage = min(l, 50)
	}
	return p
}

// RunSearch queries Typesense and converts hits to cards.
func (a *Application) RunSearch(ctx context.Context, locale string, p search.Params) (*SearchResult, error) {
	if !a.Search.Enabled() {
		return nil, search.ErrUnavailable
	}
	res, err := a.Search.Search(ctx, locale, p)
	if err != nil {
		return nil, err
	}
	out := &SearchResult{Found: res.Found, Page: res.Page, PerPage: p.PerPage, Facets: res.Facets, Projects: make([]Card, 0, len(res.Hits))}
	for _, d := range res.Hits {
		c := Card{Slug: d.Slug, Name: d.Name, FullName: d.FullName, Tagline: d.Tagline, Category: d.Category, CategorySlug: d.CategorySlug,
			Stars: int(d.Stars), Language: d.Language, Avatar: d.Avatar, HasVideo: d.HasVideo, Path: a.Path(locale, "/project/"+d.Slug), URL: a.Abs(locale, "/project/"+d.Slug)}
		score, tier := d.Score, int(d.Tier)
		c.Score, c.Tier = &score, &tier
		out.Projects = append(out.Projects, c)
	}
	return out, nil
}
