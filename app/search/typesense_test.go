package search

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchBuildsQueryAndParses(t *testing.T) {
	var gotQuery, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		if r.Header.Get("X-TYPESENSE-API-KEY") != "k" {
			t.Error("api key header missing")
		}
		io.WriteString(w, `{"found":1,"page":1,"search_time_ms":2,"hits":[{"document":{"id":"1","slug":"a-b","name":"b","score":4.2}}],
		"facet_counts":[{"field_name":"category_slug","counts":[{"value":"db","count":3}]}]}`)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "k", "t")
	res, err := c.Search(context.Background(), "ro", Params{Query: "bază de date", Category: "db", MinScore: 3.5})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/collections/t_projects_ro/documents/search" {
		t.Fatal(gotPath)
	}
	for _, want := range []string{"filter_by=category_slug%3A%3D%60db%60+%26%26+score%3A%3E%3D3.5", "q=baz%C4%83+de+date", "prefix=true"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q lacks %q", gotQuery, want)
		}
	}
	if res.Found != 1 || res.Hits[0].Slug != "a-b" || res.Facets["category_slug"][0].Count != 3 {
		t.Fatalf("%+v", res)
	}
}

func TestUnavailable(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "k", "t")
	if _, err := c.Search(context.Background(), "en", Params{}); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("got %v", err)
	}
}
