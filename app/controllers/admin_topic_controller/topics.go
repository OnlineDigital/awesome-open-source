// Package admin_topic_controller manages topic pages: texts, comparison features, per-project values and curated membership.
package admin_topic_controller

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
)

func id(c fiber.Ctx, key string) (int64, error) {
	n, err := strconv.ParseInt(application.RouteParameter(c, key), 10, 64)
	if err != nil || n < 1 {
		return 0, fiber.ErrNotFound
	}
	return n, nil
}

func back(app *application.Application, c fiber.Ctx, topicID int64, errs map[string]string, ok string) error {
	return app.Redirect303(c, "/admin/topics/"+strconv.FormatInt(topicID, 10), errs, ok)
}

func asErrs(err error) (map[string]string, bool) {
	var inv application.ErrInvalid
	if errors.As(err, &inv) {
		return inv, true
	}
	return nil, false
}

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		ts, err := app.Store.ListTopics(c.Context(), app.Cfg.DefaultLocale, false)
		if err != nil {
			return err
		}
		return app.AdminPage(c, "Admin/Topics/Index", map[string]any{"topics": ts})
	}
}

// Store creates a topic from a slug and a default-locale name, then opens its editor.
func Store(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			Slug string `json:"slug" form:"slug"`
			Name string `json:"name" form:"name"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return fiber.ErrBadRequest
		}
		tid, err := app.SaveTopic(c.Context(), application.TopicInput{Slug: req.Slug, Translations: map[string]application.TopicTexts{app.Cfg.DefaultLocale: {Name: req.Name}}})
		if errs, ok := asErrs(err); ok {
			return app.Redirect303(c, "/admin/topics", errs, "")
		} else if err != nil {
			return err
		}
		return back(app, c, tid, nil, "Topic created. Add its texts and features below.")
	}
}

func Edit(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()
		tid, err := id(c, "id")
		if err != nil {
			return err
		}
		topic, err := app.Store.TopicByID(ctx, tid, app.Cfg.DefaultLocale)
		if errors.Is(err, store.ErrNotFound) {
			return fiber.ErrNotFound
		} else if err != nil {
			return err
		}
		trs, err := app.Store.TopicTranslations(ctx, tid)
		if err != nil {
			return err
		}
		features, err := app.Store.Features(ctx, tid, app.Cfg.DefaultLocale)
		if err != nil {
			return err
		}
		values, err := app.Store.FeatureValues(ctx, tid)
		if err != nil {
			return err
		}
		ps, _, err := app.Store.ListProjects(ctx, store.ProjectFilter{Topic: topic.Slug, Sort: "name", Limit: 200})
		if err != nil {
			return err
		}
		type row struct {
			ID       int64    `json:"id"`
			Slug     string   `json:"slug"`
			FullName string   `json:"fullName"`
			Avatar   string   `json:"avatar"`
			Status   string   `json:"status"`
			Score    *float64 `json:"score"`
			Curated  bool     `json:"curated"`
			GitHub   bool     `json:"github"`
		}
		rows := make([]row, 0, len(ps))
		for _, p := range ps {
			rows = append(rows, row{ID: p.ID, Slug: p.Slug, FullName: p.FullName, Avatar: p.AvatarURL, Status: p.Status, Score: p.Score,
				Curated: slices.Contains(p.ExtraTopics, topic.Slug), GitHub: slices.Contains(p.Topics, topic.Slug)})
		}
		return app.AdminPage(c, "Admin/Topics/Edit", map[string]any{
			"topic": topic, "translations": trs, "features": features, "projects": rows, "values": values,
			"publicPath": app.Path(app.Cfg.DefaultLocale, "/topic/"+topic.Slug),
		})
	}
}

func Update(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		tid, err := id(c, "id")
		if err != nil {
			return err
		}
		var in application.TopicInput
		if err := c.Bind().Body(&in); err != nil {
			return fiber.ErrBadRequest
		}
		in.ID = tid
		if _, err := app.SaveTopic(c.Context(), in); err != nil {
			if errs, ok := asErrs(err); ok {
				return back(app, c, tid, errs, "")
			}
			return err
		}
		return back(app, c, tid, nil, "Topic saved.")
	}
}

func Destroy(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		tid, err := id(c, "id")
		if err != nil {
			return err
		}
		if err := app.Store.DeleteTopic(c.Context(), tid); err != nil {
			return err
		}
		return app.Redirect303(c, "/admin/topics", nil, "Topic deleted.")
	}
}

func SaveFeatures(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		tid, err := id(c, "id")
		if err != nil {
			return err
		}
		var req struct {
			Features []application.FeatureInput `json:"features"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return fiber.ErrBadRequest
		}
		if _, err := app.SaveFeatures(c.Context(), tid, req.Features); err != nil {
			if errs, ok := asErrs(err); ok {
				return back(app, c, tid, errs, "")
			}
			return err
		}
		return back(app, c, tid, nil, "Features saved.")
	}
}

