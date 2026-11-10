package store

import (
	"context"
	"strings"
)

const categorySelect = `
SELECT c.id, c.slug, c.icon, c.sort_order, COALESCE(NULLIF(ct.name,''), cd.name, c.slug), COALESCE(NULLIF(ct.description,''), cd.description, ''),
  (SELECT count(*) FROM projects p WHERE p.category_id = c.id AND p.status = 'published'),
  COALESCE(ct.name <> '', false)
FROM categories c
LEFT JOIN category_translations ct ON ct.category_id = c.id AND ct.locale = $1
LEFT JOIN category_translations cd ON cd.category_id = c.id AND cd.locale = $2`

func (s *Store) ListCategories(ctx context.Context, locale string) ([]Category, error) {
	rows, err := s.DB.Query(ctx, categorySelect+` ORDER BY c.sort_order, 5`, s.loc(locale), s.DefaultLocale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Slug, &c.Icon, &c.SortOrder, &c.Name, &c.Description, &c.ProjectCount, &c.Translated); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CategoryBySlug(ctx context.Context, slug, locale string) (*Category, error) {
	var c Category
	err := s.DB.QueryRow(ctx, categorySelect+` WHERE c.slug = $3`, s.loc(locale), s.DefaultLocale, slug).
		Scan(&c.ID, &c.Slug, &c.Icon, &c.SortOrder, &c.Name, &c.Description, &c.ProjectCount, &c.Translated)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) CategoryByID(ctx context.Context, id int64, locale string) (*Category, error) {
	var c Category
	err := s.DB.QueryRow(ctx, categorySelect+` WHERE c.id = $3`, s.loc(locale), s.DefaultLocale, id).
		Scan(&c.ID, &c.Slug, &c.Icon, &c.SortOrder, &c.Name, &c.Description, &c.ProjectCount, &c.Translated)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) CategoryTranslations(ctx context.Context, id int64) (map[string]CategoryTranslation, error) {
	rows, err := s.DB.Query(ctx, `SELECT locale, name, description FROM category_translations WHERE category_id=$1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]CategoryTranslation{}
	for rows.Next() {
		var t CategoryTranslation
		if err := rows.Scan(&t.Locale, &t.Name, &t.Description); err != nil {
			return nil, err
		}
		out[t.Locale] = t
	}
	return out, rows.Err()
}

// SaveCategory creates (ID==0) or updates a category and replaces its translations.
func (s *Store) SaveCategory(ctx context.Context, c Category, trs []CategoryTranslation) (int64, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	c.Slug = Slugify(c.Slug)
	if c.ID == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO categories (slug, icon, sort_order) VALUES ($1,$2,$3) RETURNING id`, c.Slug, c.Icon, c.SortOrder).Scan(&c.ID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE categories SET slug=$2, icon=$3, sort_order=$4, updated_at=now() WHERE id=$1`, c.ID, c.Slug, c.Icon, c.SortOrder)
	}
	if err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM category_translations WHERE category_id=$1`, c.ID); err != nil {
		return 0, err
	}
	for _, t := range trs {
		if strings.TrimSpace(t.Name) == "" {
			continue
		}
		if _, err = tx.Exec(ctx, `INSERT INTO category_translations (category_id, locale, name, description) VALUES ($1,$2,$3,$4)`, c.ID, t.Locale, t.Name, t.Description); err != nil {
			return 0, err
		}
	}
	return c.ID, tx.Commit(ctx)
}

func (s *Store) DeleteCategory(ctx context.Context, id int64) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM categories WHERE id=$1`, id)
	return err
}

// CategoryLocales maps category ids to the non-default locales that have a translated name.
func (s *Store) CategoryLocales(ctx context.Context) (map[int64][]string, error) {
	rows, err := s.DB.Query(ctx, `SELECT category_id, locale FROM category_translations WHERE locale <> $1 AND name <> '' ORDER BY category_id, locale`, s.DefaultLocale)
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
