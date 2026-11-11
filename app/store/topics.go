package store

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Topic struct {
	ID           int64  `json:"id"`
	Slug         string `json:"slug"`
	SortOrder    int    `json:"sortOrder"`
	Published    bool   `json:"published"`
	Name         string `json:"name"`
	Summary      string `json:"summary"`
	Body         string `json:"body"`
	Translated   bool   `json:"translated"`
	ProjectCount int    `json:"projectCount"`
}

type TopicTranslation struct {
	Locale  string `json:"locale"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Body    string `json:"body"`
}

type FeatureText struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Feature is one comparison row. Name/Description are resolved for the requested locale;
// Translations holds every locale (admin editing).
type Feature struct {
	ID           int64                  `json:"id"`
	Position     int                    `json:"position"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Translations map[string]FeatureText `json:"translations"`
}

// FeatureValue is a project's answer for one feature: a state plus an optional note per locale.
type FeatureValue struct {
	State string            `json:"state"` // yes | no | partial | text
	Notes map[string]string `json:"notes"`
}

var FeatureStates = map[string]bool{"yes": true, "no": true, "partial": true, "text": true}

const topicSelect = `
SELECT t.id, t.slug, t.sort_order, t.published, COALESCE(NULLIF(tt.name,''), NULLIF(td.name,''), t.slug),
  COALESCE(NULLIF(tt.summary,''), td.summary, ''), COALESCE(NULLIF(tt.body,''), td.body, ''),
  COALESCE(tt.name <> '', false),
  (SELECT count(*) FROM projects p WHERE p.status = 'published' AND t.slug = ANY(p.topics || p.extra_topics))
FROM topics t
LEFT JOIN topic_translations tt ON tt.topic_id = t.id AND tt.locale = $1
LEFT JOIN topic_translations td ON td.topic_id = t.id AND td.locale = $2`

