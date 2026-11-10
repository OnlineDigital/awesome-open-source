// Package store contains all SQL access. It has no HTTP or framework imports.
package store

import (
	"encoding/json"
	"time"
)

type Project struct {
	ID             int64           `json:"id"`
	Slug           string          `json:"slug"`
	Owner          string          `json:"owner"`
	Repo           string          `json:"repo"`
	FullName       string          `json:"fullName"`
	CategoryID     *int64          `json:"categoryId"`
	CategorySlug   string          `json:"categorySlug"`
	CategoryName   string          `json:"categoryName"`
	Homepage       string          `json:"homepage"`
	License        string          `json:"license"`
	Language       string          `json:"language"`
	Topics         []string        `json:"topics"`
	AvatarURL      string          `json:"avatarUrl"`
	Stars          int             `json:"stars"`
	Forks          int             `json:"forks"`
	OpenIssues     int             `json:"openIssues"`
	Archived       bool            `json:"archived"`
	DefaultBranch  string          `json:"defaultBranch"`
	PushedAt       *time.Time      `json:"pushedAt"`
	LastCommitAt   *time.Time      `json:"lastCommitAt"`
	LastReleaseAt  *time.Time      `json:"lastReleaseAt"`
	LastReleaseTag string          `json:"lastReleaseTag"`
	Score          *float64        `json:"score"`
	Tier           *int            `json:"tier"`
	Breakdown      json.RawMessage `json:"breakdown,omitempty"`
	YouTubeID      string          `json:"youtubeId"`
	Status         string          `json:"status"`
	Featured       bool            `json:"featured"`
	SyncedAt       *time.Time      `json:"syncedAt"`
	SyncError      string          `json:"syncError"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`

	// Localized (falls back to the default locale).
	Tagline     string `json:"tagline"`
	Description string `json:"description"`
	Machine     bool   `json:"machineTranslated"`
	Translated  bool   `json:"translated"` // text exists in the requested locale (not a fallback)

	Screenshots []Screenshot `json:"screenshots,omitempty"`
}

type Screenshot struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"projectId"`
	Path      string `json:"path"` // public URL path, e.g. /uploads/12/abc.webp
	Alt       string `json:"alt"`
	Position  int    `json:"position"`
}

type Translation struct {
	Locale      string `json:"locale"`
	Tagline     string `json:"tagline"`
	Description string `json:"description"`
	Source      string `json:"source"` // human | machine
}

type Category struct {
	ID           int64  `json:"id"`
	Slug         string `json:"slug"`
	Icon         string `json:"icon"`
	SortOrder    int    `json:"sortOrder"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ProjectCount int    `json:"projectCount"`
	Translated   bool   `json:"translated"`
}

type CategoryTranslation struct {
	Locale      string `json:"locale"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AdminUser struct {
	ID           int64
	Email        string
	PasswordHash string
}

type ProjectFilter struct {
	Locale     string
	CategoryID int64
	Query      string // admin list: substring match on name
	Status     string // "", draft, published
	MinScore   float64
	Featured   bool
	Sort       string // score | stars | new | name
	Limit      int
	Offset     int
}
