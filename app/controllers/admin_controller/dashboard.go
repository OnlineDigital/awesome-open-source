package admin_controller

import (
	"context"
	"log"
	"time"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
)

func Dashboard(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		stats, err := app.Store.Stats(c.Context())
		if err != nil {
			return err
		}
		recent, _, err := app.Store.ListProjects(c.Context(), store.ProjectFilter{Sort: "new", Limit: 8})
		if err != nil {
			return err
		}
		return app.AdminPage(c, "Admin/Dashboard", map[string]any{
			"stats":  map[string]int{"projects": stats.Projects, "published": stats.Published, "categories": stats.Categories, "syncErrors": stats.SyncErrors, "neverSynced": stats.NeverSynced},
			"recent": recent,
		})
	}
}

// Reindex rebuilds the Typesense collections in the background.
func Reindex(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !app.Search.Enabled() {
			return app.Redirect303(c, "/admin", map[string]string{"reindex": "Typesense is not configured."}, "")
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			n, err := app.Indexer.Reindex(ctx)
			log.Printf("admin: reindex finished: %d projects, err=%v", n, err)
		}()
		return app.Redirect303(c, "/admin", nil, "Reindex started in the background.")
	}
}

// SyncStale refreshes outdated projects in the background.
func SyncStale(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
			defer cancel()
			ok, failed, err := app.Syncer.SyncStale(ctx, app.Cfg.SyncStaleAge, 500)
			log.Printf("admin: sync finished: %d ok, %d failed, err=%v", ok, failed, err)
		}()
		return app.Redirect303(c, "/admin", nil, "Sync started in the background.")
	}
}