func scanTopic(row pgx.Row) (*Topic, error) {
	var t Topic
	err := row.Scan(&t.ID, &t.Slug, &t.SortOrder, &t.Published, &t.Name, &t.Summary, &t.Body, &t.Translated, &t.ProjectCount)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) ListTopics(ctx context.Context, locale string, publishedOnly bool) ([]Topic, error) {
	q := topicSelect
	if publishedOnly {
		q += ` WHERE t.published`
	}
	rows, err := s.DB.Query(ctx, q+` ORDER BY t.sort_order, 5`, s.loc(locale), s.DefaultLocale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Topic{}
	for rows.Next() {
		t, err := scanTopic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (s *Store) TopicBySlug(ctx context.Context, slug, locale string, publishedOnly bool) (*Topic, error) {
	q := topicSelect + ` WHERE t.slug = $3`
	if publishedOnly {
		q += ` AND t.published`
	}
	return scanTopic(s.DB.QueryRow(ctx, q, s.loc(locale), s.DefaultLocale, slug))
}

func (s *Store) TopicByID(ctx context.Context, id int64, locale string) (*Topic, error) {
	return scanTopic(s.DB.QueryRow(ctx, topicSelect+` WHERE t.id = $3`, s.loc(locale), s.DefaultLocale, id))
}

func (s *Store) TopicTranslations(ctx context.Context, id int64) (map[string]TopicTranslation, error) {
	rows, err := s.DB.Query(ctx, `SELECT locale, name, summary, body FROM topic_translations WHERE topic_id=$1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]TopicTranslation{}
	for rows.Next() {
		var t TopicTranslation
		if err := rows.Scan(&t.Locale, &t.Name, &t.Summary, &t.Body); err != nil {
			return nil, err
		}
		out[t.Locale] = t
	}
	return out, rows.Err()
}

// SaveTopic creates (ID==0) or updates a topic and replaces its translations.
func (s *Store) SaveTopic(ctx context.Context, t Topic, trs []TopicTranslation) (int64, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	t.Slug = Slugify(t.Slug)
	if t.ID == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO topics (slug, sort_order, published) VALUES ($1,$2,$3) RETURNING id`, t.Slug, t.SortOrder, t.Published).Scan(&t.ID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE topics SET slug=$2, sort_order=$3, published=$4, updated_at=now() WHERE id=$1`, t.ID, t.Slug, t.SortOrder, t.Published)
	}
	if err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM topic_translations WHERE topic_id=$1`, t.ID); err != nil {
		return 0, err
	}
	for _, tr := range trs {
		if strings.TrimSpace(tr.Name) == "" && strings.TrimSpace(tr.Summary) == "" && strings.TrimSpace(tr.Body) == "" {
			continue
		}
		if _, err = tx.Exec(ctx, `INSERT INTO topic_translations (topic_id, locale, name, summary, body) VALUES ($1,$2,$3,$4,$5)`, t.ID, tr.Locale, tr.Name, tr.Summary, tr.Body); err != nil {
			return 0, err
		}
	}
	return t.ID, tx.Commit(ctx)
}

func (s *Store) DeleteTopic(ctx context.Context, id int64) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM topics WHERE id=$1`, id)
	return err
}

// TopicLocales maps topic ids to the non-default locales that have a translated name.
func (s *Store) TopicLocales(ctx context.Context) (map[int64][]string, error) {
	rows, err := s.DB.Query(ctx, `SELECT topic_id, locale FROM topic_translations WHERE locale <> $1 AND name <> '' ORDER BY topic_id, locale`, s.DefaultLocale)
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

// TopicsForSlugs returns the published topics whose slug is in the list (used for "compared in" links on project pages).
func (s *Store) TopicsForSlugs(ctx context.Context, slugs []string, locale string) ([]Topic, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	rows, err := s.DB.Query(ctx, topicSelect+` WHERE t.published AND t.slug = ANY($3) ORDER BY t.sort_order, 5`, s.loc(locale), s.DefaultLocale, slugs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Topic
	for rows.Next() {
		t, err := scanTopic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// Features lists a topic's comparison rows in order; Name/Description are resolved for locale.
func (s *Store) Features(ctx context.Context, topicID int64, locale string) ([]Feature, error) {
	rows, err := s.DB.Query(ctx, `SELECT f.id, f.position, ft.locale, ft.name, ft.description FROM topic_features f
		LEFT JOIN topic_feature_translations ft ON ft.feature_id = f.id WHERE f.topic_id=$1 ORDER BY f.position, f.id`, topicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	loc := s.loc(locale)
	var out []Feature
	idx := map[int64]int{}
	for rows.Next() {
		var id int64
		var pos int
		var l, name, desc *string
		if err := rows.Scan(&id, &pos, &l, &name, &desc); err != nil {
			return nil, err
		}
		i, ok := idx[id]
		if !ok {
			i = len(out)
			idx[id] = i
			out = append(out, Feature{ID: id, Position: pos, Translations: map[string]FeatureText{}})
		}
		if l != nil {
			out[i].Translations[*l] = FeatureText{Name: *name, Description: *desc}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		tr, def := out[i].Translations[loc], out[i].Translations[s.DefaultLocale]
		out[i].Name, out[i].Description = firstNonEmpty(tr.Name, def.Name), firstNonEmpty(tr.Description, def.Description)
	}
	return out, nil
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// SaveFeatures syncs a topic's rows to the given list: entries with ID 0 are created, known ids updated, missing ones deleted
// (their values go with them). Order follows the slice.
func (s *Store) SaveFeatures(ctx context.Context, topicID int64, list []Feature) ([]Feature, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	keep := []int64{}
	for i := range list {
		f := &list[i]
		f.Position = i
		if f.ID == 0 {
			err = tx.QueryRow(ctx, `INSERT INTO topic_features (topic_id, position) VALUES ($1,$2) RETURNING id`, topicID, i).Scan(&f.ID)
		} else {
			var n int64
			err = tx.QueryRow(ctx, `UPDATE topic_features SET position=$3 WHERE id=$1 AND topic_id=$2 RETURNING id`, f.ID, topicID, i).Scan(&n)
		}
		if err != nil {
			return nil, err
		}
		keep = append(keep, f.ID)
		if _, err = tx.Exec(ctx, `DELETE FROM topic_feature_translations WHERE feature_id=$1`, f.ID); err != nil {
			return nil, err
		}
		for loc, tr := range f.Translations {
			if strings.TrimSpace(tr.Name) == "" && strings.TrimSpace(tr.Description) == "" {
				continue
			}
			if _, err = tx.Exec(ctx, `INSERT INTO topic_feature_translations (feature_id, locale, name, description) VALUES ($1,$2,$3,$4)`, f.ID, loc, strings.TrimSpace(tr.Name), strings.TrimSpace(tr.Description)); err != nil {
				return nil, err
			}
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM topic_features WHERE topic_id=$1 AND NOT (id = ANY($2))`, topicID, keep); err != nil {
		return nil, err
	}
	return list, tx.Commit(ctx)
}

// FeatureValues returns projectID -> featureID -> value for a topic.
func (s *Store) FeatureValues(ctx context.Context, topicID int64) (map[int64]map[int64]FeatureValue, error) {
	rows, err := s.DB.Query(ctx, `SELECT v.project_id, v.feature_id, v.state, v.notes FROM project_feature_values v
		JOIN topic_features f ON f.id = v.feature_id WHERE f.topic_id=$1`, topicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]map[int64]FeatureValue{}
	for rows.Next() {
		var pid, fid int64
		var v FeatureValue
		var notes []byte
		if err := rows.Scan(&pid, &fid, &v.State, &notes); err != nil {
			return nil, err
		}
		v.Notes = map[string]string{}
		_ = json.Unmarshal(notes, &v.Notes)
		if out[pid] == nil {
			out[pid] = map[int64]FeatureValue{}
		}
		out[pid][fid] = v
	}
	return out, rows.Err()
}

// SetFeatureValue upserts a value; an empty state clears it. The feature must belong to the topic.
func (s *Store) SetFeatureValue(ctx context.Context, topicID, projectID, featureID int64, state string, notes map[string]string) error {
	var ok bool
	if err := s.DB.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM topic_features WHERE id=$1 AND topic_id=$2)`, featureID, topicID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	clean := map[string]string{}
	for l, n := range notes {
		if n = strings.TrimSpace(n); n != "" {
			clean[l] = n
		}
	}
	if state == "" && len(clean) == 0 {
		_, err := s.DB.Exec(ctx, `DELETE FROM project_feature_values WHERE project_id=$1 AND feature_id=$2`, projectID, featureID)
		return err
	}
	if state == "" {
		state = "text"
	}
	raw, _ := json.Marshal(clean)
	_, err := s.DB.Exec(ctx, `INSERT INTO project_feature_values (project_id, feature_id, state, notes) VALUES ($1,$2,$3,$4)
		ON CONFLICT (project_id, feature_id) DO UPDATE SET state=EXCLUDED.state, notes=EXCLUDED.notes, updated_at=now()`, projectID, featureID, state, raw)
	return err
}

// SetExtraTopics replaces the curated topics of a project. They survive GitHub resyncs.
func (s *Store) SetExtraTopics(ctx context.Context, projectID int64, slugs []string) error {
	seen := map[string]bool{}
	clean := []string{}
	for _, t := range slugs {
		if t = Slugify(t); t != "" && !seen[t] {
			seen[t] = true
			clean = append(clean, t)
		}
	}
	_, err := s.DB.Exec(ctx, `UPDATE projects SET extra_topics=$2, updated_at=now() WHERE id=$1`, projectID, clean)
	return err
}

func (s *Store) AddExtraTopic(ctx context.Context, projectID int64, slug string) error {
	_, err := s.DB.Exec(ctx, `UPDATE projects SET extra_topics = CASE WHEN $2 = ANY(extra_topics) THEN extra_topics ELSE array_append(extra_topics, $2) END, updated_at=now() WHERE id=$1`, projectID, Slugify(slug))
	return err
}

func (s *Store) RemoveExtraTopic(ctx context.Context, projectID int64, slug string) error {
	_, err := s.DB.Exec(ctx, `UPDATE projects SET extra_topics = array_remove(extra_topics, $2), updated_at=now() WHERE id=$1`, projectID, Slugify(slug))
	return err
}

// ---- MCP access tokens ----

type MCPToken struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Token      string     `json:"token"`
	CreatedAt  time.Time  `json:"createdAt"`
	RotatedAt  *time.Time `json:"rotatedAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	RevokedAt  *time.Time `json:"revokedAt"`
}

