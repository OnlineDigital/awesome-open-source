// Package mcp serves the admin operations to AI agents over the Model Context Protocol.
// Two transports share one JSON-RPC core: legacy HTTP+SSE (GET .../sse + POST .../messages) and streamable HTTP (POST ...).
// Access is controlled by a secret in the URL, managed in Admin > Settings.
package mcp

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"awesomeopensource.dev/app/application"
	"github.com/gofiber/fiber/v3"
)

const (
	latestProtocol = "2025-03-26"
	serverName     = "awesome-open-source-admin"
)

var supportedProtocols = map[string]bool{"2024-11-05": true, "2025-03-26": true, "2025-06-18": true}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type session struct {
	token string
	out   chan []byte
}

type Server struct {
	app      *application.Application
	tools    []Tool
	byName   map[string]*Tool
	mu       sync.Mutex
	sessions map[string]*session
}

func New(app *application.Application) *Server {
	s := &Server{app: app, sessions: map[string]*session{}, byName: map[string]*Tool{}}
	s.tools = buildTools(app)
	for i := range s.tools {
		s.byName[s.tools[i].Name] = &s.tools[i]
	}
	return s
}

// handle executes one JSON-RPC message. It returns nil for notifications.
func (s *Server) handle(ctx context.Context, raw []byte) *rpcResponse {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return &rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{Code: -32700, Message: "parse error"}}
	}
	isNotification := len(req.ID) == 0
	fail := func(code int, msg string) *rpcResponse {
		if isNotification {
			return nil
		}
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: code, Message: msg}}
	}
	ok := func(result any) *rpcResponse {
		if isNotification {
			return nil
		}
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
	}
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		version := latestProtocol
		if supportedProtocols[p.ProtocolVersion] {
			version = p.ProtocolVersion
		}
		return ok(map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": serverName, "version": "1.0.0"},
			"instructions":    "Manage the Awesome Open Source catalog: projects (GitHub repos with scores, translations, screenshots), categories, and topic pages with feature comparison tables. Start with get_stats, list_projects, list_topics. Texts are per locale; the default locale is required.",
		})
	case "ping":
		return ok(map[string]any{})
	case "tools/list":
		list := make([]map[string]any, 0, len(s.tools))
		for _, t := range s.tools {
			list = append(list, map[string]any{"name": t.Name, "description": t.Description, "inputSchema": t.Schema})
		}
		return ok(map[string]any{"tools": list})
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return fail(-32602, "invalid params")
		}
		tool, found := s.byName[p.Name]
		if !found {
			return fail(-32602, "unknown tool: "+p.Name)
		}
		if len(p.Arguments) == 0 {
			p.Arguments = json.RawMessage("{}")
		}
		cctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()
		result, err := tool.Handler(cctx, p.Arguments)
		if err != nil {
			return ok(map[string]any{"isError": true, "content": []map[string]any{{"type": "text", "text": err.Error()}}})
		}
		text, _ := json.MarshalIndent(result, "", "  ")
		return ok(map[string]any{"content": []map[string]any{{"type": "text", "text": string(text)}}})
	case "resources/list":
		return ok(map[string]any{"resources": []any{}})
	case "prompts/list":
		return ok(map[string]any{"prompts": []any{}})
	}
	if isNotification || len(req.Method) > 14 && req.Method[:14] == "notifications/" {
		return nil
	}
	return fail(-32601, "method not found: "+req.Method)
}

func (s *Server) tokenOK(c fiber.Ctx) (string, bool) {
	token := application.RouteParameter(c, "token")
	ok, err := s.app.Store.MCPTokenValid(c.Context(), token)
	return token, err == nil && ok
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SSE opens the event stream (legacy transport). The first event announces the URL to POST messages to.
func (s *Server) SSE() fiber.Handler {
	return func(c fiber.Ctx) error {
		token, ok := s.tokenOK(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).SendString("invalid or revoked token")
		}
		sess := &session{token: token, out: make(chan []byte, 64)}
		id := newID()
		s.mu.Lock()
		s.sessions[id] = sess
		s.mu.Unlock()
		endpoint := fmt.Sprintf("/mcp/%s/messages?sessionId=%s", token, id)
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache, no-transform")
		c.Set("Connection", "keep-alive")
		c.Set("X-Accel-Buffering", "no")
		return c.SendStreamWriter(func(w *bufio.Writer) {
			defer func() {
				s.mu.Lock()
				delete(s.sessions, id)
				s.mu.Unlock()
			}()
			write := func(event, data string) bool {
				if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
					return false
				}
				return w.Flush() == nil
			}
			if !write("endpoint", endpoint) {
				return
			}
			tick := time.NewTicker(15 * time.Second)
			defer tick.Stop()
			for {
				select {
				case msg := <-sess.out:
					if !write("message", string(msg)) {
						return
					}
				case <-tick.C:
					valid, err := s.app.Store.MCPTokenValid(context.Background(), token)
					if err != nil || !valid {
						return
					}
					if _, err := io.WriteString(w, ": keepalive\n\n"); err != nil || w.Flush() != nil {
						return
					}
				}
			}
		})
	}
}

// Messages receives JSON-RPC requests for an SSE session; responses travel back over the stream.
func (s *Server) Messages() fiber.Handler {
	return func(c fiber.Ctx) error {
		token, ok := s.tokenOK(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).SendString("invalid or revoked token")
		}
		s.mu.Lock()
		sess := s.sessions[c.Query("sessionId")]
		s.mu.Unlock()
		if sess == nil || sess.token != token {
			return c.Status(fiber.StatusNotFound).SendString("unknown session")
		}
		body := append([]byte(nil), c.Body()...)
		go func() {
			for _, raw := range splitBatch(body) {
				if resp := s.handle(context.Background(), raw); resp != nil {
					out, _ := json.Marshal(resp)
					select {
					case sess.out <- out:
					case <-time.After(10 * time.Second):
					}
				}
			}
		}()
		return c.Status(fiber.StatusAccepted).SendString("Accepted")
	}
}

// HTTP implements the streamable HTTP transport: POST a JSON-RPC message (or batch), get the JSON response.
func (s *Server) HTTP() fiber.Handler {
	return func(c fiber.Ctx) error {
		if _, ok := s.tokenOK(c); !ok {
			return c.Status(fiber.StatusUnauthorized).SendString("invalid or revoked token")
		}
		switch c.Method() {
		case fiber.MethodPost:
		case fiber.MethodDelete:
			return c.SendStatus(fiber.StatusOK)
		default:
			c.Set("Allow", "POST, DELETE")
			return c.Status(fiber.StatusMethodNotAllowed).SendString("use POST (or the /sse endpoint)")
		}
		raws := splitBatch(c.Body())
		var responses []*rpcResponse
		for _, raw := range raws {
			if resp := s.handle(c.Context(), raw); resp != nil {
				responses = append(responses, resp)
			}
		}
		if len(responses) == 0 {
			return c.SendStatus(fiber.StatusAccepted)
		}
		if len(responses) == 1 && len(raws) == 1 && !isBatch(c.Body()) {
			return c.JSON(responses[0])
		}
		return c.JSON(responses)
	}
}

func isBatch(b []byte) bool {
	for _, ch := range b {
		if ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t' {
			continue
		}
		return ch == '['
	}
	return false
}

func splitBatch(b []byte) [][]byte {
	if !isBatch(b) {
		return [][]byte{b}
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(b, &arr); err != nil {
		return [][]byte{b}
	}
	out := make([][]byte, 0, len(arr))
	for _, m := range arr {
		out = append(out, m)
	}
	return out
}
