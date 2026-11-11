package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"awesomeopensource.dev/app/github"
	"awesomeopensource.dev/app/store"
	"github.com/jackc/pgx/v5/pgconn"
)

// TopicTexts is the editable text of a topic in one locale.
type TopicTexts struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Body    string `json:"body"`
}

type TopicInput struct {
	ID           int64                 `json:"id"`
	Slug         string                `json:"slug"`
	Published    bool                  `json:"published"`
	SortOrder    int                   `json:"sortOrder"`
	Translations map[string]TopicTexts `json:"translations"`
}

// ErrInvalid carries user-facing validation errors keyed by field.
type ErrInvalid map[string]string

func (e ErrInvalid) Error() string {
	parts := make([]string, 0, len(e))
	for k, v := range e {
		parts = append(parts, k+": "+v)
	}
	return strings.Join(parts, "; ")
}

// SaveTopic validates and stores a topic with all its translations.
func (a *Application) SaveTopic(ctx context.Context, in TopicInput) (int64, error) {
	errs := ErrInvalid{}
	var trs []store.TopicTranslation
	def := ""
	for loc, t := range in.Translations {
		if !a.I18n.Has(loc) {
			continue
		}
		tr := store.TopicTranslation{Locale: loc, Name: strings.TrimSpace(t.Name), Summary: strings.TrimSpace(t.Summary), Body: strings.TrimSpace(t.Body)}
		if len(tr.Summary) > 400 {
			errs["summary"] = "Summary is too long (max 400 characters)."
		}
		if len(tr.Body) > 20000 {
			errs["body"] = "Text is too long (max 20000 characters)."
		}
		if loc == a.Cfg.DefaultLocale {
			def = tr.Name
		}
		trs = append(trs, tr)
	}
	if def == "" {
		errs["name"] = "The " + a.Cfg.DefaultLocale + " name is required."
	}
	slug := store.Slugify(in.Slug)
	if slug == "" {
		slug = store.Slugify(def)
	}
	if slug == "" {
		errs["slug"] = "A slug is required."
	}
	if len(errs) > 0 {
		return 0, errs
	}
	id, err := a.Store.SaveTopic(ctx, store.Topic{ID: in.ID, Slug: slug, Published: in.Published, SortOrder: in.SortOrder}, trs)
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return 0, ErrInvalid{"slug": "This slug is already used."}
	}
	return id, err
}

type FeatureInput struct {
	ID           int64                        `json:"id"`
	Translations map[string]store.FeatureText `json:"translations"`
}

// SaveFeatures replaces a topic's feature list (new ids are created, omitted ones deleted with their values).
func (a *Application) SaveFeatures(ctx context.Context, topicID int64, in []FeatureInput) ([]store.Feature, error) {
	list := make([]store.Feature, 0, len(in))
	for i, f := range in {
		trs := map[string]store.FeatureText{}
		for loc, t := range f.Translations {
			if a.I18n.Has(loc) {
				trs[loc] = store.FeatureText{Name: strings.TrimSpace(t.Name), Description: strings.TrimSpace(t.Description)}
			}
		}
		if strings.TrimSpace(trs[a.Cfg.DefaultLocale].Name) == "" {
			return nil, ErrInvalid{"features": fmt.Sprintf("Feature %d needs a %s name.", i+1, a.Cfg.DefaultLocale)}
		}
		list = append(list, store.Feature{ID: f.ID, Translations: trs})
	}
	return a.Store.SaveFeatures(ctx, topicID, list)
}

// SetFeatureValue stores a project's value for a feature. An empty state with no notes clears it.
func (a *Application) SetFeatureValue(ctx context.Context, topicID, projectID, featureID int64, state string, notes map[string]string) error {
	if state != "" && !store.FeatureStates[state] {
		return ErrInvalid{"state": "State must be yes, no, partial or text."}
	}
	clean := map[string]string{}
	for loc, n := range notes {
		if a.I18n.Has(loc) {
			if len(n) > 500 {
				return ErrInvalid{"notes": "A note is too long (max 500 characters)."}
			}
			clean[loc] = n
		}
	}
	return a.Store.SetFeatureValue(ctx, topicID, projectID, featureID, state, clean)
}

