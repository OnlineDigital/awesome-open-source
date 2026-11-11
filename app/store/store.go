package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"awesomeopensource.dev/app/github"
	"awesomeopensource.dev/app/scoring"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = pgx.ErrNoRows

type Store struct {
	DB            *pgxpool.Pool
	DefaultLocale string
}

func New(db *pgxpool.Pool, defaultLocale string) *Store {
	return &Store{DB: db, DefaultLocale: defaultLocale}
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify lowercases and collapses non-alphanumerics to single hyphens.
func Slugify(s string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

const projectSelect = `
SELECT p.id, p.slug, p.github_owner, p.github_repo, p.category_id, COALESCE(c.slug,''), COALESCE(ct.name, cd.name, c.slug, ''),
  p.homepage, p.license, p.language, p.topics, p.extra_topics, p.avatar_url, p.stars, p.forks, p.open_issues, p.archived, p.default_branch,
  p.pushed_at, p.last_commit_at, p.last_release_at, p.last_release_tag,
  p.activity_score::float8, p.activity_tier::int, p.score_breakdown, p.youtube_id, p.status, p.featured, p.synced_at, p.sync_error, p.created_at, p.updated_at,
  COALESCE(NULLIF(t.tagline,''), d.tagline, ''), COALESCE(NULLIF(t.description,''), d.description, ''), COALESCE(t.source = 'machine' AND t.tagline <> '', false),
  COALESCE(t.tagline <> '' OR t.description <> '', false)
FROM projects p
LEFT JOIN project_translations t ON t.project_id = p.id AND t.locale = $1
LEFT JOIN project_translations d ON d.project_id = p.id AND d.locale = $2
LEFT JOIN categories c ON c.id = p.category_id
LEFT JOIN category_translations ct ON ct.category_id = c.id AND ct.locale = $1
LEFT JOIN category_translations cd ON cd.category_id = c.id AND cd.locale = $2`

func scanProject(row pgx.Row) (*Project, error) {
	var p Project
	var breakdown []byte
	err := row.Scan(&p.ID, &p.Slug, &p.Owner, &p.Repo, &p.CategoryID, &p.CategorySlug, &p.CategoryName,
		&p.Homepage, &p.License, &p.Language, &p.Topics, &p.ExtraTopics, &p.AvatarURL, &p.Stars, &p.Forks, &p.OpenIssues, &p.Archived, &p.DefaultBranch,
		&p.PushedAt, &p.LastCommitAt, &p.LastReleaseAt, &p.LastReleaseTag,
		&p.Score, &p.Tier, &breakdown, &p.YouTubeID, &p.Status, &p.Featured, &p.SyncedAt, &p.SyncError, &p.CreatedAt, &p.UpdatedAt,
		&p.Tagline, &p.Description, &p.Machine, &p.Translated)
	if err != nil {
		return nil, err
	}
	p.FullName = p.Owner + "/" + p.Repo
	p.Breakdown = breakdown
	if p.Topics == nil {
		p.Topics = []string{}
	}
	if p.ExtraTopics == nil {
		p.ExtraTopics = []string{}
	}
	return &p, nil
}

func (s *Store) loc(l string) string {
	if l == "" {
		return s.DefaultLocale
	}
	return l
}

func (s *Store) GetProject(ctx context.Context, id int64, locale string) (*Project, error) {
	p, err := scanProject(s.DB.QueryRow(ctx, projectSelect+` WHERE p.id = $3`, s.loc(locale), s.DefaultLocale, id))
	if err != nil {
		return nil, err
	}
	p.Screenshots, err = s.Screenshots(ctx, p.ID)
	return p, err
}

func (s *Store) GetProjectBySlug(ctx context.Context, slug, locale string, publishedOnly bool) (*Project, error) {
	q := projectSelect + ` WHERE p.slug = $3`
	if publishedOnly {
		q += ` AND p.status = 'published'`
	}
	p, err := scanProject(s.DB.QueryRow(ctx, q, s.loc(locale), s.DefaultLocale, slug))
	if err != nil {
		return nil, err
	}
	p.Screenshots, err = s.Screenshots(ctx, p.ID)
	return p, err
}

func (s *Store) ListProjects(ctx context.Context, f ProjectFilter) ([]Project, int, error) {
	args := []any{s.loc(f.Locale), s.DefaultLocale}
	where := []string{"$1::text IS NOT NULL", "$2::text IS NOT NULL"} // keep $1/$2 referenced in the count query
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if f.Status != "" {
		add("p.status = ?", f.Status)
	}
	if f.CategoryID != 0 {
		add("p.category_id = ?", f.CategoryID)
	}
	if f.MinScore > 0 {
		add("p.activity_score >= ?", f.MinScore)
	}
	if f.Featured {
		where = append(where, "p.featured")
	}
	if f.Topic != "" {
		add("? = ANY(p.topics || p.extra_topics)", f.Topic)
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		add("(p.slug ILIKE ? OR p.github_owner ILIKE ?)", "%"+escapeLike(q)+"%")
	}
	cond := " WHERE " + strings.Join(where, " AND ")
	var total int
	if err := s.DB.QueryRow(ctx, "SELECT count(*) FROM projects p"+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := "p.activity_score DESC NULLS LAST, p.stars DESC, p.id"
	switch f.Sort {
	case "stars":
		order = "p.stars DESC, p.id"
	case "new":
		order = "p.created_at DESC, p.id DESC"
	case "name":
		order = "p.slug"
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 24
	}
	rows, err := s.DB.Query(ctx, projectSelect+cond+" ORDER BY "+order+fmt.Sprintf(" LIMIT %d OFFSET %d", limit, max(0, f.Offset)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *p)
	}
	return out, total, rows.Err()
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// CreateProject inserts a draft; the first sync fills in the metadata.
func (s *Store) CreateProject(ctx context.Context, owner, repo string, categoryID *int64) (int64, error) {
	slug := Slugify(owner + "-" + repo)
	var id int64
	err := s.DB.QueryRow(ctx, `INSERT INTO projects (slug, github_owner, github_repo, category_id) VALUES ($1,$2,$3,$4)
		ON CONFLICT (github_owner, github_repo) DO NOTHING RETURNING id`, slug, owner, repo, categoryID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrExists
	}
	return id, err
}

var ErrExists = errors.New("already exists")

type ProjectEdit struct {
	CategoryID *int64
	YouTubeID  string
	Status     string
	Featured   bool
	Homepage   string
}

func (s *Store) UpdateProject(ctx context.Context, id int64, e ProjectEdit) error {
	_, err := s.DB.Exec(ctx, `UPDATE projects SET category_id=$2, youtube_id=$3, status=$4, featured=$5, homepage=$6, updated_at=now(),
		published_at = CASE WHEN $4 = 'published' THEN COALESCE(published_at, now()) ELSE published_at END WHERE id=$1`,
		id, e.CategoryID, e.YouTubeID, e.Status, e.Featured, e.Homepage)
	return err
}

func (s *Store) DeleteProject(ctx context.Context, id int64) ([]string, error) {
	shots, _ := s.Screenshots(ctx, id)
	if _, err := s.DB.Exec(ctx, `DELETE FROM projects WHERE id=$1`, id); err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(shots))
	for _, sh := range shots {
		paths = append(paths, sh.Path)
	}
	return paths, nil
}

// ApplyGitHub stores refreshed metadata and the computed score.
func (s *Store) ApplyGitHub(ctx context.Context, id int64, r *github.Repo, res scoring.Result) error {
	breakdown, _ := json.Marshal(map[string]any{"result": res, "signals": r.Signals, "graphql": r.GraphQL})
	var relAt *time.Time
	relTag := ""
	if r.LastRelease != nil {
		relAt, relTag = &r.LastRelease.PublishedAt, r.LastRelease.Tag
	}
	last := r.Signals.LastCommitAt
	var lastPtr, pushedPtr, createdPtr *time.Time
	if !last.IsZero() {
		lastPtr = &last
	}
	if !r.PushedAt.IsZero() {
		pushedPtr = &r.PushedAt
	}
	if !r.CreatedAt.IsZero() {
		createdPtr = &r.CreatedAt
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE projects SET
		homepage = CASE WHEN homepage = '' THEN $2 ELSE homepage END,
		license=$3, language=$4, topics=$5, avatar_url=$6, stars=$7, forks=$8, open_issues=$9, archived=$10, default_branch=$11,
		gh_created_at=$12, pushed_at=$13, last_commit_at=$14, last_release_at=$15, last_release_tag=$16,
		activity_score=$17, activity_tier=$18, score_breakdown=$19, signals=$20, synced_at=now(), sync_error='', updated_at=now()
		WHERE id=$1`,
		id, r.Homepage, r.License, r.Language, nonNil(r.Topics), r.AvatarURL, r.Stars, r.Forks, r.OpenIssues, r.Archived, r.DefaultBranch,
		createdPtr, pushedPtr, lastPtr, relAt, relTag, res.Score, res.Tier, breakdown, breakdown)
	if err != nil {
		return err
	}
	if r.Description != "" { // seed the default-locale text once; never overwrite edits
		_, err = tx.Exec(ctx, `INSERT INTO project_translations (project_id, locale, tagline, source) VALUES ($1,$2,$3,'human')
			ON CONFLICT (project_id, locale) DO UPDATE SET tagline = EXCLUDED.tagline WHERE project_translations.tagline = ''`, id, s.DefaultLocale, r.Description)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (s *Store) MarkSyncError(ctx context.Context, id int64, msg string) error {
	_, err := s.DB.Exec(ctx, `UPDATE projects SET sync_error=$2, synced_at=now() WHERE id=$1`, id, msg)
	return err
}

type SyncTarget struct {
	ID          int64
	Owner, Repo string
}

// StaleProjects returns never-synced projects first, then the oldest syncs.
func (s *Store) StaleProjects(ctx context.Context, olderThan time.Duration, limit int) ([]SyncTarget, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, github_owner, github_repo FROM projects
		WHERE synced_at IS NULL OR synced_at < now() - make_interval(secs => $1)
		ORDER BY synced_at NULLS FIRST LIMIT $2`, olderThan.Seconds(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SyncTarget
	for rows.Next() {
		var t SyncTarget
		if err := rows.Scan(&t.ID, &t.Owner, &t.Repo); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) ProjectTarget(ctx context.Context, id int64) (SyncTarget, error) {
	t := SyncTarget{ID: id}
	err := s.DB.QueryRow(ctx, `SELECT github_owner, github_repo FROM projects WHERE id=$1`, id).Scan(&t.Owner, &t.Repo)
	return t, err
}

// ---- translations ----

func (s *Store) Translations(ctx context.Context, projectID int64) (map[string]Translation, error) {
	rows, err := s.DB.Query(ctx, `SELECT locale, tagline, description, source FROM project_translations WHERE project_id=$1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Translation{}
	for rows.Next() {
		var t Translation
		if err := rows.Scan(&t.Locale, &t.Tagline, &t.Description, &t.Source); err != nil {
			return nil, err
		}
		out[t.Locale] = t
	}
	return out, rows.Err()
}

func (s *Store) SaveTranslation(ctx context.Context, projectID int64, t Translation) error {
	if t.Source == "" {
		t.Source = "human"
	}
	_, err := s.DB.Exec(ctx, `INSERT INTO project_translations (project_id, locale, tagline, description, source) VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (project_id, locale) DO UPDATE SET tagline=EXCLUDED.tagline, description=EXCLUDED.description, source=EXCLUDED.source, updated_at=now()`,
		projectID, t.Locale, t.Tagline, t.Description, t.Source)
	return err
}

func (s *Store) DeleteTranslation(ctx context.Context, projectID int64, locale string) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM project_translations WHERE project_id=$1 AND locale=$2`, projectID, locale)
	return err
}

// ---- screenshots ----

func (s *Store) Screenshots(ctx context.Context, projectID int64) ([]Screenshot, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, project_id, path, alt, position FROM project_screenshots WHERE project_id=$1 ORDER BY position, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Screenshot{}
	for rows.Next() {
		var sh Screenshot
		if err := rows.Scan(&sh.ID, &sh.ProjectID, &sh.Path, &sh.Alt, &sh.Position); err != nil {
			return nil, err
		}
		out = append(out, sh)
	}
	return out, rows.Err()
}

func (s *Store) AddScreenshot(ctx context.Context, projectID int64, path, alt string) (int64, error) {
	var id int64
	err := s.DB.QueryRow(ctx, `INSERT INTO project_screenshots (project_id, path, alt, position)
		VALUES ($1,$2,$3,(SELECT COALESCE(max(position),0)+1 FROM project_screenshots WHERE project_id=$1)) RETURNING id`, projectID, path, alt).Scan(&id)
	return id, err
}

func (s *Store) UpdateScreenshot(ctx context.Context, id, projectID int64, alt string, position int) error {
	_, err := s.DB.Exec(ctx, `UPDATE project_screenshots SET alt=$3, position=$4 WHERE id=$1 AND project_id=$2`, id, projectID, alt, position)
	return err
}

// DeleteScreenshot removes the row and returns its stored path.
func (s *Store) DeleteScreenshot(ctx context.Context, id, projectID int64) (string, error) {
	var path string
	err := s.DB.QueryRow(ctx, `DELETE FROM project_screenshots WHERE id=$1 AND project_id=$2 RETURNING path`, id, projectID).Scan(&path)
	return path, err
}

// ---- embeddings / related ----

func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

func (s *Store) SetEmbedding(ctx context.Context, id int64, v []float32) error {
	_, err := s.DB.Exec(ctx, `UPDATE projects SET embedding = $2::vector WHERE id=$1`, id, vectorLiteral(v))
	return err
}

// EmbeddingTargets lists published projects without an embedding.
func (s *Store) EmbeddingTargets(ctx context.Context, limit int) ([]int64, error) {
	rows, err := s.DB.Query(ctx, `SELECT id FROM projects WHERE embedding IS NULL AND status='published' ORDER BY id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// Related returns nearest neighbours by vector similarity (pgvector) and, when
// no embedding exists yet, the best-scored projects of the same category.
func (s *Store) Related(ctx context.Context, p *Project, locale string, n int) ([]Project, error) {
	var hasEmb bool
	_ = s.DB.QueryRow(ctx, `SELECT embedding IS NOT NULL FROM projects WHERE id=$1`, p.ID).Scan(&hasEmb)
	var rows pgx.Rows
	var err error
	if hasEmb {
		rows, err = s.DB.Query(ctx, projectSelect+` WHERE p.status='published' AND p.id <> $3 AND p.embedding IS NOT NULL
			ORDER BY p.embedding <=> (SELECT embedding FROM projects WHERE id = $3) LIMIT $4`, s.loc(locale), s.DefaultLocale, p.ID, n)
		if err != nil { // e.g. mixed vector dimensions: fall back to category
			hasEmb = false
		}
	}
	if !hasEmb {
		if p.CategoryID == nil {
			return nil, nil
		}
		rows, err = s.DB.Query(ctx, projectSelect+` WHERE p.status='published' AND p.id <> $3 AND p.category_id = $4
			ORDER BY p.activity_score DESC NULLS LAST, p.stars DESC LIMIT $5`, s.loc(locale), s.DefaultLocale, p.ID, *p.CategoryID, n)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		r, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// ---- stats ----

type Stats struct {
	Projects, Published, Categories, SyncErrors, NeverSynced int
}

func (s *Store) Stats(ctx context.Context) (st Stats, err error) {
	err = s.DB.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM projects),
		(SELECT count(*) FROM projects WHERE status='published'),
		(SELECT count(*) FROM categories),
		(SELECT count(*) FROM projects WHERE sync_error <> ''),
		(SELECT count(*) FROM projects WHERE synced_at IS NULL)`).Scan(&st.Projects, &st.Published, &st.Categories, &st.SyncErrors, &st.NeverSynced)
	return
}

// ---- admin users ----

func (s *Store) AdminByEmail(ctx context.Context, email string) (*AdminUser, error) {
	var u AdminUser
	err := s.DB.QueryRow(ctx, `SELECT id, email, password_hash FROM admin_users WHERE lower(email)=lower($1)`, email).Scan(&u.ID, &u.Email, &u.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) AdminByID(ctx context.Context, id int64) (*AdminUser, error) {
	var u AdminUser
	err := s.DB.QueryRow(ctx, `SELECT id, email, password_hash FROM admin_users WHERE id=$1`, id).Scan(&u.ID, &u.Email, &u.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UpsertAdmin(ctx context.Context, email, hash string) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO admin_users (email, password_hash) VALUES ($1,$2)
		ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash`, strings.ToLower(email), hash)
	return err
}

func (s *Store) CountAdmins(ctx context.Context) (n int, err error) {
	err = s.DB.QueryRow(ctx, `SELECT count(*) FROM admin_users`).Scan(&n)
	return
}

// ProjectLocales maps published project ids to the non-default locales that have real translated text.
func (s *Store) ProjectLocales(ctx context.Context) (map[int64][]string, error) {
	rows, err := s.DB.Query(ctx, `SELECT t.project_id, t.locale FROM project_translations t JOIN projects p ON p.id = t.project_id
		WHERE p.status='published' AND t.locale <> $1 AND (t.tagline <> '' OR t.description <> '') ORDER BY t.project_id, t.locale`, s.DefaultLocale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]string{}
	for rows.Next() {
		var id int64
		var loc string
		if err := rows.Scan(&id, &loc); err != nil {
			return nil, err
		}
		out[id] = append(out[id], loc)
	}
	return out, rows.Err()
}

// ProjectTranslatedLocales lists the non-default locales with text for one project.
func (s *Store) ProjectTranslatedLocales(ctx context.Context, id int64) ([]string, error) {
	rows, err := s.DB.Query(ctx, `SELECT locale FROM project_translations WHERE project_id=$1 AND locale <> $2 AND (tagline <> '' OR description <> '') ORDER BY locale`, id, s.DefaultLocale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var l string
		if err := rows.Scan(&l); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// SitemapProject is the minimum needed to describe a project in a sitemap.
type SitemapProject struct {
	ID          int64
	Slug        string
	FullName    string
	Tagline     string
	YouTubeID   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Screenshots []string
}

func (s *Store) CountPublished(ctx context.Context) (n int, err error) {
	err = s.DB.QueryRow(ctx, `SELECT count(*) FROM projects WHERE status='published'`).Scan(&n)
	return
}

// SitemapPageLastMod returns the newest update inside one page of the publication-ordered list.
func (s *Store) SitemapPageLastMod(ctx context.Context, offset, limit int) (t time.Time, err error) {
	err = s.DB.QueryRow(ctx, `SELECT COALESCE(max(u), now()) FROM (SELECT updated_at AS u FROM projects WHERE status='published'
		ORDER BY published_at, id OFFSET $1 LIMIT $2) x`, offset, limit).Scan(&t)
	return
}

// SitemapProjects returns published projects ordered by publication date ascending (new ones append at the end).
func (s *Store) SitemapProjects(ctx context.Context, offset, limit int) ([]SitemapProject, error) {
	rows, err := s.DB.Query(ctx, `SELECT p.id, p.slug, p.github_owner || '/' || p.github_repo, COALESCE(d.tagline,''), p.youtube_id, p.created_at, p.updated_at
		FROM projects p LEFT JOIN project_translations d ON d.project_id = p.id AND d.locale = $3
		WHERE p.status='published' ORDER BY p.published_at, p.id OFFSET $1 LIMIT $2`, offset, limit, s.DefaultLocale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SitemapProject
	idx := map[int64]int{}
	var ids []int64
	for rows.Next() {
		var p SitemapProject
		if err := rows.Scan(&p.ID, &p.Slug, &p.FullName, &p.Tagline, &p.YouTubeID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		idx[p.ID] = len(out)
		ids = append(ids, p.ID)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil || len(ids) == 0 {
		return out, err
	}
	shots, err := s.DB.Query(ctx, `SELECT project_id, path FROM project_screenshots WHERE project_id = ANY($1) ORDER BY position, id`, ids)
	if err != nil {
		return nil, err
	}
	defer shots.Close()
	for shots.Next() {
		var id int64
		var path string
		if err := shots.Scan(&id, &path); err != nil {
			return nil, err
		}
		out[idx[id]].Screenshots = append(out[idx[id]].Screenshots, path)
	}
	return out, shots.Err()
}
