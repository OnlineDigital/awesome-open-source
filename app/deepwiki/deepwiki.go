// Package deepwiki reports whether a GitHub repository has an AI-generated wiki on
// deepwiki.com. The website itself blocks scripted clients, so it asks DeepWiki's
// public MCP endpoint, which answers "Repository not found" for unindexed repos.
package deepwiki

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Status string

const (
	Ready   Status = "ready"   // wiki exists
	Missing Status = "missing" // repository is not indexed yet
	Unknown Status = "unknown" // DeepWiki could not be reached or answered unexpectedly
)

const (
	siteURL     = "https://deepwiki.com/"
	ttlReady    = 24 * time.Hour
	ttlMissing  = 30 * time.Minute // short, so a freshly requested index shows up soon
	ttlUnknown  = 2 * time.Minute
	maxCacheLen = 5000
)

type entry struct {
	status  Status
	expires time.Time
}

type Client struct {
	Endpoint string // MCP endpoint, default https://mcp.deepwiki.com/mcp
	HTTP     *http.Client

	mu    sync.Mutex
	cache map[string]entry
}

func New() *Client {
	return &Client{Endpoint: "https://mcp.deepwiki.com/mcp", HTTP: &http.Client{Timeout: 10 * time.Second}, cache: map[string]entry{}}
}

// URL is the DeepWiki page for "owner/repo". Visiting it is also how indexing is requested.
func URL(fullName string) string { return siteURL + fullName }

// Check returns the cached status for "owner/repo", asking DeepWiki when the cache is cold.
func (c *Client) Check(ctx context.Context, fullName string) Status {
	key := strings.ToLower(fullName)
	c.mu.Lock()
	e, ok := c.cache[key]
	c.mu.Unlock()
	if ok && time.Now().Before(e.expires) {
		return e.status
	}
	st := c.fetch(ctx, fullName)
	ttl := map[Status]time.Duration{Ready: ttlReady, Missing: ttlMissing, Unknown: ttlUnknown}[st]
	c.mu.Lock()
	if len(c.cache) >= maxCacheLen {
		c.cache = map[string]entry{}
	}
	c.cache[key] = entry{st, time.Now().Add(ttl)}
	c.mu.Unlock()
	return st
}

func (c *Client) fetch(ctx context.Context, fullName string) Status {
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": "read_wiki_structure", "arguments": map[string]string{"repoName": fullName}}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return Unknown
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Unknown
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Unknown
	}
	st, err := parse(resp.Body)
	if err != nil {
		return Unknown
	}
	return st
}

type rpcReply struct {
	Result struct {
		IsError bool `json:"isError"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"result"`
	Error *struct{ Message string } `json:"error"`
}

// parse reads a JSON or server-sent-events reply and classifies it.
func parse(r io.Reader) (Status, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		line = strings.TrimPrefix(line, "data:")
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var rep rpcReply
		if json.Unmarshal([]byte(line), &rep) != nil {
			continue
		}
		if rep.Error != nil {
			return Unknown, fmt.Errorf("deepwiki: %s", rep.Error.Message)
		}
		if !rep.Result.IsError {
			return Ready, nil
		}
		for _, ct := range rep.Result.Content {
			if strings.Contains(strings.ToLower(ct.Text), "repository not found") {
				return Missing, nil
			}
		}
		return Unknown, nil
	}
	return Unknown, fmt.Errorf("deepwiki: empty reply")
}
