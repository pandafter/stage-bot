package mcp

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
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
	tokenTTL         = time.Hour
)

type Handler struct {
	cfg    *config.Config
	logger *zap.Logger
	http   *http.Client
	now    func() time.Time
	codeMu    sync.Mutex
	codeStore map[string]authCodeEntry
}

type authCodeEntry struct {
	CodeChallenge string
	RedirectURI   string
	Expires       time.Time
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
		now: time.Now,
		codeStore: make(map[string]authCodeEntry),
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
		c.Set("WWW-Authenticate", `Bearer realm="stage-bot-mcp"`)
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	if !h.validateAccessToken(token, h.now()) {
		c.Set("WWW-Authenticate", `Bearer realm="stage-bot-mcp"`)
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	return c.Next()
}


func (h *Handler) Token(c *fiber.Ctx) error {
	secret := strings.TrimSpace(h.cfg.MCPSecret)
	if secret == "" {
		return c.Status(fiber.StatusServiceUnavailable).SendString("MCP_SECRET not configured")
	}

	grantType := strings.TrimSpace(c.FormValue("grant_type"))
	clientID := strings.TrimSpace(c.FormValue("client_id"))
	clientSecret := strings.TrimSpace(c.FormValue("client_secret"))

		switch grantType {
	case "client_credentials":
		if subtle.ConstantTimeCompare([]byte(clientSecret), []byte(secret)) != 1 {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		if clientID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_client"})
		}
		accessToken, err := h.issueAccessToken(clientID, h.now())
		if err != nil {
			h.logger.Error("failed to issue mcp token", zap.Error(err))
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.JSON(fiber.Map{
			"access_token": accessToken,
			"token_type":   "bearer",
			"expires_in":   int(tokenTTL.Seconds()),
		})

	case "authorization_code":
		code := strings.TrimSpace(c.FormValue("code"))
		codeVerifier := strings.TrimSpace(c.FormValue("code_verifier"))
		if code == "" || codeVerifier == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request"})
		}
		h.codeMu.Lock()
		entry, ok := h.codeStore[code]
		if ok {
			delete(h.codeStore, code)
		}
		h.codeMu.Unlock()
		if !ok || h.now().After(entry.Expires) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_grant"})
		}
		// Verify PKCE: BASE64URL(SHA256(code_verifier)) == code_challenge
		digest := sha256.Sum256([]byte(codeVerifier))
		computed := base64.RawURLEncoding.EncodeToString(digest[:])
		if subtle.ConstantTimeCompare([]byte(computed), []byte(entry.CodeChallenge)) != 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_grant"})
		}
		if clientID == "" {
			clientID = "claude"
		}
		accessToken, err := h.issueAccessToken(clientID, h.now())
		if err != nil {
			h.logger.Error("failed to issue mcp token", zap.Error(err))
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.JSON(fiber.Map{
			"access_token": accessToken,
			"token_type":   "bearer",
			"expires_in":   int(tokenTTL.Seconds()),
		})

	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unsupported_grant_type"})
	}

	
	if subtle.ConstantTimeCompare([]byte(clientSecret), []byte(secret)) != 1 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_client"})
	}

	accessToken, err := h.issueAccessToken(clientID, h.now())
	if err != nil {
		h.logger.Error("failed to issue mcp token", zap.Error(err))
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	return c.JSON(fiber.Map{
		"access_token": accessToken,
		"token_type":   "bearer",
		"expires_in":   int(tokenTTL.Seconds()),
	})
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
			"description": "Create or update a file in the configured GitHub repository default branch",
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
		if strings.TrimSpace(path) == "" || strings.TrimSpace(message) == "" || strings.TrimSpace(content) == "" {
			return "path, content, and message are required", true
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
		return "bot configuration is not available", true
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
	if len(parts) != 2 {
		return "", "", fmt.Errorf("GITHUB_REPO must use owner/repo format (e.g., facebook/react)")
	}
	owner := strings.TrimSpace(parts[0])
	repo := strings.TrimSpace(parts[1])
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("GITHUB_REPO must use owner/repo format (e.g., facebook/react)")
	}
	ownerRepo = owner + "/" + repo
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
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
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

func (h *Handler) issueAccessToken(clientID string, now time.Time) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"cid": clientID,
		"exp": now.Add(tokenTTL).Unix(),
	})
	if err != nil {
		return "", err
	}
	sig := h.signTokenPayload(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func (h *Handler) validateAccessToken(token string, now time.Time) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	expectedSig := h.signTokenPayload(payload)
	if subtle.ConstantTimeCompare(signature, expectedSig) != 1 {
		return false
	}

	var claims struct {
		ClientID string `json:"cid"`
		Exp      int64  `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return false
	}
	if claims.ClientID == "" {
		return false
	}
	return now.Unix() < claims.Exp
}

func (h *Handler) signTokenPayload(payload []byte) []byte {
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(h.cfg.MCPSecret)))
	mac.Write(payload)
	return mac.Sum(nil)
}

func (h *Handler) Authorize(c *fiber.Ctx) error {
	state := c.Query("state")
	redirectURI := c.Query("redirect_uri")
	codeChallenge := c.Query("code_challenge")
	clientID := c.Query("client_id")

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Stage Bot – Autorizar</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
  body{font-family:system-ui,sans-serif;display:flex;align-items:center;
       justify-content:center;min-height:100vh;margin:0;background:#0f0f0f;color:#fff}
  .box{background:#1a1a1a;padding:2rem;border-radius:12px;width:90%%;max-width:360px}
  h2{margin:0 0 1rem}
  input[type=password]{width:100%%;padding:.75rem;border-radius:8px;border:1px solid #333;
    background:#0f0f0f;color:#fff;font-size:1rem;box-sizing:border-box;margin-bottom:1rem}
  button{width:100%%;padding:.75rem;background:#e63946;color:#fff;border:none;
    border-radius:8px;font-size:1rem;cursor:pointer}
  button:hover{background:#c1121f}
</style>
</head>
<body>
<div class="box">
  <h2>🏎️ Stage Bot MCP</h2>
  <p>Autoriza a Claude para conectarse al bot.</p>
  <form method="POST" action="/authorize">
    <input type="hidden" name="state" value="%s">
    <input type="hidden" name="redirect_uri" value="%s">
    <input type="hidden" name="code_challenge" value="%s">
    <input type="hidden" name="client_id" value="%s">
    <input type="password" name="password" placeholder="MCP Secret" autofocus>
    <button type="submit">Autorizar</button>
  </form>
</div>
</body>
</html>`, state, redirectURI, codeChallenge, clientID)

	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

func (h *Handler) AuthorizeSubmit(c *fiber.Ctx) error {
	secret := strings.TrimSpace(h.cfg.MCPSecret)
	password := strings.TrimSpace(c.FormValue("password"))
	state := c.FormValue("state")
	redirectURI := c.FormValue("redirect_uri")
	codeChallenge := c.FormValue("code_challenge")

	if subtle.ConstantTimeCompare([]byte(password), []byte(secret)) != 1 {
		c.Set("Content-Type", "text/html")
		return c.Status(fiber.StatusUnauthorized).SendString(`<!DOCTYPE html>
<html><body style="font-family:system-ui;text-align:center;padding:2rem;background:#0f0f0f;color:#fff">
<h2>❌ Contraseña incorrecta</h2>
<a href="javascript:history.back()" style="color:#e63946">Volver</a>
</body></html>`)
	}

	// Generate auth code
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	code := hex.EncodeToString(b)

	h.codeMu.Lock()
	h.codeStore[code] = authCodeEntry{
		CodeChallenge: codeChallenge,
		RedirectURI:   redirectURI,
		Expires:       h.now().Add(5 * time.Minute),
	}
	h.codeMu.Unlock()

	target := redirectURI + "?code=" + code + "&state=" + url.QueryEscape(state)
	return c.Redirect(target, fiber.StatusFound)
}

func (h *Handler) OAuthMetadata(c *fiber.Ctx) error {
	baseURL := strings.TrimRight(h.cfg.PublicURL, "/")
	if baseURL == "" {
		baseURL = "https://stage-bot-production.up.railway.app"
	}
	return c.JSON(map[string]any{
		"issuer":                 baseURL,
		"authorization_endpoint": baseURL + "/authorize",
		"token_endpoint":         baseURL + "/mcp/token",
		"response_types_supported": []string{"code"},
		"grant_types_supported":    []string{"authorization_code", "client_credentials"},
		"code_challenge_methods_supported": []string{"S256"},
	})
}

