// Package admin_project_controller implements project CRUD, translations, screenshots and sync.
package admin_project_controller

import (
	"context"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"awesomeopensource.dev/app/application"
	"github.com/gofiber/fiber/v3"
)

var youtubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
var youtubeURL = regexp.MustCompile(`(?:v=|youtu\.be/|/embed/|/shorts/|/live/)([A-Za-z0-9_-]{11})`)

// ParseYouTube accepts a bare 11-character id or any common YouTube URL. Empty input is valid (no video).
func ParseYouTube(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", true
	}
	if youtubeID.MatchString(s) {
		return s, true
	}
	if m := youtubeURL.FindStringSubmatch(s); m != nil {
		return m[1], true
	}
	return "", false
}

func validHTTPURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func id(c fiber.Ctx, key string) (int64, error) {
	n, err := strconv.ParseInt(application.RouteParameter(c, key), 10, 64)
	if err != nil || n < 1 {
		return 0, fiber.ErrNotFound
	}
	return n, nil
}

// reindex refreshes one project in Typesense without delaying the response.
func reindex(app *application.Application, projectID int64) {
	if !app.Search.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := app.Indexer.IndexProject(ctx, projectID); err != nil {
			log.Printf("admin: index project %d: %v", projectID, err)
		}
	}()
}

func back(app *application.Application, c fiber.Ctx, projectID int64, errs map[string]string, ok string) error {
	return app.Redirect303(c, "/admin/projects/"+strconv.FormatInt(projectID, 10), errs, ok)
}
