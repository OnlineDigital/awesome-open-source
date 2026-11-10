// Package admin_category_controller implements category CRUD with per-locale names.
package admin_category_controller

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgconn"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		cats, err := app.Store.ListCategories(c.Context(), app.Cfg.DefaultLocale)
		if err != nil {
			return err
		}
		trs := map[int64]map[string]store.CategoryTranslation{}
		for _, cat := range cats {
			if trs[cat.ID], err = app.Store.CategoryTranslations(c.Context(), cat.ID); err != nil {
				return err
			}
		}
		return app.AdminPage(c, "Admin/Categories/Index", map[string]any{"categories": cats, "translations": trs})
	}
}

type request struct {
	Slug         string `json:"slug" form:"slug"`
	Icon         string `json:"icon" form:"icon"`
	SortOrder    int    `json:"sortOrder" form:"sortOrder"`
	Translations map[string]struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"translations"`
}

func parse(app *application.Application, c fiber.Ctx) (store.Category, []store.CategoryTranslation, map[string]string, error) {
	var req request
	if err := c.Bind().Body(&req); err != nil {
		return store.Category{}, nil, nil, fiber.ErrBadRequest
	}
	errs := map[string]string{}
	var trs []store.CategoryTranslation
	for loc, t := range req.Translations {
		if app.I18n.Has(loc) {
			trs = append(trs, store.CategoryTranslation{Locale: loc, Name: strings.TrimSpace(t.Name), Description: strings.TrimSpace(t.Description)})
		}
	}
	def := ""
	for _, t := range trs {
		if t.Locale == app.Cfg.DefaultLocale {
			def = t.Name
		}
	}
	if def == "" {
		errs["name"] = "The " + app.Cfg.DefaultLocale + " name is required."
	}
	slug := store.Slugify(req.Slug)
	if slug == "" {
		slug = store.Slugify(def)
	}
	if slug == "" {
		errs["slug"] = "A slug is required."
	}
	return store.Category{Slug: slug, Icon: strings.TrimSpace(req.Icon), SortOrder: req.SortOrder}, trs, errs, nil
}

func save(app *application.Application, c fiber.Ctx, id int64) error {
	cat, trs, errs, err := parse(app, c)
	if err != nil {
		return err
	}
	cat.ID = id
	if len(errs) > 0 {
		return app.Redirect303(c, "/admin/categories", errs, "")
	}
	if _, err := app.Store.SaveCategory(c.Context(), cat, trs); err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return app.Redirect303(c, "/admin/categories", map[string]string{"slug": "This slug is already used."}, "")
		}
		return err
	}
	return app.Redirect303(c, "/admin/categories", nil, "Category saved.")
}

func Store(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error { return save(app, c, 0) }
}

func Update(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, err := strconv.ParseInt(application.RouteParameter(c, "id"), 10, 64)
		if err != nil {
			return fiber.ErrNotFound
		}
		return save(app, c, id)
	}
}

func Destroy(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, err := strconv.ParseInt(application.RouteParameter(c, "id"), 10, 64)
		if err != nil {
			return fiber.ErrNotFound
		}
		if err := app.Store.DeleteCategory(c.Context(), id); err != nil {
			return err
		}
		return app.Redirect303(c, "/admin/categories", nil, "Category deleted. Its projects are now uncategorized.")
	}
}

func Translate(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, err := strconv.ParseInt(application.RouteParameter(c, "id"), 10, 64)
		if err != nil {
			return fiber.ErrNotFound
		}
		if !app.Translator.Enabled() {
			return app.Redirect303(c, "/admin/categories", map[string]string{"translate": "Set ANTHROPIC_API_KEY to enable machine translation."}, "")
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Minute)
		defer cancel()
		done, err := app.TranslateCategory(ctx, id, false)
		if err != nil {
			return app.Redirect303(c, "/admin/categories", map[string]string{"translate": err.Error()}, "")
		}
		return app.Redirect303(c, "/admin/categories", nil, "Translated into: "+strings.Join(done, ", "))
	}
}
