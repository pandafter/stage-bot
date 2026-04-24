package mcp

import (
	"bufio"
	"bytes"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/config"
)

const (
	githubAPIBase    = "https://api.github.com"
	githubAPIVersion = "2022-11-28"
	defaultBranch    = "main"
	pingInterval     = 20 * time.Second
)

type Handler struct {
	cfg    *config.Config
	logger *zap.Logger
	http   *http.Client
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewHandler(cfg *config.Config, logger *zap.Logger) *Handler {
	return &Handler{
		cfg:    cfg,
		logger: logger,
		http: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (h *Handler) Auth(c *fiber.Ctx) error {
	secret := strings.TrimSpace(h.cfg.MCPSecret)
	if secret == "" {
		return c.Status(fiber.StatusServiceUnavailable).SendString("MCP_SECRET not configured")
	}

	authHeader := strings.TrimSpace(c.Get("Authorization"))
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	if subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	return c.Next()
}

func (h *Handler) Stream(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		_ = writeSSE(w, "ready", map[string]any{
			"server": "stage-bot-mcp",
			"path":   "/mcp",
		})

		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := writeSSE(w, "ping", map[string]any{
				"time": time.Now().UTC().Format(time.RFC3339),
			}); err != nil {
				return
			}
		}
	})

	return nil
}

func writeSSE(w *bufio.Writer, event string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(b)); err != nil {
		return err
	}
	return w.Flush()
}

func (h *Handler) HandleJSONRPC(c *fiber.Ctx) error {
	var req rpcRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.JSON(rpcResponse{
			JSONRPC: "2.0",
			Error: &rpcError{
				Code:    -32700,
				Message: "parse error",
			},
		})
	}

	if req.JSONRPC != "2.0" {
		return c.JSON(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &rpcError{
				Code:    -32600,
				Message: "invalid request",
			},
		})
	}

	switch req.Method {
	case "initialize":
		return c.JSON(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]any{
					"name":    "stage-bot-mcp",
					"version": "1.0.0",
				},
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
			},
		})
	case "tools/list":
		return c.JSON(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": h.tools(),
			},
		})
	case "tools/call":
		return h.handleToolsCall(c, req)
	default:
		return c.JSON(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &rpcError{
				Code:    -32601,
				Message: "method not found",
			},
		})
	}
}

func (h *Handler) tools() []map[string]any {
	return []map[string]any{
		{
			"name":        "get_file",
			"description": "Get file content from the configured GitHub repository",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"path"},
				"properties": map[string]any{
					"path": map[string]any{
						"type": "string",
					},
				},
			},
		},
		{
			"name":        "edit_file",
			"description": "Create or update a file in main branch in the configured GitHub repository",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"path", "content", "message"},
				"properties": map[string]any{
					"path": map[string]any{
						"type": "string",
					},
					"content": map[string]any{
						"type": "string",
					},
					"message": map[string]any{
						"type": "string",
					},
				},
			},
		},
		{
			"name":        "get_bot_config",
			"description": "Returns current bot prompt config from BOT_PROMPT or SYSTEM_PROMPT",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

func (h *Handler) handleToolsCall(c *fiber.Ctx, req rpcRequest) error {
	type callParams struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	var params callParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return c.JSON(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &rpcError{
				Code:    -32602,
				Message: "invalid params",
			},
		})
	}

	result, isErr := h.callTool(params.Name, params.Arguments)
	return c.JSON(rpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": result,
				},
			},
			"isError": isErr,
		},
	})
}

