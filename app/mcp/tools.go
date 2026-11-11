package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/controllers/admin_project_controller"
	"awesomeopensource.dev/app/github"
	"awesomeopensource.dev/app/store"
)

type Tool struct {
	Name        string
	Description string
	Schema      map[string]any
	Handler     func(ctx context.Context, args json.RawMessage) (any, error)
}

// ---- schema helpers ----

type m = map[string]any

func obj(props m, required ...string) m {
	s := m{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}
func str(desc string) m  { return m{"type": "string", "description": desc} }
func num(desc string) m  { return m{"type": "integer", "description": desc} }
func flag(desc string) m { return m{"type": "boolean", "description": desc} }
func list(item m, desc string) m {
	return m{"type": "array", "items": item, "description": desc}
}
func strEnum(desc string, values ...string) m {
	return m{"type": "string", "enum": values, "description": desc}
}

const (
	projectRef = "Project id, slug (owner-repo), or owner/repo / GitHub URL."
	topicRef   = "Topic id or slug."
)

var textsSchema = m{"type": "object", "description": "Map of locale code to texts. The default locale is required.", "additionalProperties": true}

func decode[T any](raw json.RawMessage) (T, error) {
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, fmt.Errorf("invalid arguments: %w", err)
	}
	return v, nil
}

func projectSummary(p *store.Project) m {
	return m{"id": p.ID, "slug": p.Slug, "fullName": p.FullName, "status": p.Status, "featured": p.Featured, "category": p.CategorySlug, "score": p.Score,
		"stars": p.Stars, "language": p.Language, "githubTopics": p.Topics, "extraTopics": p.ExtraTopics, "youtubeId": p.YouTubeID, "tagline": p.Tagline, "syncError": p.SyncError}
}

