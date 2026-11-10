package translate

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if r.Header.Get("x-api-key") != "k" || !strings.Contains(string(b), "Romanian") || !strings.Contains(string(b), "fast database") {
			t.Errorf("bad request %s", b)
		}
		io.WriteString(w, "{\"content\":[{\"type\":\"text\",\"text\":\"```json\\n{\\\"tagline\\\":\\\"bază de date rapidă\\\"}\\n```\"}]}")
	}))
	defer srv.Close()
	tr := New("k", "m")
	tr.BaseURL = srv.URL
	out, err := tr.Fields(context.Background(), "en", "ro", map[string]string{"tagline": "fast database", "description": " "})
	if err != nil || out["tagline"] != "bază de date rapidă" {
		t.Fatalf("%v %v", out, err)
	}
	if _, err := New("", "m").Fields(context.Background(), "en", "ro", nil); err != ErrNotConfigured {
		t.Fatal(err)
	}
}
