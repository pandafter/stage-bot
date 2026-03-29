package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/config"
)

type Handler struct {
	cfg    *config.Config
	logger *zap.Logger
}

func NewHandler(cfg *config.Config, logger *zap.Logger) *Handler {
	return &Handler{
		cfg:    cfg,
		logger: logger,
	}
}

// Verify handles GET /webhook — Meta's webhook verification challenge.
func (h *Handler) Verify(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && token == h.cfg.WebhookVerifyToken {
		h.logger.Info("webhook verified successfully")
		return c.SendString(challenge)
	}

	h.logger.Warn("webhook verification failed",
		zap.String("mode", mode),
		zap.String("token_received", token),
	)
	return c.SendStatus(fiber.StatusForbidden)
}

// Receive handles POST /webhook — incoming messages from Instagram.
func (h *Handler) Receive(c *fiber.Ctx) error {
	body := c.Body()

	// Validate HMAC signature if app secret is configured
	if h.cfg.AppSecret != "" {
		signature := c.Get("X-Hub-Signature-256")
		if !validateSignature(body, signature, h.cfg.AppSecret) {
			h.logger.Warn("invalid webhook signature")
			return c.SendStatus(fiber.StatusForbidden)
		}
	}

	// Parse payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse webhook payload", zap.Error(err))
		return c.SendStatus(fiber.StatusBadRequest)
	}

	// Extract messages
	messages := ParseMessages(&payload)

	// Process in background — respond 200 immediately to avoid Instagram timeout
	if len(messages) > 0 {
		go h.processMessages(messages)
	}

	return c.SendStatus(fiber.StatusOK)
}

// processMessages handles incoming messages asynchronously.
// This is where the brain will be connected in Phase 4.
func (h *Handler) processMessages(messages []IncomingMessage) {
	for _, msg := range messages {
		h.logger.Info("message received",
			zap.String("sender_id", msg.SenderID),
			zap.String("message_id", msg.MessageID),
			zap.String("type", string(msg.Type)),
			zap.String("text", msg.Text),
			zap.String("media_url", msg.MediaURL),
			zap.Int64("timestamp", msg.Timestamp),
		)

		// TODO Phase 1: Echo response via Instagram API
		// TODO Phase 4: brain.Process(msg) → response
	}
}

// validateSignature verifies the X-Hub-Signature-256 header using HMAC SHA256.
func validateSignature(body []byte, signature string, appSecret string) bool {
	if signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}