// SetValue saves one cell of the comparison matrix.
func SetValue(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		tid, err := id(c, "id")
		if err != nil {
			return err
		}
		var req struct {
			ProjectID int64             `json:"projectId"`
			FeatureID int64             `json:"featureId"`
			State     string            `json:"state"`
			Notes     map[string]string `json:"notes"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return fiber.ErrBadRequest
		}
		if err := app.SetFeatureValue(c.Context(), tid, req.ProjectID, req.FeatureID, req.State, req.Notes); err != nil {
			if errs, ok := asErrs(err); ok {
				return back(app, c, tid, errs, "")
			}
			if errors.Is(err, store.ErrNotFound) {
				return fiber.ErrNotFound
			}
			return err
		}
		return back(app, c, tid, nil, "")
	}
}

// AddProject adds a project (by slug, id or owner/repo) to the topic as a curated member that survives GitHub resyncs.
func AddProject(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		tid, err := id(c, "id")
		if err != nil {
			return err
		}
		var req struct {
			Project string `json:"project" form:"project"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return fiber.ErrBadRequest
		}
		p, err := app.AddProjectToTopic(c.Context(), tid, req.Project)
		if errs, ok := asErrs(err); ok {
			return back(app, c, tid, errs, "")
		} else if err != nil {
			return err
		}
		return back(app, c, tid, nil, p.FullName+" added to this topic.")
	}
}

func RemoveProject(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		tid, err := id(c, "id")
		if err != nil {
			return err
		}
		pid, err := id(c, "project")
		if err != nil {
			return err
		}
		t, err := app.Store.TopicByID(c.Context(), tid, app.Cfg.DefaultLocale)
		if err != nil {
			return fiber.ErrNotFound
		}
		if err := app.Store.RemoveExtraTopic(c.Context(), pid, t.Slug); err != nil {
			return err
		}
		return back(app, c, tid, nil, "Removed from the curated list. Projects that carry this GitHub topic stay in the table.")
	}
}

func Translate(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		tid, err := id(c, "id")
		if err != nil {
			return err
		}
		if !app.Translator.Enabled() {
			return back(app, c, tid, map[string]string{"translate": "Set ANTHROPIC_API_KEY to enable machine translation."}, "")
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Minute)
		defer cancel()
		done, err := app.TranslateTopic(ctx, tid, false)
		if err != nil {
			return back(app, c, tid, map[string]string{"translate": err.Error()}, "")
		}
		return back(app, c, tid, nil, "Translated into: "+strings.Join(done, ", "))
	}
}

// ProjectTopics saves the curated (resync-proof) topics of one project from the project editor.
func ProjectTopics(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		pid, err := id(c, "id")
		if err != nil {
			return err
		}
		var req struct {
			Topics string `json:"topics" form:"topics"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return fiber.ErrBadRequest
		}
		slugs := strings.FieldsFunc(req.Topics, func(r rune) bool { return r == ',' || r == '\n' || r == ' ' })
		if err := app.Store.SetExtraTopics(c.Context(), pid, slugs); err != nil {
			return err
		}
		return app.Redirect303(c, "/admin/projects/"+strconv.FormatInt(pid, 10), nil, "Extra topics saved. They are kept when the project is re-synced from GitHub.")
	}
}
