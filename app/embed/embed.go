// Package embed creates text embeddings through any OpenAI-compatible /v1/embeddings endpoint.
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL string // e.g. https://api.openai.com or http://localhost:11434 (Ollama)
	APIKey  string
	Model   string
	HTTP    *http.Client
}

func New(baseURL, apiKey, model string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, Model: model, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

func (c *Client) Enabled() bool { return c != nil && c.BaseURL != "" }

func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(map[string]any{"model": c.Model, "input": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embeddings: HTTP %d: %.200s", resp.StatusCode, raw)
	}
	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 || len(out.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("embeddings: empty response")
	}
	return out.Data[0].Embedding, nil
}
