package search

import (
	"context"
	"fmt"
	"strconv"

	"awesomeopensource.dev/app/store"
)

// Indexer mirrors published projects into Typesense, one collection per locale.
type Indexer struct {
	Client  *Client
	Store   *store.Store
	Locales []string
}

func docFor(p *store.Project) Doc {
	d := Doc{
		ID: strconv.FormatInt(p.ID, 10), Slug: p.Slug, Name: p.Repo, FullName: p.FullName,
		Tagline: p.Tagline, Description: p.Description, Topics: p.Topics, Language: p.Language,
		Category: p.CategoryName, CategorySlug: p.CategorySlug, Stars: int32(p.Stars), Featured: p.Featured,
		HasVideo: p.YouTubeID != "", Avatar: p.AvatarURL, UpdatedAt: p.UpdatedAt.Unix(),
	}
	if d.Topics == nil {
		d.Topics = []string{}
	}
	if p.Score != nil {
		d.Score = *p.Score
	}
	if p.Tier != nil {
		d.Tier = int32(*p.Tier)
	}
	return d
}

// IndexProject upserts (published) or removes (draft/deleted) one project in every locale.
func (ix *Indexer) IndexProject(ctx context.Context, id int64) error {
	if !ix.Client.Enabled() {
		return nil
	}
	for _, loc := range ix.Locales {
		if err := ix.Client.EnsureCollection(ctx, loc); err != nil {
			return err
		}
		p, err := ix.Store.GetProject(ctx, id, loc)
		if err != nil || p.Status != "published" {
			if derr := ix.Client.Delete(ctx, loc, strconv.FormatInt(id, 10)); derr != nil {
				return derr
			}
			continue
		}
		if err := ix.Client.Import(ctx, loc, []Doc{docFor(p)}); err != nil {
			return err
		}
	}
	return nil
}

func (ix *Indexer) Remove(ctx context.Context, id int64) error {
	if !ix.Client.Enabled() {
		return nil
	}
	for _, loc := range ix.Locales {
		if err := ix.Client.Delete(ctx, loc, strconv.FormatInt(id, 10)); err != nil {
			return err
		}
	}
	return nil
}

// Reindex rebuilds every locale collection from the database. It returns the project count.
func (ix *Indexer) Reindex(ctx context.Context) (int, error) {
	if !ix.Client.Enabled() {
		return 0, fmt.Errorf("typesense is not configured (TYPESENSE_API_KEY)")
	}
	total := 0
	for _, loc := range ix.Locales {
		if err := ix.Client.RecreateCollection(ctx, loc); err != nil {
			return 0, err
		}
		const page = 200
		total = 0
		for offset := 0; ; offset += page {
			ps, _, err := ix.Store.ListProjects(ctx, store.ProjectFilter{Locale: loc, Status: "published", Limit: page, Offset: offset, Sort: "name"})
			if err != nil {
				return 0, err
			}
			docs := make([]Doc, 0, len(ps))
			for i := range ps {
				docs = append(docs, docFor(&ps[i]))
			}
			if err := ix.Client.Import(ctx, loc, docs); err != nil {
				return 0, err
			}
			total += len(ps)
			if len(ps) < page {
				break
			}
		}
	}
	return total, nil
}
