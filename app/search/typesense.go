// Package search is a small Typesense client plus the indexer that keeps one
// collection per locale in sync with PostgreSQL.
package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("search: typesense unavailable")

type Client struct {
	BaseURL string
	APIKey  string
	Prefix  string
	HTTP    *http.Client
}

func NewClient(baseURL, apiKey, prefix string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, Prefix: prefix, HTTP: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) Enabled() bool { return c != nil && c.BaseURL != "" && c.APIKey != "" }

func (c *Client) Collection(locale string) string { return c.Prefix + "_projects_" + locale }

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.APIKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	return b, resp.StatusCode, nil
}

func (c *Client) json(ctx context.Context, method, path string, payload any) ([]byte, int, error) {
	var r io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		r = bytes.NewReader(b)
	}
	return c.do(ctx, method, path, r, "application/json")
}

// Doc is the searchable representation of a published project in one locale.
type Doc struct {
	ID           string   `json:"id"`
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	FullName     string   `json:"full_name"`
	Tagline      string   `json:"tagline"`
	Description  string   `json:"description"`
	Topics       []string `json:"topics"`
	Language     string   `json:"language"`
	Category     string   `json:"category"`
	CategorySlug string   `json:"category_slug"`
	Score        float64  `json:"score"`
	Tier         int32    `json:"tier"`
	Stars        int32    `json:"stars"`
	Featured     bool     `json:"featured"`
	HasVideo     bool     `json:"has_video"`
	Avatar       string   `json:"avatar"`
	UpdatedAt    int64    `json:"updated_at"`
}

func (c *Client) schema(locale string) map[string]any {
	// Typesense has no language-specific analyzers for every locale; setting `locale`
	// on text fields enables correct tokenisation for ja/zh and similar.
	tl := ""
	if locale == "ja" || locale == "zh" {
		tl = locale
	}
	str := func(name string, extra map[string]any) map[string]any {
		f := map[string]any{"name": name, "type": "string"}
		if tl != "" {
			f["locale"] = tl
		}
		for k, v := range extra {
			f[k] = v
		}
		return f
	}
	return map[string]any{
		"name": c.Collection(locale),
		"fields": []map[string]any{
			str("slug", map[string]any{"index": false, "optional": true}),
			str("name", nil), str("full_name", nil), str("tagline", map[string]any{"optional": true}), str("description", map[string]any{"optional": true}),
			{"name": "topics", "type": "string[]", "facet": true, "optional": true},
			str("language", map[string]any{"facet": true, "optional": true}),
			str("category", map[string]any{"optional": true}),
			{"name": "category_slug", "type": "string", "facet": true, "optional": true},
			{"name": "score", "type": "float"}, {"name": "tier", "type": "int32", "facet": true},
			{"name": "stars", "type": "int32"}, {"name": "featured", "type": "bool"}, {"name": "has_video", "type": "bool"},
			{"name": "avatar", "type": "string", "index": false, "optional": true},
			{"name": "updated_at", "type": "int64"},
		},
		"default_sorting_field": "stars",
	}
}

// EnsureCollection creates the collection when missing.
func (c *Client) EnsureCollection(ctx context.Context, locale string) error {
	_, status, err := c.json(ctx, http.MethodGet, "/collections/"+c.Collection(locale), nil)
	if err != nil {
		return err
	}
	if status == http.StatusOK {
		return nil
	}
	body, status, err := c.json(ctx, http.MethodPost, "/collections", c.schema(locale))
	if err != nil {
		return err
	}
	if status >= 300 && status != http.StatusConflict {
		return fmt.Errorf("typesense create collection: %d %s", status, body)
	}
	return nil
}

// RecreateCollection drops and recreates the collection (used by full reindex).
func (c *Client) RecreateCollection(ctx context.Context, locale string) error {
	if _, _, err := c.json(ctx, http.MethodDelete, "/collections/"+c.Collection(locale), nil); err != nil {
		return err
	}
	return c.EnsureCollection(ctx, locale)
}

