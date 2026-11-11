// Package api_controller exposes read-only JSON used by WebMCP tools and AI agents.
package api_controller

import (
	"errors"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
)

func locale(app *application.Application, c fiber.Ctx) string {
	if l := c.Query("locale"); app.I18n.Has(l) {
		return l
	}
	return app.Cfg.DefaultLocale
}

func public(c fiber.Ctx) {
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Cache-Control", "public, max-age=60")
}

// Search handles GET /api/search?q=&category=&min_score=&sort=&page=&locale=
func Search(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		public(c)
		loc := locale(app, c)
		res, err := app.RunSearch(c.Context(), loc, application.SearchParamsFrom(c))
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "search is temporarily unavailable"})
		}
		return c.JSON(res)
	}
}

func Categories(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		public(c)
		loc := locale(app, c)
		cs, err := app.Store.ListCategories(c.Context(), loc)
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"categories": app.CategoryCards(cs, loc)})
	}
}

func Project(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		public(c)
		loc := locale(app, c)
		p, err := app.Store.GetProjectBySlug(c.Context(), application.RouteParameter(c, "slug"), loc, true)
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
		} else if err != nil {
			return err
		}
		card := app.CardOf(p, loc)
		out := fiber.Map{"project": card, "description": p.Description, "license": p.License, "topics": p.Topics, "homepage": p.Homepage,
			"github": "https://github.com/" + p.FullName, "activity": application.ActivityOf(p), "markdown": app.Abs(loc, "/llms/"+p.Slug+".md")}
		if p.YouTubeID != "" {
			out["video"] = application.YouTubeWatch(p.YouTubeID)
		}
		return c.JSON(out)
	}
}

func Topics(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		public(c)
		loc := locale(app, c)
		ts, err := app.Store.ListTopics(c.Context(), loc, true)
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"topics": app.TopicCards(ts, loc)})
	}
}

// Topic returns a topic with its feature comparison table.
func Topic(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		public(c)
		ctx, loc := c.Context(), locale(app, c)
		topic, err := app.Store.TopicBySlug(ctx, application.RouteParameter(c, "slug"), loc, true)
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "topic not found"})
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
		tbl := app.BuildTopicTable(features, values, ps, loc)
		return c.JSON(fiber.Map{"topic": app.TopicCards([]store.Topic{*topic}, loc)[0], "body": topic.Body, "features": tbl.Features, "cells": tbl.Cells, "projects": app.Cards(ps, loc), "markdown": app.Abs(loc, "/llms/topic/"+topic.Slug+".md")})
	}
}
