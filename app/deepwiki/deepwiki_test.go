package deepwiki

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func serve(t *testing.T, reply string, hits *int32) *Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\ndata: " + reply + "\n\n"))
	}))
	t.Cleanup(srv.Close)
	c := New()
	c.Endpoint = srv.URL
	return c
}

func TestCheck(t *testing.T) {
	cases := map[string]Status{
		`{"jsonrpc":"2.0","id":1,"result":{"content":[{"text":"Available pages for a/b","type":"text"}]}}`:                            Ready,
		`{"jsonrpc":"2.0","id":1,"result":{"isError":true,"content":[{"text":"Error: Repository not found. Visit x to index it."}]}}`: Missing,
		`{"jsonrpc":"2.0","id":1,"result":{"isError":true,"content":[{"text":"Error: upstream exploded"}]}}`:                          Unknown,
	}
	for reply, want := range cases {
		var hits int32
		c := serve(t, reply, &hits)
		if got := c.Check(context.Background(), "a/b"); got != want {
			t.Errorf("got %s want %s for %s", got, want, reply)
		}
		c.Check(context.Background(), "A/B") // cached, case-insensitive
		if hits != 1 {
			t.Errorf("expected one upstream call, got %d", hits)
		}
	}
}

func TestUnreachable(t *testing.T) {
	c := New()
	c.Endpoint = "http://127.0.0.1:1"
	if got := c.Check(context.Background(), "a/b"); got != Unknown {
		t.Fatalf("got %s", got)
	}
}