func (c *Client) Import(ctx context.Context, locale string, docs []Doc) error {
	if len(docs) == 0 {
		return nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, d := range docs {
		if err := enc.Encode(d); err != nil {
			return err
		}
	}
	body, status, err := c.do(ctx, http.MethodPost, "/collections/"+c.Collection(locale)+"/documents/import?action=upsert", &buf, "text/plain")
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("typesense import: %d %s", status, body)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		if strings.Contains(line, `"success":false`) {
			return fmt.Errorf("typesense import: %s", line)
		}
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, locale, id string) error {
	body, status, err := c.json(ctx, http.MethodDelete, "/collections/"+c.Collection(locale)+"/documents/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}
	if status >= 300 && status != http.StatusNotFound {
		return fmt.Errorf("typesense delete: %d %s", status, body)
	}
	return nil
}

type Params struct {
	Query    string
	Category string // category slug
	Language string
	MinScore float64
	Sort     string // relevance (default) | score | stars | new
	Page     int
	PerPage  int
}

type Facet struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type Result struct {
	Found  int                `json:"found"`
	Page   int                `json:"page"`
	Hits   []Doc              `json:"hits"`
	Facets map[string][]Facet `json:"facets"`
	TookMS int                `json:"tookMs"`
}

func escapeFilter(s string) string { return "`" + strings.ReplaceAll(s, "`", "") + "`" }

func (c *Client) Search(ctx context.Context, locale string, p Params) (*Result, error) {
	q := strings.TrimSpace(p.Query)
	if q == "" {
		q = "*"
	}
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 50 {
		p.PerPage = 24
	}
	var filters []string
	if p.Category != "" {
		filters = append(filters, "category_slug:="+escapeFilter(p.Category))
	}
	if p.Language != "" {
		filters = append(filters, "language:="+escapeFilter(p.Language))
	}
	if p.MinScore > 0 {
		filters = append(filters, fmt.Sprintf("score:>=%.1f", p.MinScore))
	}
	sortBy := "_text_match:desc,score:desc,stars:desc"
	switch p.Sort {
	case "score":
		sortBy = "score:desc,stars:desc"
	case "stars":
		sortBy = "stars:desc"
	case "new":
		sortBy = "updated_at:desc"
	}
	if q == "*" && p.Sort == "" {
		sortBy = "score:desc,stars:desc"
	}
	v := url.Values{}
	v.Set("q", q)
	v.Set("query_by", "name,full_name,topics,tagline,description,language,category")
	v.Set("query_by_weights", "10,8,6,4,2,2,2")
	v.Set("prefix", "true")
	v.Set("num_typos", "1")
	v.Set("sort_by", sortBy)
	v.Set("facet_by", "category_slug,language,tier")
	v.Set("max_facet_values", "30")
	v.Set("page", fmt.Sprint(p.Page))
	v.Set("per_page", fmt.Sprint(p.PerPage))
	v.Set("highlight_full_fields", "")
	if len(filters) > 0 {
		v.Set("filter_by", strings.Join(filters, " && "))
	}
	body, status, err := c.do(ctx, http.MethodGet, "/collections/"+c.Collection(locale)+"/documents/search?"+v.Encode(), nil, "")
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return &Result{Page: p.Page, Hits: []Doc{}, Facets: map[string][]Facet{}}, nil
	}
	if status >= 300 {
		return nil, fmt.Errorf("typesense search: %d %s", status, body)
	}
	var raw struct {
		Found        int `json:"found"`
		Page         int `json:"page"`
		SearchTimeMS int `json:"search_time_ms"`
		Hits         []struct {
			Document Doc `json:"document"`
		} `json:"hits"`
		FacetCounts []struct {
			FieldName string `json:"field_name"`
			Counts    []struct {
				Value string `json:"value"`
				Count int    `json:"count"`
			} `json:"counts"`
		} `json:"facet_counts"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	res := &Result{Found: raw.Found, Page: raw.Page, TookMS: raw.SearchTimeMS, Hits: make([]Doc, 0, len(raw.Hits)), Facets: map[string][]Facet{}}
	for _, h := range raw.Hits {
		res.Hits = append(res.Hits, h.Document)
	}
	for _, f := range raw.FacetCounts {
		for _, c := range f.Counts {
			res.Facets[f.FieldName] = append(res.Facets[f.FieldName], Facet{Value: c.Value, Count: c.Count})
		}
	}
	return res, nil
}
