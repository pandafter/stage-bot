package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/config"
)

func TestTokenEndpointIssuesBearerTokenAndAuthUsesIt(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	h := NewHandler(&config.Config{MCPSecret: "top-secret"}, zap.NewNop())
	h.now = func() time.Time { return now }

	app := fiber.New()
	app.Post("/mcp/token", h.Token)
	app.Get("/mcp", h.Auth, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	form := url.Values{
		"grant_type":    []string{"client_credentials"},
		"client_id":     []string{"my-client"},
		"client_secret": []string{"top-secret"},
	}
	tokenReq := httptest.NewRequest(http.MethodPost, "/mcp/token", strings.NewReader(form.Encode()))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenResp, err := app.Test(tokenReq)
	if err != nil {
		t.Fatalf("token endpoint request failed: %v", err)
	}
	if tokenResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from token endpoint, got %d", tokenResp.StatusCode)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode token response: %v", err)
	}
	if payload.AccessToken == "" {
		t.Fatal("expected non-empty access_token")
	}
	if payload.TokenType != "bearer" {
		t.Fatalf("expected token_type bearer, got %q", payload.TokenType)
	}
	if payload.ExpiresIn != 3600 {
		t.Fatalf("expected expires_in 3600, got %d", payload.ExpiresIn)
	}

	oldAuthReq := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	oldAuthReq.Header.Set("Authorization", "Bearer top-secret")
	oldAuthResp, err := app.Test(oldAuthReq)
	if err != nil {
		t.Fatalf("legacy auth request failed: %v", err)
	}
	if oldAuthResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected legacy secret bearer to be rejected with 401, got %d", oldAuthResp.StatusCode)
	}

	newAuthReq := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	newAuthReq.Header.Set("Authorization", "Bearer "+payload.AccessToken)
	newAuthResp, err := app.Test(newAuthReq)
	if err != nil {
		t.Fatalf("new auth request failed: %v", err)
	}
	if newAuthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected issued token to be accepted with 200, got %d", newAuthResp.StatusCode)
	}
}

func TestIssuedTokenExpires(t *testing.T) {
	baseTime := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	h := NewHandler(&config.Config{MCPSecret: "top-secret"}, zap.NewNop())
	h.now = func() time.Time { return baseTime }

	token, err := h.issueAccessToken("client-a", baseTime)
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}

	h.now = func() time.Time { return baseTime.Add(tokenTTL + time.Second) }
	if h.validateAccessToken(token, h.now()) {
		t.Fatal("expected expired token to be invalid")
	}
}
