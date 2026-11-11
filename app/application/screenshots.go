package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const MaxUpload = 5 << 20

var imageExt = map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}

// SaveScreenshotBytes validates an image by content and stores it under UPLOAD_DIR for the project.
func (a *Application) SaveScreenshotBytes(ctx context.Context, projectID int64, data []byte, alt string) (int64, string, error) {
	if len(data) == 0 || len(data) > MaxUpload {
		return 0, "", ErrInvalid{"screenshot": "Image must be between 1 byte and 5 MB."}
	}
	ext, ok := imageExt[http.DetectContentType(data[:min(len(data), 512)])]
	if !ok {
		return 0, "", ErrInvalid{"screenshot": "Only PNG, JPEG, GIF and WebP images are accepted."}
	}
	dir := filepath.Join(a.Cfg.UploadDir, strconv.FormatInt(projectID, 10))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, "", err
	}
	rnd := make([]byte, 8)
	_, _ = rand.Read(rnd)
	name := hex.EncodeToString(rnd) + ext
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		return 0, "", err
	}
	alt = strings.TrimSpace(alt)
	if len(alt) > 200 {
		alt = alt[:200]
	}
	public := fmt.Sprintf("/uploads/%d/%s", projectID, name)
	id, err := a.Store.AddScreenshot(ctx, projectID, public, alt)
	if err != nil {
		_ = os.Remove(filepath.Join(dir, name))
		return 0, "", err
	}
	return id, public, nil
}

// RemoveUpload deletes a stored screenshot file given its public /uploads/ path.
func (a *Application) RemoveUpload(publicPath string) {
	if !strings.HasPrefix(publicPath, "/uploads/") {
		return
	}
	rel := filepath.FromSlash(strings.TrimPrefix(publicPath, "/uploads/"))
	if clean := filepath.Clean(rel); clean == rel && !strings.HasPrefix(clean, "..") {
		_ = os.Remove(filepath.Join(a.Cfg.UploadDir, clean))
	}
}
