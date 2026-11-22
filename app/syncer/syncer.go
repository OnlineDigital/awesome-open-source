// Package syncer refreshes forge (GitHub, GitLab, ...) data, recomputes scores and updates the search index and embeddings.
package syncer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"awesomeopensource.dev/app/embed"
	"awesomeopensource.dev/app/forge"
	"awesomeopensource.dev/app/scoring"
	"awesomeopensource.dev/app/search"
	"awesomeopensource.dev/app/store"
)

type Syncer struct {
	Store  *store.Store
	Forges *forge.Registry
	Index  *search.Indexer
	Embed  *embed.Client
	Delay  time.Duration // pause between forge calls
}

// SyncOne fetches, scores and persists a single project, then refreshes search and embedding.
func (s *Syncer) SyncOne(ctx context.Context, id int64) error {
	t, err := s.Store.ProjectTarget(ctx, id)
	if err != nil {
		return err
	}
	return s.syncTarget(ctx, t)
}

func (s *Syncer) syncTarget(ctx context.Context, t store.SyncTarget) error {
	d, ok := s.Forges.Get(t.Forge)
	if !ok {
		err := fmt.Errorf("unsupported forge %q", t.Forge)
		_ = s.Store.MarkSyncError(ctx, t.ID, err.Error())
		return err
	}
	repo, err := d.Fetch(ctx, t.Owner, t.Name)
	if err != nil {
		if !errors.Is(err, forge.ErrRateLimited) {
			_ = s.Store.MarkSyncError(ctx, t.ID, err.Error())
		}
		return err
	}
	if err := s.Store.ApplyRepo(ctx, t.ID, repo, scoring.Compute(repo.Signals)); err != nil {
		return err
	}
	if s.Embed.Enabled() {
		if err := s.EmbedProject(ctx, t.ID); err != nil {
			log.Printf("syncer: embedding %s: %v", t.FullName(), err)
		}
	}
	if s.Index != nil {
		if err := s.Index.IndexProject(ctx, t.ID); err != nil {
			log.Printf("syncer: indexing %s: %v", t.FullName(), err)
		}
	}
	return nil
}

// EmbedProject stores a vector built from the default-locale text.
func (s *Syncer) EmbedProject(ctx context.Context, id int64) error {
	p, err := s.Store.GetProject(ctx, id, s.Store.DefaultLocale)
	if err != nil {
		return err
	}
	text := fmt.Sprintf("%s\n%s\n%s\nCategory: %s\nLanguage: %s\nTopics: %s", p.FullName, p.Tagline, p.Description, p.CategoryName, p.Language, strings.Join(p.Topics, ", "))
	vec, err := s.Embed.Embed(ctx, text)
	if err != nil {
		return err
	}
	return s.Store.SetEmbedding(ctx, id, vec)
}

// SyncStale refreshes up to limit projects not synced within olderThan.
func (s *Syncer) SyncStale(ctx context.Context, olderThan time.Duration, limit int) (ok, failed int, err error) {
	targets, err := s.Store.StaleProjects(ctx, olderThan, limit)
	if err != nil {
		return 0, 0, err
	}
	limited := map[string]error{} // forges that hit their rate limit; others keep syncing
	for _, t := range targets {
		if ctx.Err() != nil {
			return ok, failed, ctx.Err()
		}
		if limited[t.Forge] != nil {
			continue
		}
		if err := s.syncTarget(ctx, t); err != nil {
			failed++
			log.Printf("syncer: %s %s: %v", t.Forge, t.FullName(), err)
			if errors.Is(err, forge.ErrRateLimited) {
				limited[t.Forge] = err
				if len(limited) == len(s.Forges.Drivers()) {
					return ok, failed, err
				}
			}
		} else {
			ok++
		}
		select {
		case <-ctx.Done():
			return ok, failed, ctx.Err()
		case <-time.After(s.Delay):
		}
	}
	for _, err := range limited {
		return ok, failed, err
	}
	return ok, failed, nil
}

// Run refreshes stale projects periodically until ctx is cancelled.
func (s *Syncer) Run(ctx context.Context, every, staleAge time.Duration, batch int) {
	log.Printf("syncer: background refresh every %s (stale after %s, batch %d)", every, staleAge, batch)
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		ok, failed, err := s.SyncStale(ctx, staleAge, batch)
		if ok+failed > 0 || err != nil {
			log.Printf("syncer: refreshed %d projects, %d failed, err=%v", ok, failed, err)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
