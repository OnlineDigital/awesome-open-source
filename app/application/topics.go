package application

import (
	"strconv"
	"strings"

	"awesomeopensource.dev/app/store"
)

type TopicCard struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Summary      string `json:"summary"`
	ProjectCount int    `json:"projectCount"`
	Path         string `json:"path"`
	URL          string `json:"url"`
}

func (a *Application) TopicCards(ts []store.Topic, locale string) []TopicCard {
	out := make([]TopicCard, 0, len(ts))
	for _, t := range ts {
		out = append(out, TopicCard{Slug: t.Slug, Name: t.Name, Summary: t.Summary, ProjectCount: t.ProjectCount,
			Path: a.Path(locale, "/topic/"+t.Slug), URL: a.Abs(locale, "/topic/"+t.Slug)})
	}
	return out
}

// Cell is one table cell: a state and the note resolved for the locale (falling back to the default locale).
type Cell struct {
	State string `json:"state"`
	Note  string `json:"note,omitempty"`
}

type FeatureRow struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// TopicTable is the localized comparison matrix of a topic page: rows are features, values are keyed project slug -> feature id.
type TopicTable struct {
	Features []FeatureRow               `json:"features"`
	Cells    map[string]map[string]Cell `json:"cells"`
}

// BuildTopicTable resolves feature names and cell notes for a locale.
func (a *Application) BuildTopicTable(features []store.Feature, values map[int64]map[int64]store.FeatureValue, projects []store.Project, locale string) TopicTable {
	tbl := TopicTable{Features: make([]FeatureRow, 0, len(features)), Cells: map[string]map[string]Cell{}}
	for _, f := range features {
		tbl.Features = append(tbl.Features, FeatureRow{ID: f.ID, Name: f.Name, Description: f.Description})
	}
	for _, p := range projects {
		row := map[string]Cell{}
		for _, f := range features {
			v, ok := values[p.ID][f.ID]
			if !ok {
				continue
			}
			note := strings.TrimSpace(v.Notes[locale])
			if note == "" {
				note = strings.TrimSpace(v.Notes[a.Cfg.DefaultLocale])
			}
			row[strconv.FormatInt(f.ID, 10)] = Cell{State: v.State, Note: note}
		}
		tbl.Cells[p.Slug] = row
	}
	return tbl
}
