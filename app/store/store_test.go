package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"awesomeopensource.dev/app/db"
	"awesomeopensource.dev/app/github"
	"awesomeopensource.dev/app/scoring"
	"awesomeopensource.dev/app/store"
)

// Integration test: needs TEST_DATABASE_URL (a throwaway database with pgvector).
func TestStoreLifecycle(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if applied, _ := db.Migrate(ctx, pool); len(applied) != 0 {
		t.Fatalf("migrations should be idempotent, applied %v", applied)
	}
	s := store.New(pool, "en")

	catID, err := s.SaveCategory(ctx, store.Category{Slug: "Databases & Storage", Icon: "database"}, []store.CategoryTranslation{
		{Locale: "en", Name: "Databases", Description: "Stores"}, {Locale: "ro", Name: "Baze de date"}})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := s.CategoryBySlug(ctx, "databases-storage", "ro")
	if err != nil || cat.Name != "Baze de date" || cat.Description != "Stores" {
		t.Fatalf("category fallback wrong: %+v %v", cat, err)
	}

	id, err := s.CreateProject(ctx, "acme", "Widget", &catID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateProject(ctx, "acme", "Widget", &catID); err != store.ErrExists {
		t.Fatalf("duplicate should be ErrExists, got %v", err)
	}
	now := time.Now()
	repo := &github.Repo{Owner: "acme", Name: "Widget", Description: "A widget", Stars: 42, Topics: []string{"go", "db"}, Language: "Go",
		PushedAt: now, Signals: scoring.Signals{LastCommitAt: now}, LastRelease: &scoring.Release{Tag: "v1.0.0", PublishedAt: now}}
	if err := s.ApplyGitHub(ctx, id, repo, scoring.Compute(repo.Signals)); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateProject(ctx, id, store.ProjectEdit{CategoryID: &catID, YouTubeID: "dQw4w9WgXcQ", Status: "published"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveTranslation(ctx, id, store.Translation{Locale: "ro", Tagline: "Un widget", Description: "Descriere", Source: "machine"}); err != nil {
		t.Fatal(err)
	}
	p, err := s.GetProjectBySlug(ctx, "acme-widget", "ro", true)
	if err != nil {
		t.Fatal(err)
	}
	if p.Tagline != "Un widget" || !p.Machine || p.Stars != 42 || p.Score == nil || p.CategoryName != "Baze de date" || p.LastReleaseTag != "v1.0.0" {
		t.Fatalf("%+v", p)
	}
	if en, _ := s.GetProjectBySlug(ctx, "acme-widget", "en", true); en.Tagline != "A widget" {
		t.Fatalf("english tagline: %q", en.Tagline)
	}
	// Re-syncing must not overwrite edited english text.
	_ = s.SaveTranslation(ctx, id, store.Translation{Locale: "en", Tagline: "Edited", Description: "d"})
	_ = s.ApplyGitHub(ctx, id, repo, scoring.Compute(repo.Signals))
	if en, _ := s.GetProjectBySlug(ctx, "acme-widget", "en", true); en.Tagline != "Edited" {
		t.Fatalf("sync overwrote edit: %q", en.Tagline)
	}

	list, total, err := s.ListProjects(ctx, store.ProjectFilter{Locale: "ro", Status: "published", CategoryID: catID, MinScore: 1, Query: "acme"})
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("list: %v total=%d", err, total)
	}
	if err := s.SetEmbedding(ctx, id, []float32{0.1, 0.2, 0.3}); err != nil {
		t.Fatal(err)
	}
	id2, _ := s.CreateProject(ctx, "acme", "Gadget", &catID)
	_ = s.UpdateProject(ctx, id2, store.ProjectEdit{CategoryID: &catID, Status: "published"})
	_ = s.SetEmbedding(ctx, id2, []float32{0.1, 0.2, 0.31})
	rel, err := s.Related(ctx, p, "en", 5)
	if err != nil || len(rel) != 1 || rel[0].Slug != "acme-gadget" {
		t.Fatalf("related: %v %+v", err, rel)
	}
	if _, err := s.AddScreenshot(ctx, id, "/uploads/1/a.png", "alt"); err != nil {
		t.Fatal(err)
	}
	if st, err := s.Stats(ctx); err != nil || st.Projects != 2 || st.Published != 2 {
		t.Fatalf("stats %+v %v", st, err)
	}
	if _, err := s.DeleteProject(ctx, id); err != nil {
		t.Fatal(err)
	}
}