// ResolveProject finds a project by numeric id, slug, or "owner/repo".
func (a *Application) ResolveProject(ctx context.Context, ref string) (*store.Project, error) {
	ref = strings.TrimSpace(ref)
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return a.Store.GetProject(ctx, id, a.Cfg.DefaultLocale)
	}
	if owner, repo, ok := github.ParseRepo(ref); ok {
		ref = store.Slugify(owner + "-" + repo)
	}
	return a.Store.GetProjectBySlug(ctx, store.Slugify(ref), a.Cfg.DefaultLocale, false)
}

// AddProjectToTopic adds the topic to the project's curated topics (kept across GitHub resyncs).
func (a *Application) AddProjectToTopic(ctx context.Context, topicID int64, ref string) (*store.Project, error) {
	t, err := a.Store.TopicByID(ctx, topicID, a.Cfg.DefaultLocale)
	if err != nil {
		return nil, err
	}
	p, err := a.ResolveProject(ctx, ref)
	if err != nil {
		return nil, ErrInvalid{"project": "No project matches \"" + ref + "\"."}
	}
	return p, a.Store.AddExtraTopic(ctx, p.ID, t.Slug)
}

// TranslateTopic machine-translates the topic texts, feature names and cell notes into missing locales.
func (a *Application) TranslateTopic(ctx context.Context, id int64, force bool) ([]string, error) {
	trs, err := a.Store.TopicTranslations(ctx, id)
	if err != nil {
		return nil, err
	}
	src, ok := trs[a.Cfg.DefaultLocale]
	if !ok || src.Name == "" {
		return nil, fmt.Errorf("add the %s name first", a.Cfg.DefaultLocale)
	}
	topic, err := a.Store.TopicByID(ctx, id, a.Cfg.DefaultLocale)
	if err != nil {
		return nil, err
	}
	features, err := a.Store.Features(ctx, id, a.Cfg.DefaultLocale)
	if err != nil {
		return nil, err
	}
	values, err := a.Store.FeatureValues(ctx, id)
	if err != nil {
		return nil, err
	}
	var done []string
	for _, loc := range a.I18n.Locales {
		if loc == a.Cfg.DefaultLocale {
			continue
		}
		if cur, exists := trs[loc]; exists && cur.Name != "" && !force {
			continue
		}
		in := map[string]string{"name": src.Name, "summary": src.Summary, "body": src.Body}
		for _, f := range features {
			d := f.Translations[a.Cfg.DefaultLocale]
			in["f"+strconv.FormatInt(f.ID, 10)+"_name"], in["f"+strconv.FormatInt(f.ID, 10)+"_desc"] = d.Name, d.Description
		}
		for pid, row := range values {
			for fid, v := range row {
				if n := v.Notes[a.Cfg.DefaultLocale]; n != "" {
					in["n"+strconv.FormatInt(pid, 10)+"_"+strconv.FormatInt(fid, 10)] = n
				}
			}
		}
		out, err := a.Translator.Fields(ctx, a.Cfg.DefaultLocale, loc, in)
		if err != nil {
			return done, fmt.Errorf("%s: %w", loc, err)
		}
		trs[loc] = store.TopicTranslation{Locale: loc, Name: out["name"], Summary: out["summary"], Body: out["body"]}
		for i := range features {
			k := "f" + strconv.FormatInt(features[i].ID, 10)
			if features[i].Translations == nil {
				features[i].Translations = map[string]store.FeatureText{}
			}
			features[i].Translations[loc] = store.FeatureText{Name: out[k+"_name"], Description: out[k+"_desc"]}
		}
		for pid, row := range values {
			for fid, v := range row {
				if out2, has := out["n"+strconv.FormatInt(pid, 10)+"_"+strconv.FormatInt(fid, 10)]; has && out2 != "" {
					v.Notes[loc] = out2
				}
			}
		}
		done = append(done, loc)
	}
	list := make([]store.TopicTranslation, 0, len(trs))
	for _, t := range trs {
		list = append(list, t)
	}
	if _, err := a.Store.SaveTopic(ctx, *topic, list); err != nil {
		return done, err
	}
	if _, err := a.Store.SaveFeatures(ctx, id, features); err != nil {
		return done, err
	}
	for pid, row := range values {
		for fid, v := range row {
			if err := a.Store.SetFeatureValue(ctx, id, pid, fid, v.State, v.Notes); err != nil {
				return done, err
			}
		}
	}
	return done, nil
}