func (s *Store) CreateMCPToken(ctx context.Context, name, token string) (*MCPToken, error) {
	var t MCPToken
	err := s.DB.QueryRow(ctx, `INSERT INTO mcp_tokens (name, token) VALUES ($1,$2) RETURNING id, name, token, created_at, rotated_at, last_used_at, revoked_at`, name, token).
		Scan(&t.ID, &t.Name, &t.Token, &t.CreatedAt, &t.RotatedAt, &t.LastUsedAt, &t.RevokedAt)
	return &t, err
}

func (s *Store) ListMCPTokens(ctx context.Context) ([]MCPToken, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, name, token, created_at, rotated_at, last_used_at, revoked_at FROM mcp_tokens ORDER BY revoked_at IS NOT NULL, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MCPToken{}
	for rows.Next() {
		var t MCPToken
		if err := rows.Scan(&t.ID, &t.Name, &t.Token, &t.CreatedAt, &t.RotatedAt, &t.LastUsedAt, &t.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) RenameMCPToken(ctx context.Context, id int64, name string) error {
	_, err := s.DB.Exec(ctx, `UPDATE mcp_tokens SET name=$2 WHERE id=$1`, id, name)
	return err
}

// RotateMCPToken swaps in a new secret (and re-activates a revoked token); the old link stops working immediately.
func (s *Store) RotateMCPToken(ctx context.Context, id int64, token string) error {
	_, err := s.DB.Exec(ctx, `UPDATE mcp_tokens SET token=$2, rotated_at=now(), revoked_at=NULL WHERE id=$1`, id, token)
	return err
}

func (s *Store) RevokeMCPToken(ctx context.Context, id int64) error {
	_, err := s.DB.Exec(ctx, `UPDATE mcp_tokens SET revoked_at=COALESCE(revoked_at, now()) WHERE id=$1`, id)
	return err
}

func (s *Store) DeleteMCPToken(ctx context.Context, id int64) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM mcp_tokens WHERE id=$1`, id)
	return err
}

// MCPTokenValid reports whether the secret belongs to an active token and records its use.
func (s *Store) MCPTokenValid(ctx context.Context, token string) (bool, error) {
	var id int64
	err := s.DB.QueryRow(ctx, `UPDATE mcp_tokens SET last_used_at = CASE WHEN last_used_at IS NULL OR last_used_at < now() - interval '1 minute' THEN now() ELSE last_used_at END
		WHERE token=$1 AND revoked_at IS NULL RETURNING id`, token).Scan(&id)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
