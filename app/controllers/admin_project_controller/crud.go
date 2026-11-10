package admin_project_controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/github"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgconn"
)

const pageSize = 30

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		page := max(1, application.QueryInt(c, "page", 1))
		f := store.ProjectFilter{Query: c.Query("q"), Status: c.Query("status"), CategoryID: int64(application.QueryInt(c, "category", 0)), Sort: "new", Limit: pageSize, Offset: (page - 1) * pageSize}
		if f.Status != "draft" && f.Status != "published" {
			f.Status = ""
		}
		ps, total, err := app.Store.ListProjects(c.Context(), f)
		if err != nil {
			return err
		}
		cats, err := app.Store.ListCategories(c.Context(), app.Cfg.DefaultLocale)
		if err != nil {
			return err
		}
		return app.AdminPage(c, "Admin/Projects/Index", map[string]any{
			"projects": ps, "total": total, "page": page, "pages": max(1, (total+pageSize-1)/pageSize), "categories": cats,
			"filters": map[string]any{"q": f.Query, "status": f.Status, "category": f.CategoryID},
		})
	}
}

type createRequest struct {
	URLs       string `json:"urls" form:"urls"`
	CategoryID int64  `json:"categoryId" form:"categoryId"`
}

// Store adds one or many GitHub repositories as drafts and syncs them in the background.
func Store(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req createRequest
		if err := c.Bind().Body(&req); err != nil {
			return fiber.ErrBadRequest
		}
		var catID *int64
		if req.CategoryID > 0 {
			catID = &req.CategoryID
		}
		var ids []int64
		added, existing := 0, 0
		var invalid []string
		for _, line := range strings.FieldsFunc(req.URLs, func(r rune) bool { return r == '\n' || r == '\r' || r == ',' || r == ' ' || r == '\t' }) {
			owner, name, ok := github.ParseRepo(line)
			if !ok {
				invalid = append(invalid, line)
				continue
			}
			id, err := app.Store.CreateProject(c.Context(), owner, name, catID)
			if errors.Is(err, store.ErrExists) {
				existing++
				continue
			} else if err != nil {
				return err
			}
			added++
			ids = append(ids, id)
		}
		if added == 0 && len(invalid) > 0 {
			return app.Redirect303(c, "/admin/projects", map[string]string{"urls": "Not a GitHub repository: " + strings.Join(invalid, ", ")}, "")
		}
		if added == 0 && existing == 0 {
			return app.Redirect303(c, "/admin/projects", map[string]string{"urls": "Enter at least one GitHub URL or owner/repo."}, "")
		}
		go syncIDs(app, ids)
		msg := fmt.Sprintf("Added %d project(s), %d already existed; fetching GitHub data and scores in the background.", added, existing)
		if len(invalid) > 0 {
			msg += fmt.Sprintf(" Skipped %d invalid entries.", len(invalid))
		}
		if added == 1 {
			return back(app, c, ids[0], nil, msg)
		}
		return app.Redirect303(c, "/admin/projects?status=draft", nil, msg)
	}
}

func syncIDs(app *application.Application, ids []int64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	for _, id := range ids {
		if err := app.Syncer.SyncOne(ctx, id); err != nil {
			log.Printf("admin: sync project %d: %v", id, err)
			if errors.Is(err, github.ErrRateLimited) {
				return
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func Edit(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		pid, err := id(c, "id")
		if err != nil {
			return err
		}
		p, err := app.Store.GetProject(c.Context(), pid, app.Cfg.DefaultLocale)
		if errors.Is(err, store.ErrNotFound) {
			return fiber.ErrNotFound
		} else if err != nil {
			return err
		}
		trs, err := app.Store.Translations(c.Context(), pid)
		if err != nil {
			return err
		}
		cats, err := app.Store.ListCategories(c.Context(), app.Cfg.DefaultLocale)
		if err != nil {
			return err
		}
		return app.AdminPage(c, "Admin/Projects/Edit", map[string]any{
			"project": p, "translations": trs, "categories": cats, "activity": application.ActivityOf(p),
			"publicPath": app.Path(app.Cfg.DefaultLocale, "/project/"+p.Slug),
		})
	}
}

type updateRequest struct {
	CategoryID int64  `json:"categoryId" form:"categoryId"`
	YouTube    string `json:"youtube" form:"youtube"`
	Status     string `json:"status" form:"status"`
	Featured   bool   `json:"featured" form:"featured"`
	Homepage   string `json:"homepage" form:"homepage"`
}

func Update(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		pid, err := id(c, "id")
		if err != nil {
			return err
		}
		var req updateRequest
		if err := c.Bind().Body(&req); err != nil {
			return fiber.ErrBadRequest
		}
		errs := map[string]string{}
		yt, ok := ParseYouTube(req.YouTube)
		if !ok {
			errs["youtube"] = "Enter a YouTube URL or an 11-character video id."
		}
		if req.Status != "draft" && req.Status != "published" {
			errs["status"] = "Invalid status."
		}
		if h := strings.TrimSpace(req.Homepage); h != "" && !validHTTPURL(h) {
			errs["homepage"] = "Enter a full http(s) URL."
		}
		if len(errs) > 0 {
			return back(app, c, pid, errs, "")
		}
		var catID *int64
		if req.CategoryID > 0 {
			catID = &req.CategoryID
		}
		if err := app.Store.UpdateProject(c.Context(), pid, store.ProjectEdit{CategoryID: catID, YouTubeID: yt, Status: req.Status, Featured: req.Featured, Homepage: strings.TrimSpace(req.Homepage)}); err != nil {
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "23503" {
				return back(app, c, pid, map[string]string{"categoryId": "That category no longer exists."}, "")
			}
			return err
		}
		reindex(app, pid)
		return back(app, c, pid, nil, "Project saved.")
	}
}

func Destroy(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		pid, err := id(c, "id")
		if err != nil {
			return err
		}
		paths, err := app.Store.DeleteProject(c.Context(), pid)
		if err != nil {
			return err
		}
		for _, p := range paths {
			removeUpload(app, p)
		}
		if err := app.Indexer.Remove(c.Context(), pid); err != nil {
			log.Printf("admin: remove %d from index: %v", pid, err)
		}
		return app.Redirect303(c, "/admin/projects", nil, "Project deleted.")
	}
}

// Sync refreshes GitHub data and score for one project immediately.
func Sync(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		pid, err := id(c, "id")
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
		defer cancel()
		if err := app.Syncer.SyncOne(ctx, pid); err != nil {
			return back(app, c, pid, map[string]string{"sync": "GitHub sync failed: " + err.Error()}, "")
		}
		return back(app, c, pid, nil, "Synced with GitHub; score recalculated.")
	}
}

func removeUpload(app *application.Application, publicPath string) {
	if !strings.HasPrefix(publicPath, "/uploads/") {
		return
	}
	rel := filepath.FromSlash(strings.TrimPrefix(publicPath, "/uploads/"))
	if clean := filepath.Clean(rel); clean == rel && !strings.HasPrefix(clean, "..") {
		_ = os.Remove(filepath.Join(app.Cfg.UploadDir, clean))
	}
}
