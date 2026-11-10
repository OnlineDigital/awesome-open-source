// Package translate machine-translates short project/category texts with the Anthropic Messages API.
package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"awesomeopensource.dev/app/i18n"
)

var ErrNotConfigured = errors.New("translation is not configured (set ANTHROPIC_API_KEY)")

type Translator struct {
	APIKey  string
	Model   string
	BaseURL string // default https://api.anthropic.com
	HTTP    *http.Client
}

func New(apiKey, model string) *Translator {
	return &Translator{APIKey: apiKey, Model: model, BaseURL: "https://api.anthropic.com", HTTP: &http.Client{Timeout: 90 * time.Second}}
}

func (t *Translator) Enabled() bool { return t != nil && t.APIKey != "" }

const system = `You translate short descriptions of open source software for a software directory.
Rules: keep product names, library names, code identifiers, URLs, and technical acronyms untranslated;
keep Markdown formatting and paragraph breaks; do not add or remove information; use a neutral, clear tone.
Reply with ONLY a JSON object with exactly the same keys as the input, with translated string values.`

// Fields translates every non-empty value of fields from one locale to another.
func (t *Translator) Fields(ctx context.Context, from, to string, fields map[string]string) (map[string]string, error) {
	if !t.Enabled() {
		return nil, ErrNotConfigured
	}
	in := map[string]string{}
	for k, v := range fields {
		if strings.TrimSpace(v) != "" {
			in[k] = v
		}
	}
	if len(in) == 0 {
		return map[string]string{}, nil
	}
	payload, _ := json.Marshal(in)
	prompt := fmt.Sprintf("Translate the values of this JSON object from %s to %s.\n\n%s", i18n.EnglishNames[from], i18n.EnglishNames[to], payload)
	body, _ := json.Marshal(map[string]any{
		"model": t.Model, "max_tokens": 2048, "system": system,
		"messages": []map[string]any{{"role": "user", "content": prompt}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(t.BaseURL, "/")+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", t.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	resp, err := t.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("anthropic: HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	var msg struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, err
	}
	var text string
	for _, c := range msg.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}
	out, err := ParseJSONObject(text)
	if err != nil {
		return nil, err
	}
	for k := range in {
		if strings.TrimSpace(out[k]) == "" {
			return nil, fmt.Errorf("translation response is missing %q", k)
		}
	}
	return out, nil
}

// ParseJSONObject extracts the first {...} object from a model reply (tolerates code fences).
func ParseJSONObject(s string) (map[string]string, error) {
	start, end := strings.Index(s, "{"), strings.LastIndex(s, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("translation response is not JSON: %s", truncate(s, 120))
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s[start:end+1]), &m); err != nil {
		return nil, fmt.Errorf("translation response is not valid JSON: %w", err)
	}
	return m, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