func buildTools(app *application.Application) []Tool {
	ctxLoc := app.Cfg.DefaultLocale
	project := func(ctx context.Context, ref string) (*store.Project, error) {
		p, err := app.ResolveProject(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("project %q not found", ref)
		}
		return p, nil
	}
	topic := func(ctx context.Context, ref string) (*store.Topic, error) {
		var t *store.Topic
		var err error
		var id int64
		if _, e := fmt.Sscan(ref, &id); e == nil && fmt.Sprint(id) == strings.TrimSpace(ref) {
			t, err = app.Store.TopicByID(ctx, id, ctxLoc)
		} else {
			t, err = app.Store.TopicBySlug(ctx, store.Slugify(ref), ctxLoc, false)
		}
		if err != nil {
			return nil, fmt.Errorf("topic %q not found", ref)
		}
		return t, nil
	}
	reindex := func(id int64) {
		if !app.Search.Enabled() {
			return
		}
		go func() {
			c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := app.Indexer.IndexProject(c, id); err != nil {
				log.Printf("mcp: index project %d: %v", id, err)
			}
		}()
	}

	tools := []Tool{
		{Name: "get_stats", Description: "Catalog counters: projects, published, categories, sync errors.", Schema: obj(m{}),
			Handler: func(ctx context.Context, _ json.RawMessage) (any, error) {
				st, err := app.Store.Stats(ctx)
				return m{"projects": st.Projects, "published": st.Published, "categories": st.Categories, "syncErrors": st.SyncErrors, "neverSynced": st.NeverSynced, "locales": app.I18n.Locales, "defaultLocale": ctxLoc}, err
			}},

		// ---- projects ----
		{Name: "list_projects", Description: "List projects (all statuses). Filter by text, status, category id, or topic slug.",
			Schema: obj(m{"query": str("Substring of slug or owner."), "status": strEnum("Filter by status.", "draft", "published"), "category_id": num("Category id."), "topic": str("GitHub or curated topic slug."),
				"sort": strEnum("Sort order.", "new", "score", "stars", "name"), "limit": num("Max 100, default 30."), "offset": num("Offset.")}),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					Query, Status, Topic, Sort string
					CategoryID                 int64 `json:"category_id"`
					Limit, Offset              int
				}](raw)
				if err != nil {
					return nil, err
				}
				if a.Limit <= 0 || a.Limit > 100 {
					a.Limit = 30
				}
				ps, total, err := app.Store.ListProjects(ctx, store.ProjectFilter{Query: a.Query, Status: a.Status, CategoryID: a.CategoryID, Topic: a.Topic, Sort: firstNonEmpty(a.Sort, "new"), Limit: a.Limit, Offset: a.Offset})
				if err != nil {
					return nil, err
				}
				out := make([]m, 0, len(ps))
				for i := range ps {
					out = append(out, projectSummary(&ps[i]))
				}
				return m{"total": total, "projects": out}, nil
			}},
		{Name: "get_project", Description: "Full project record with per-locale texts, screenshots and score breakdown.", Schema: obj(m{"project": str(projectRef)}, "project"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct{ Project string }](raw)
				if err != nil {
					return nil, err
				}
				p, err := project(ctx, a.Project)
				if err != nil {
					return nil, err
				}
				trs, err := app.Store.Translations(ctx, p.ID)
				if err != nil {
					return nil, err
				}
				return m{"project": p, "translations": trs, "activity": application.ActivityOf(p)}, nil
			}},
		{Name: "add_projects", Description: "Add GitHub repositories as drafts (GitHub data and scores are fetched in the background). Optionally publish immediately.",
			Schema: obj(m{"repositories": list(str("GitHub URL or owner/repo"), "Repositories to add."), "category_id": num("Category id for all of them."), "publish": flag("Publish right away instead of leaving drafts.")}, "repositories"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					Repositories []string
					CategoryID   int64 `json:"category_id"`
					Publish      bool
				}](raw)
				if err != nil {
					return nil, err
				}
				var cat *int64
				if a.CategoryID > 0 {
					cat = &a.CategoryID
				}
				added, existing := []string{}, []string{}
				invalid := []string{}
				var ids []int64
				for _, r := range a.Repositories {
					owner, name, ok := github.ParseRepo(r)
					if !ok {
						invalid = append(invalid, r)
						continue
					}
					id, err := app.Store.CreateProject(ctx, owner, name, cat)
					if errors.Is(err, store.ErrExists) {
						existing = append(existing, owner+"/"+name)
						continue
					} else if err != nil {
						return nil, err
					}
					added = append(added, owner+"/"+name)
					ids = append(ids, id)
					if a.Publish {
						_ = app.Store.UpdateProject(ctx, id, store.ProjectEdit{CategoryID: cat, Status: "published"})
					}
				}
				go func() {
					c, cancel := context.WithTimeout(context.Background(), time.Hour)
					defer cancel()
					for _, id := range ids {
						if err := app.Syncer.SyncOne(c, id); err != nil {
							log.Printf("mcp: sync project %d: %v", id, err)
							if errors.Is(err, github.ErrRateLimited) {
								return
							}
						}
						time.Sleep(300 * time.Millisecond)
					}
				}()
				return m{"added": added, "alreadyExisted": existing, "invalid": invalid, "ids": ids}, nil
			}},
		{Name: "update_project", Description: "Update project settings. Only the provided fields change.",
			Schema: obj(m{"project": str(projectRef), "category_id": num("Category id; 0 clears it."), "youtube": str("YouTube URL or 11-char id; empty clears the video."), "status": strEnum("Visibility.", "draft", "published"),
				"featured": flag("Featured flag."), "homepage": str("Homepage URL.")}, "project"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					Project    string
					CategoryID *int64 `json:"category_id"`
					YouTube    *string
					Status     *string
					Featured   *bool
					Homepage   *string
				}](raw)
				if err != nil {
					return nil, err
				}
				p, err := project(ctx, a.Project)
				if err != nil {
					return nil, err
				}
				e := store.ProjectEdit{CategoryID: p.CategoryID, YouTubeID: p.YouTubeID, Status: p.Status, Featured: p.Featured, Homepage: p.Homepage}
				if a.CategoryID != nil {
					if *a.CategoryID > 0 {
						e.CategoryID = a.CategoryID
					} else {
						e.CategoryID = nil
					}
				}
				if a.YouTube != nil {
					yt, ok := admin_project_controller.ParseYouTube(*a.YouTube)
					if !ok {
						return nil, errors.New("youtube must be a YouTube URL or an 11-character video id")
					}
					e.YouTubeID = yt
				}
				if a.Status != nil {
					if *a.Status != "draft" && *a.Status != "published" {
						return nil, errors.New("status must be draft or published")
					}
					e.Status = *a.Status
				}
				if a.Featured != nil {
					e.Featured = *a.Featured
				}
				if a.Homepage != nil {
					e.Homepage = strings.TrimSpace(*a.Homepage)
				}
				if err := app.Store.UpdateProject(ctx, p.ID, e); err != nil {
					return nil, err
				}
				reindex(p.ID)
				return m{"ok": true}, nil
			}},
		{Name: "set_project_text", Description: "Set the tagline and description of a project in one locale (marked as human-written). Empty tagline and description delete the locale.",
			Schema: obj(m{"project": str(projectRef), "locale": str("Locale code, e.g. en or ro."), "tagline": str("Short tagline (max 300)."), "description": str("Long description; paragraphs separated by blank lines (max 8000).")}, "project", "locale"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct{ Project, Locale, Tagline, Description string }](raw)
				if err != nil {
					return nil, err
				}
				if !app.I18n.Has(a.Locale) {
					return nil, fmt.Errorf("unknown locale %q (available: %s)", a.Locale, strings.Join(app.I18n.Locales, ", "))
				}
				a.Tagline, a.Description = strings.TrimSpace(a.Tagline), strings.TrimSpace(a.Description)
				if len(a.Tagline) > 300 || len(a.Description) > 8000 {
					return nil, errors.New("text too long (tagline max 300, description max 8000)")
				}
				p, err := project(ctx, a.Project)
				if err != nil {
					return nil, err
				}
				if a.Tagline == "" && a.Description == "" {
					err = app.Store.DeleteTranslation(ctx, p.ID, a.Locale)
				} else {
					err = app.Store.SaveTranslation(ctx, p.ID, store.Translation{Locale: a.Locale, Tagline: a.Tagline, Description: a.Description, Source: "human"})
				}
				reindex(p.ID)
				return m{"ok": err == nil}, err
			}},
		{Name: "translate_project", Description: "Machine-translate the default-locale text into other locales (needs ANTHROPIC_API_KEY). Human texts are kept unless force is true.",
			Schema: obj(m{"project": str(projectRef), "locales": list(str("locale"), "Locales; all when empty."), "force": flag("Overwrite human translations.")}, "project"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					Project string
					Locales []string
					Force   bool
				}](raw)
				if err != nil {
					return nil, err
				}
				if !app.Translator.Enabled() {
					return nil, errors.New("machine translation is not configured (ANTHROPIC_API_KEY)")
				}
				p, err := project(ctx, a.Project)
				if err != nil {
					return nil, err
				}
				done, err := app.TranslateProject(ctx, p.ID, a.Locales, a.Force)
				reindex(p.ID)
				return m{"translated": done}, err
			}},
		{Name: "set_project_extra_topics", Description: "Set the curated topics of a project. Unlike GitHub topics they are kept when the project is re-synced.",
			Schema: obj(m{"project": str(projectRef), "topics": list(str("topic slug"), "Complete list of extra topic slugs.")}, "project", "topics"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					Project string
					Topics  []string
				}](raw)
				if err != nil {
					return nil, err
				}
				p, err := project(ctx, a.Project)
				if err != nil {
					return nil, err
				}
				return m{"ok": true}, app.Store.SetExtraTopics(ctx, p.ID, a.Topics)
			}},
		{Name: "delete_project", Description: "Delete a project permanently, with its translations, screenshots and comparison values.", Schema: obj(m{"project": str(projectRef)}, "project"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct{ Project string }](raw)
				if err != nil {
					return nil, err
				}
				p, err := project(ctx, a.Project)
				if err != nil {
					return nil, err
				}
				paths, err := app.Store.DeleteProject(ctx, p.ID)
				if err != nil {
					return nil, err
				}
				for _, path := range paths {
					app.RemoveUpload(path)
				}
				_ = app.Indexer.Remove(ctx, p.ID)
				return m{"deleted": p.FullName}, nil
			}},
		{Name: "sync_project", Description: "Refresh GitHub data and recalculate the maintenance score of one project now.", Schema: obj(m{"project": str(projectRef)}, "project"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct{ Project string }](raw)
				if err != nil {
					return nil, err
				}
				p, err := project(ctx, a.Project)
				if err != nil {
					return nil, err
				}
				if err := app.Syncer.SyncOne(ctx, p.ID); err != nil {
					return nil, fmt.Errorf("GitHub sync failed: %w", err)
				}
				p, _ = app.Store.GetProject(ctx, p.ID, ctxLoc)
				return projectSummary(p), nil
			}},
		{Name: "sync_stale_projects", Description: "Start a background sync of projects whose GitHub data is outdated.", Schema: obj(m{}),
			Handler: func(context.Context, json.RawMessage) (any, error) {
				go func() {
					c, cancel := context.WithTimeout(context.Background(), time.Hour)
					defer cancel()
					ok, failed, err := app.Syncer.SyncStale(c, app.Cfg.SyncStaleAge, 500)
					log.Printf("mcp: sync finished: %d ok, %d failed, err=%v", ok, failed, err)
				}()
				return m{"started": true}, nil
			}},
		{Name: "reindex_search", Description: "Rebuild the Typesense search index in the background.", Schema: obj(m{}),
			Handler: func(context.Context, json.RawMessage) (any, error) {
				if !app.Search.Enabled() {
					return nil, errors.New("Typesense is not configured")
				}
				go func() {
					c, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
					defer cancel()
					n, err := app.Indexer.Reindex(c)
					log.Printf("mcp: reindex finished: %d projects, err=%v", n, err)
				}()
				return m{"started": true}, nil
			}},
		{Name: "add_screenshot", Description: "Upload a screenshot (PNG, JPEG, GIF or WebP up to 5 MB) as base64.",
			Schema: obj(m{"project": str(projectRef), "image_base64": str("Base64-encoded image bytes (no data: prefix)."), "alt": str("Alt text (max 200).")}, "project", "image_base64"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					Project     string
					ImageBase64 string `json:"image_base64"`
					Alt         string
				}](raw)
				if err != nil {
					return nil, err
				}
				data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(a.ImageBase64))
				if err != nil {
					return nil, errors.New("image_base64 is not valid base64")
				}
				p, err := project(ctx, a.Project)
				if err != nil {
					return nil, err
				}
				id, path, err := app.SaveScreenshotBytes(ctx, p.ID, data, a.Alt)
				if err != nil {
					return nil, err
				}
				reindex(p.ID)
				return m{"id": id, "path": path}, nil
			}},
		{Name: "update_screenshot", Description: "Change a screenshot's alt text or position.", Schema: obj(m{"project": str(projectRef), "screenshot_id": num("Screenshot id."), "alt": str("Alt text."), "position": num("Sort position (0 first).")}, "project", "screenshot_id"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					Project      string
					ScreenshotID int64 `json:"screenshot_id"`
					Alt          string
					Position     int
				}](raw)
				if err != nil {
					return nil, err
				}
				p, err := project(ctx, a.Project)
				if err != nil {
					return nil, err
				}
				if len(a.Alt) > 200 {
					a.Alt = a.Alt[:200]
				}
				return m{"ok": true}, app.Store.UpdateScreenshot(ctx, a.ScreenshotID, p.ID, strings.TrimSpace(a.Alt), a.Position)
			}},
		{Name: "delete_screenshot", Description: "Delete a screenshot and its file.", Schema: obj(m{"project": str(projectRef), "screenshot_id": num("Screenshot id.")}, "project", "screenshot_id"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					Project      string
					ScreenshotID int64 `json:"screenshot_id"`
				}](raw)
				if err != nil {
					return nil, err
				}
				p, err := project(ctx, a.Project)
				if err != nil {
					return nil, err
				}
				path, err := app.Store.DeleteScreenshot(ctx, a.ScreenshotID, p.ID)
				if err != nil && !errors.Is(err, store.ErrNotFound) {
					return nil, err
				}
				app.RemoveUpload(path)
				return m{"ok": true}, nil
			}},

		// ---- categories ----
		{Name: "list_categories", Description: "List categories with all translations.", Schema: obj(m{}),
			Handler: func(ctx context.Context, _ json.RawMessage) (any, error) {
				cats, err := app.Store.ListCategories(ctx, ctxLoc)
				if err != nil {
					return nil, err
				}
				out := make([]m, 0, len(cats))
				for _, c := range cats {
					trs, err := app.Store.CategoryTranslations(ctx, c.ID)
					if err != nil {
						return nil, err
					}
					out = append(out, m{"id": c.ID, "slug": c.Slug, "icon": c.Icon, "sortOrder": c.SortOrder, "projectCount": c.ProjectCount, "translations": trs})
				}
				return out, nil
			}},
		{Name: "save_category", Description: "Create (no id) or update a category. translations: {locale: {name, description}}; replaces all existing translations.",
			Schema: obj(m{"id": num("Existing category id; omit to create."), "slug": str("URL slug; derived from the default-locale name when empty."), "icon": str("Emoji or short icon text."), "sort_order": num("Sort order."), "translations": textsSchema}, "translations"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					ID           int64
					Slug, Icon   string
					SortOrder    int `json:"sort_order"`
					Translations map[string]struct{ Name, Description string }
				}](raw)
				if err != nil {
					return nil, err
				}
				var trs []store.CategoryTranslation
				def := ""
				for loc, t := range a.Translations {
					if !app.I18n.Has(loc) {
						continue
					}
					trs = append(trs, store.CategoryTranslation{Locale: loc, Name: strings.TrimSpace(t.Name), Description: strings.TrimSpace(t.Description)})
					if loc == ctxLoc {
						def = strings.TrimSpace(t.Name)
					}
				}
				if def == "" {
					return nil, fmt.Errorf("the %s name is required", ctxLoc)
				}
				slug := store.Slugify(firstNonEmpty(a.Slug, def))
				id, err := app.Store.SaveCategory(ctx, store.Category{ID: a.ID, Slug: slug, Icon: strings.TrimSpace(a.Icon), SortOrder: a.SortOrder}, trs)
				if err != nil {
					return nil, err
				}
				return m{"id": id, "slug": slug}, nil
			}},
		{Name: "delete_category", Description: "Delete a category; its projects become uncategorized.", Schema: obj(m{"id": num("Category id.")}, "id"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct{ ID int64 }](raw)
				if err != nil {
					return nil, err
				}
				return m{"ok": true}, app.Store.DeleteCategory(ctx, a.ID)
			}},
		{Name: "translate_category", Description: "Machine-translate a category into missing locales.", Schema: obj(m{"id": num("Category id."), "force": flag("Overwrite existing translations.")}, "id"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					ID    int64
					Force bool
				}](raw)
				if err != nil {
					return nil, err
				}
				if !app.Translator.Enabled() {
					return nil, errors.New("machine translation is not configured (ANTHROPIC_API_KEY)")
				}
				done, err := app.TranslateCategory(ctx, a.ID, a.Force)
				return m{"translated": done}, err
			}},
	}
	return append(tools, topicTools(app, topic, project)...)
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
