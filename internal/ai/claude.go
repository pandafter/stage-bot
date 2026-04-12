package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	claudeAPIURL   = "https://api.anthropic.com/v1/messages"
	claudeModel    = "claude-sonnet-4-6"
	claudeVersion  = "2023-06-01"
	defaultMaxToks = 300
)

type Claude struct {
	apiKey     string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewClaude(apiKey string, logger *zap.Logger) *Claude {
	return &Claude{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system"`
	Messages  []ClaudeMessage `json:"messages"`
}

// ClaudeMessage represents a message in the Claude conversation history.
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type claudeError struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Chat sends a message to Claude with a system prompt and conversation history.
func (c *Claude) Chat(systemPrompt string, messages []ClaudeMessage) (string, error) {
	req := claudeRequest{
		Model:     claudeModel,
		MaxTokens: defaultMaxToks,
		System:    systemPrompt,
		Messages:  messages,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", claudeVersion)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr claudeError
		_ = json.Unmarshal(respBody, &apiErr)
		c.logger.Error("claude API error",
			zap.Int("status", resp.StatusCode),
			zap.String("type", apiErr.Error.Type),
			zap.String("message", apiErr.Error.Message),
		)
		return "", fmt.Errorf("claude API error: %s", apiErr.Error.Message)
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from claude")
	}

	c.logger.Debug("claude response",
		zap.Int("input_tokens", claudeResp.Usage.InputTokens),
		zap.Int("output_tokens", claudeResp.Usage.OutputTokens),
	)

	return claudeResp.Content[0].Text, nil
}
