package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/store"
)

func topicTools(app *application.Application, topic func(context.Context, string) (*store.Topic, error), project func(context.Context, string) (*store.Project, error)) []Tool {
	loc := app.Cfg.DefaultLocale
	return []Tool{
		{Name: "list_topics", Description: "List topic pages (comparison pages), including drafts.", Schema: obj(m{}),
			Handler: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return app.Store.ListTopics(ctx, loc, false)
			}},
		{Name: "get_topic", Description: "A topic with all translations, its features (comparison rows), member projects and the value matrix (projectId -> featureId -> {state, notes}).",
			Schema: obj(m{"topic": str(topicRef)}, "topic"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct{ Topic string }](raw)
				if err != nil {
					return nil, err
				}
				t, err := topic(ctx, a.Topic)
				if err != nil {
					return nil, err
				}
				trs, err := app.Store.TopicTranslations(ctx, t.ID)
				if err != nil {
					return nil, err
				}
				features, err := app.Store.Features(ctx, t.ID, loc)
				if err != nil {
					return nil, err
				}
				values, err := app.Store.FeatureValues(ctx, t.ID)
				if err != nil {
					return nil, err
				}
				ps, _, err := app.Store.ListProjects(ctx, store.ProjectFilter{Topic: t.Slug, Sort: "name", Limit: 100})
				if err != nil {
					return nil, err
				}
				projects := make([]m, 0, len(ps))
				for i := range ps {
					s := projectSummary(&ps[i])
					s["curated"] = contains(ps[i].ExtraTopics, t.Slug)
					projects = append(projects, s)
				}
				return m{"topic": t, "translations": trs, "features": features, "projects": projects, "values": values}, nil
			}},
		{Name: "save_topic", Description: "Create (no id) or update a topic page. translations: {locale: {name, summary, body}} (summary ≤400 chars, body is SEO text with blank-line paragraphs). Replaces all translations; the default locale name is required.",
			Schema: obj(m{"id": num("Existing topic id; omit to create."), "slug": str("URL slug. For topics that mirror a GitHub topic use that topic name, e.g. laravel-admin-panel."), "published": flag("Visible on the site."), "sort_order": num("Sort order."), "translations": textsSchema}, "translations"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					ID           int64
					Slug         string
					Published    bool
					SortOrder    int `json:"sort_order"`
					Translations map[string]application.TopicTexts
				}](raw)
				if err != nil {
					return nil, err
				}
				id, err := app.SaveTopic(ctx, application.TopicInput{ID: a.ID, Slug: a.Slug, Published: a.Published, SortOrder: a.SortOrder, Translations: a.Translations})
				return m{"id": id}, err
			}},
		{Name: "delete_topic", Description: "Delete a topic with its features and values.", Schema: obj(m{"topic": str(topicRef)}, "topic"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct{ Topic string }](raw)
				if err != nil {
					return nil, err
				}
				t, err := topic(ctx, a.Topic)
				if err != nil {
					return nil, err
				}
				return m{"ok": true}, app.Store.DeleteTopic(ctx, t.ID)
			}},
		{Name: "save_features", Description: "Replace a topic's feature list (the comparison rows). Entries with an existing id keep their values; entries without id are new; features left out are deleted with their values. Order follows the array. translations: {locale: {name, description}}.",
			Schema: obj(m{"topic": str(topicRef), "features": list(obj(m{"id": num("Existing feature id."), "translations": textsSchema}, "translations"), "Features in display order.")}, "topic", "features"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					Topic    string
					Features []application.FeatureInput
				}](raw)
				if err != nil {
					return nil, err
				}
				t, err := topic(ctx, a.Topic)
				if err != nil {
					return nil, err
				}
				fs, err := app.SaveFeatures(ctx, t.ID, a.Features)
				if err != nil {
					return nil, err
				}
				ids := make([]int64, 0, len(fs))
				for _, f := range fs {
					ids = append(ids, f.ID)
				}
				return m{"featureIds": ids}, nil
			}},
		{Name: "set_feature_value", Description: "Set one cell of the comparison table. state: yes | no | partial | text (only the note is shown); empty state with empty notes clears the cell. notes: {locale: text}, e.g. a paid first-party plugin remark per language.",
			Schema: obj(m{"topic": str(topicRef), "project": str(projectRef), "feature_id": num("Feature id."), "state": strEnum("Value.", "yes", "no", "partial", "text", ""), "notes": m{"type": "object", "description": "Map locale -> note text (max 500 chars).", "additionalProperties": map[string]any{"type": "string"}}}, "topic", "project", "feature_id"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					Topic, Project string
					FeatureID      int64 `json:"feature_id"`
					State          string
					Notes          map[string]string
				}](raw)
				if err != nil {
					return nil, err
				}
				t, err := topic(ctx, a.Topic)
				if err != nil {
					return nil, err
				}
				p, err := project(ctx, a.Project)
				if err != nil {
					return nil, err
				}
				if err := app.SetFeatureValue(ctx, t.ID, p.ID, a.FeatureID, a.State, a.Notes); err != nil {
					if errors.Is(err, store.ErrNotFound) {
						return nil, errors.New("that feature does not belong to this topic")
					}
					return nil, err
				}
				return m{"ok": true}, nil
			}},
		{Name: "add_project_to_topic", Description: "Add a project to a topic as a curated member; it stays even when the project is re-synced from GitHub.", Schema: obj(m{"topic": str(topicRef), "project": str(projectRef)}, "topic", "project"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct{ Topic, Project string }](raw)
				if err != nil {
					return nil, err
				}
				t, err := topic(ctx, a.Topic)
				if err != nil {
					return nil, err
				}
				p, err := app.AddProjectToTopic(ctx, t.ID, a.Project)
				if err != nil {
					return nil, err
				}
				return m{"ok": true, "project": p.FullName}, nil
			}},
		{Name: "remove_project_from_topic", Description: "Remove a project's curated membership. Projects that carry the GitHub topic stay in the table.", Schema: obj(m{"topic": str(topicRef), "project": str(projectRef)}, "topic", "project"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct{ Topic, Project string }](raw)
				if err != nil {
					return nil, err
				}
				t, err := topic(ctx, a.Topic)
				if err != nil {
					return nil, err
				}
				p, err := project(ctx, a.Project)
				if err != nil {
					return nil, err
				}
				return m{"ok": true}, app.Store.RemoveExtraTopic(ctx, p.ID, t.Slug)
			}},
		{Name: "translate_topic", Description: "Machine-translate topic texts, feature names and cell notes into missing locales (needs ANTHROPIC_API_KEY).", Schema: obj(m{"topic": str(topicRef), "force": flag("Overwrite existing translations.")}, "topic"),
			Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
				a, err := decode[struct {
					Topic string
					Force bool
				}](raw)
				if err != nil {
					return nil, err
				}
				if !app.Translator.Enabled() {
					return nil, errors.New("machine translation is not configured (ANTHROPIC_API_KEY)")
				}
				t, err := topic(ctx, a.Topic)
				if err != nil {
					return nil, err
				}
				done, err := app.TranslateTopic(ctx, t.ID, a.Force)
				return m{"translated": done}, err
			}},
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}
