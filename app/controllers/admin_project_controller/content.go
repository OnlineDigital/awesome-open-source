package admin_project_controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
)

type translationsRequest struct {
	Translations map[string]struct {
		Tagline     string `json:"tagline"`
		Description string `json:"description"`
	} `json:"translations"`
}

// SaveTranslations stores the texts edited by a human for every locale. Empty texts delete the locale row.
func SaveTranslations(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		pid, err := id(c, "id")
		if err != nil {
			return err
		}
		var req translationsRequest
		if err := c.Bind().Body(&req); err != nil {
			return fiber.ErrBadRequest
		}
		existing, err := app.Store.Translations(c.Context(), pid)
		if err != nil {
			return err
		}
		for loc, t := range req.Translations {
			if !app.I18n.Has(loc) {
				continue
			}
			tagline, desc := strings.TrimSpace(t.Tagline), strings.TrimSpace(t.Description)
			if len(tagline) > 300 || len(desc) > 8000 {
				return back(app, c, pid, map[string]string{"translations": "Text too long (tagline max 300, description max 8000 characters)."}, "")
			}
			cur, had := existing[loc]
			switch {
			case tagline == "" && desc == "":
				if had {
					if err := app.Store.DeleteTranslation(c.Context(), pid, loc); err != nil {
						return err
					}
				}
			case had && cur.Tagline == tagline && cur.Description == desc:
				// unchanged: keep the "machine" marker
			default:
				if err := app.Store.SaveTranslation(c.Context(), pid, store.Translation{Locale: loc, Tagline: tagline, Description: desc, Source: "human"}); err != nil {
					return err
				}
			}
		}
		reindex(app, pid)
		return back(app, c, pid, nil, "Texts saved.")
	}
}

// Translate machine-translates the default-locale text into the other locales.
func Translate(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		pid, err := id(c, "id")
		if err != nil {
			return err
		}
		if !app.Translator.Enabled() {
			return back(app, c, pid, map[string]string{"translate": "Set ANTHROPIC_API_KEY to enable machine translation."}, "")
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Minute)
		defer cancel()
		done, err := app.TranslateProject(ctx, pid, nil, c.Query("force") == "1")
		if err != nil {
			return back(app, c, pid, map[string]string{"translate": err.Error()}, "")
		}
		reindex(app, pid)
		if len(done) == 0 {
			return back(app, c, pid, nil, "Nothing to translate: every locale already has text (human edits are never overwritten).")
		}
		return back(app, c, pid, nil, "Translated into: "+strings.Join(done, ", ")+".")
	}
}

const maxUpload = 5 << 20

var imageExt = map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}

// UploadScreenshot validates the image by content (not by file name) and stores it under UPLOAD_DIR.
func UploadScreenshot(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		pid, err := id(c, "id")
		if err != nil {
			return err
		}
		fh, err := c.FormFile("file")
		if err != nil {
			return back(app, c, pid, map[string]string{"screenshot": "Choose an image file."}, "")
		}
		if fh.Size > maxUpload {
			return back(app, c, pid, map[string]string{"screenshot": "Image is larger than 5 MB."}, "")
		}
		f, err := fh.Open()
		if err != nil {
			return err
		}
		defer f.Close()
		head := make([]byte, 512)
		n, _ := io.ReadFull(f, head)
		ext, ok := imageExt[http.DetectContentType(head[:n])]
		if !ok {
			return back(app, c, pid, map[string]string{"screenshot": "Only PNG, JPEG, GIF and WebP images are accepted."}, "")
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		dir := filepath.Join(app.Cfg.UploadDir, strconv.FormatInt(pid, 10))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		rnd := make([]byte, 8)
		_, _ = rand.Read(rnd)
		name := hex.EncodeToString(rnd) + ext
		dst, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dst, io.LimitReader(f, maxUpload+1)); err != nil {
			dst.Close()
			return err
		}
		if err := dst.Close(); err != nil {
			return err
		}
		alt := strings.TrimSpace(c.FormValue("alt"))
		if len(alt) > 200 {
			alt = alt[:200]
		}
		if _, err := app.Store.AddScreenshot(c.Context(), pid, fmt.Sprintf("/uploads/%d/%s", pid, name), alt); err != nil {
			_ = os.Remove(filepath.Join(dir, name))
			return err
		}
		reindex(app, pid)
		return back(app, c, pid, nil, "Screenshot added.")
	}
}

type screenshotRequest struct {
	Alt      string `json:"alt" form:"alt"`
	Position int    `json:"position" form:"position"`
}

func UpdateScreenshot(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		pid, err := id(c, "id")
		if err != nil {
			return err
		}
		sid, err := id(c, "shot")
		if err != nil {
			return err
		}
		var req screenshotRequest
		if err := c.Bind().Body(&req); err != nil {
			return fiber.ErrBadRequest
		}
		if len(req.Alt) > 200 {
			req.Alt = req.Alt[:200]
		}
		if err := app.Store.UpdateScreenshot(c.Context(), sid, pid, strings.TrimSpace(req.Alt), req.Position); err != nil {
			return err
		}
		return back(app, c, pid, nil, "Screenshot updated.")
	}
}

func DeleteScreenshot(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		pid, err := id(c, "id")
		if err != nil {
			return err
		}
		sid, err := id(c, "shot")
		if err != nil {
			return err
		}
		path, err := app.Store.DeleteScreenshot(c.Context(), sid, pid)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return err
		}
		removeUpload(app, path)
		return back(app, c, pid, nil, "Screenshot removed.")
	}
}