func (h *Handler) callTool(name string, args map[string]interface{}) (string, bool) {
	switch name {
	case "get_file":
		path, ok := args["path"].(string)
		if !ok {
			return "path must be a string", true
		}
		if strings.TrimSpace(path) == "" {
			return "path is required", true
		}
		content, err := h.githubGetFile(path)
		if err != nil {
			return err.Error(), true
		}
		return content, false
	case "edit_file":
		path, ok := args["path"].(string)
		if !ok {
			return "path must be a string", true
		}
		content, ok := args["content"].(string)
		if !ok {
			return "content must be a string", true
		}
		message, ok := args["message"].(string)
		if !ok {
			return "message must be a string", true
		}
		if strings.TrimSpace(path) == "" || strings.TrimSpace(message) == "" {
			return "path and message are required strings (content may be an empty string)", true
		}
		resp, err := h.githubEditFile(path, content, message)
		if err != nil {
			return err.Error(), true
		}
		return resp, false
	case "get_bot_config":
		if strings.TrimSpace(h.cfg.BotPrompt) != "" {
			return h.cfg.BotPrompt, false
		}
		if strings.TrimSpace(h.cfg.SystemPrompt) != "" {
			return h.cfg.SystemPrompt, false
		}
		return "BOT_PROMPT and SYSTEM_PROMPT are not configured", true
	default:
		return "unknown tool", true
	}
}

func (h *Handler) githubGetFile(path string) (string, error) {
	ownerRepo, token, err := h.githubAuth()
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/repos/%s/contents/%s", githubAPIBase, ownerRepo, escapePath(path))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	setGitHubHeaders(req, token)

	resp, err := h.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("github get_file failed: %s", extractGitHubErr(body))
	}

	var payload struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if payload.Encoding != "base64" {
		return "", fmt.Errorf("unsupported github encoding: %s", payload.Encoding)
	}

	decoded, err := decodeGitHubBase64(payload.Content)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func (h *Handler) githubEditFile(path, content, message string) (string, error) {
	ownerRepo, token, err := h.githubAuth()
	if err != nil {
		return "", err
	}

	sha, err := h.getCurrentFileSHA(ownerRepo, token, path)
	if err != nil {
		return "", err
	}

	payload := map[string]any{
		"message": message,
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
		"branch":  defaultBranch,
	}
	if sha != "" {
		payload["sha"] = sha
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/repos/%s/contents/%s", githubAPIBase, ownerRepo, escapePath(path))
	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	setGitHubHeaders(req, token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("github edit_file failed: %s", extractGitHubErr(respBody))
	}

	var out struct {
		Content struct {
			Path string `json:"path"`
			SHA  string `json:"sha"`
		} `json:"content"`
		Commit struct {
			SHA     string `json:"sha"`
			HTMLURL string `json:"html_url"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", err
	}
	return fmt.Sprintf("updated %s on %s\ncommit: %s\nurl: %s", out.Content.Path, defaultBranch, out.Commit.SHA, out.Commit.HTMLURL), nil
}

func (h *Handler) getCurrentFileSHA(ownerRepo, token, path string) (string, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", githubAPIBase, ownerRepo, escapePath(path), defaultBranch)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	setGitHubHeaders(req, token)

	resp, err := h.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("github lookup failed: %s", extractGitHubErr(body))
	}

	var payload struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	return payload.SHA, nil
}

func (h *Handler) githubAuth() (ownerRepo, token string, err error) {
	token = strings.TrimSpace(h.cfg.GitHubToken)
	if token == "" {
		return "", "", fmt.Errorf("GITHUB_TOKEN is not configured")
	}
	ownerRepo = strings.TrimSpace(h.cfg.GitHubRepo)
	parts := strings.Split(ownerRepo, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("GITHUB_REPO must use owner/repo format (e.g., facebook/react)")
	}
	return ownerRepo, token, nil
}

func setGitHubHeaders(req *http.Request, token string) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	req.Header.Set("User-Agent", "stage-bot-mcp")
}

func escapePath(path string) string {
	trimmed := strings.TrimPrefix(strings.TrimSpace(path), "/")
	escaped := url.PathEscape(trimmed)
	return strings.ReplaceAll(escaped, "%2F", "/")
}

func extractGitHubErr(body []byte) string {
	var parsed struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Message != "" {
		return parsed.Message
	}
	return strings.TrimSpace(string(body))
}

// decodeGitHubBase64 strips line breaks because GitHub's contents API may wrap
// base64 payloads with newlines.
func decodeGitHubBase64(content string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
}
