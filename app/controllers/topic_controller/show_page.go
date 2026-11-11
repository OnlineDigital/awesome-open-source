// Package topic_controller serves the public topic pages with their feature comparison tables.
package topic_controller

import (
	"errors"
	"strconv"
	"strings"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/seo"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Show(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, loc := c.Context(), app.LocaleOf(c)
		t := func(k string, kv ...any) string { return app.I18n.T(loc, k, kv...) }
		slug := application.RouteParameter(c, "slug")
		topic, err := app.Store.TopicBySlug(ctx, slug, loc, true)
		if errors.Is(err, store.ErrNotFound) {
			return app.NotFound(c)
		} else if err != nil {
			return err
		}
		ps, _, err := app.Store.ListProjects(ctx, store.ProjectFilter{Locale: loc, Status: "published", Topic: topic.Slug, Sort: "score", Limit: 200})
		if err != nil {
			return err
		}
		features, err := app.Store.Features(ctx, topic.ID, loc)
		if err != nil {
			return err
		}
		values, err := app.Store.FeatureValues(ctx, topic.ID)
		if err != nil {
			return err
		}
		others, err := app.Store.ListTopics(ctx, loc, true)
		if err != nil {
			return err
		}
		table := app.BuildTopicTable(features, values, ps, loc)
		cards := app.Cards(ps, loc)

		path := "/topic/" + topic.Slug
		title := topic.Name
		desc := topic.Summary
		if desc == "" {
			desc = title + " — " + t("common.projects", "n", len(ps))
		}
		meta := app.NewMeta(loc, path, seo.Truncate(title, 55)+" | "+app.Cfg.SiteName, seo.Truncate(desc, 160))
		if locs, err := app.Store.TopicLocales(ctx); err == nil {
			meta.Alternates = app.AlternatesIn(path, locs[topic.ID])
		}
		if loc != app.Cfg.DefaultLocale && !topic.Translated {
			app.Untranslated(meta, path)
		}
		meta.Markdown = app.Cfg.BaseURL + app.Path(loc, "/llms/topic/"+topic.Slug+".md")
		items := make([]seo.ListItem, 0, len(cards))
		links := make([]seo.Link, 0, len(cards))
		for _, p := range cards {
			items = append(items, seo.ListItem{Name: p.FullName, URL: p.URL})
			links = append(links, seo.Link{Text: p.FullName, URL: p.Path, Note: p.Tagline + featureSummary(t, table, p.Slug)})
		}
		meta.JSONLD = []any{
			seo.CollectionPage(title, desc, meta.Canonical, loc, app.Cfg.BaseURL, items),
			seo.Breadcrumbs([]seo.Crumb{{Name: app.Cfg.SiteName, URL: app.Abs(loc, "/")}, {Name: title, URL: app.Abs(loc, path)}}),
		}
		sections := []seo.Section{}
		if body := strings.TrimSpace(topic.Body); body != "" {
			var paras []string
			for _, para := range strings.Split(body, "\n\n") {
				if para = strings.TrimSpace(para); para != "" {
					paras = append(paras, para)
				}
			}
			sections = append(sections, seo.Section{Heading: title, Paragraphs: paras})
		}
		sections = append(sections, seo.Section{Heading: t("common.projects", "n", len(ps)), Links: links})
		meta.Fallback = seo.Fallback(title, desc, sections)

		var rel []store.Topic
		for _, o := range others {
			if o.Slug != topic.Slug && o.ProjectCount > 0 {
				rel = append(rel, o)
			}
		}
		return app.Page(c, "Topic", path, inertia.Props{
			"topic": app.TopicCards([]store.Topic{*topic}, loc)[0], "body": topic.Body,
			"features": table.Features, "cells": table.Cells, "projects": cards, "others": app.TopicCards(rel, loc),
		}, meta)
	}
}

// featureSummary renders "(Feature: Yes; Other: Partial)" for the no-JS fallback.
func featureSummary(t func(string, ...any) string, tbl application.TopicTable, slug string) string {
	var parts []string
	for _, f := range tbl.Features {
		cell, ok := tbl.Cells[slug][itoa(f.ID)]
		if !ok {
			continue
		}
		v := cell.Note
		if cell.State != "text" {
			v = t("topic." + cell.State)
			if cell.Note != "" {
				v += " (" + cell.Note + ")"
			}
		}
		parts = append(parts, f.Name+": "+v)
	}
	if len(parts) == 0 {
		return ""
	}
	return " — " + strings.Join(parts, "; ")
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
