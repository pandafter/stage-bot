// Package bot implements a simple Instagram DM auto-responder.
//
// Flow:
//  1. User sends a DM to the Page's Instagram account.
//  2. Meta's webhook delivers the message to POST /webhook/instagram.
//  3. The bot replies with a pre-recorded audio message from the director.
//  4. Immediately after, the bot sends a button linking to the registration page.
//  5. The lead is closed — the director can follow up manually later.
//
// In development mode (ENV=development), only messages from TEST_SENDER_ID
// are processed. In production, all messages are processed.
package bot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/config"
)

// Handler exposes the webhook routes for the Instagram Messaging API.
type Handler struct {
	cfg    *config.Config
	msgr   *Messenger
	logger *zap.Logger

	// respondedTo tracks sender IDs we've already auto-replied to in this
	// process lifetime so we don't spam on every message. Resets on restart.
	respondedTo map[string]time.Time
}

// NewHandler creates a bot handler. If required config is missing, returns nil
// and the server should skip registering the bot routes.
func NewHandler(cfg *config.Config, logger *zap.Logger) *Handler {
	if cfg.PageAccessToken == "" || cfg.IGAccountID == "" {
		logger.Info("bot: PAGE_ACCESS_TOKEN or INSTAGRAM_ACCOUNT_ID not set — bot disabled")
		return nil
	}
	return &Handler{
		cfg:         cfg,
		msgr:        NewMessenger(cfg.PageAccessToken, logger),
		logger:      logger,
		respondedTo: make(map[string]time.Time),
	}
}

// ── Webhook verification (GET) ──────────────────────────────────

// Verify handles the Meta webhook verification challenge.
// Meta sends: GET /webhook/instagram?hub.mode=subscribe&hub.verify_token=<TOKEN>&hub.challenge=<CHALLENGE>
func (h *Handler) Verify(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && token == h.cfg.WebhookVerifyToken {
		h.logger.Info("bot: webhook verified")
		return c.SendString(challenge)
	}

	h.logger.Warn("bot: webhook verification failed",
		zap.String("mode", mode),
		zap.String("token_match", fmt.Sprintf("%v", token == h.cfg.WebhookVerifyToken)),
	)
	return c.SendStatus(fiber.StatusForbidden)
}

// ── Webhook handler (POST) ──────────────────────────────────────

// WebhookPayload is the top-level structure Meta sends.
type WebhookPayload struct {
	Object string  `json:"object,Object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID        string      `json:"id"`
	Time      int64       `json:"time"`
	Messaging []Messaging `json:"messaging"`
}

type Messaging struct {
	Sender    IDField  `json:"sender"`
	Recipient IDField  `json:"recipient"`
	Timestamp int64    `json:"timestamp"`
	Message   *Message `json:"message,omitempty"`
}

type IDField struct {
	ID string `json:"id"`
}

type Message struct {
	MID  string `json:"mid"`
	Text string `json:"text"`
}

// Receive handles incoming webhook events from Meta.
func (h *Handler) Receive(c *fiber.Ctx) error {
	// Verify signature if APP_SECRET is set
	if h.cfg.AppSecret != "" {
		sig := c.Get("X-Hub-Signature-256")
		if !h.verifySignature(c.Body(), sig) {
			h.logger.Warn("bot: invalid webhook signature")
			return c.SendStatus(fiber.StatusUnauthorized)
		}
	}

	var payload WebhookPayload
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		h.logger.Error("bot: invalid payload", zap.Error(err))
		return c.SendStatus(fiber.StatusBadRequest)
	}

	// Must respond 200 quickly to avoid Meta retries
	// Process in background
	go h.processPayload(payload)

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) processPayload(payload WebhookPayload) {
	if payload.Object != "instagram" {
		return
	}

	for _, entry := range payload.Entry {
		for _, msg := range entry.Messaging {
			if msg.Message == nil {
				continue
			}
			h.handleMessage(msg)
		}
	}
}

func (h *Handler) handleMessage(msg Messaging) {
	senderID := msg.Sender.ID

	// Don't respond to our own messages
	if senderID == h.cfg.IGAccountID {
		return
	}

	h.logger.Info("bot: DM received",
		zap.String("sender", senderID),
		zap.String("text", msg.Message.Text),
	)

	// If TEST_SENDER_ID is set, only reply to that sender (regardless of env)
	if h.cfg.TestSenderID != "" && senderID != h.cfg.TestSenderID {
		h.logger.Info("bot: skipping reply — not TEST_SENDER_ID",
			zap.String("sender", senderID),
			zap.String("test_sender", h.cfg.TestSenderID),
		)
		return
	}

	// Check if we already auto-replied to this sender (within 24h)
	if lastReply, ok := h.respondedTo[senderID]; ok {
		if time.Since(lastReply) < 24*time.Hour {
			h.logger.Debug("bot: already replied to sender within 24h",
				zap.String("sender", senderID),
			)
			return
		}
	}

	h.logger.Info("bot: sending auto-reply",
		zap.String("sender", senderID),
	)

	registrationURL := h.cfg.PublicURL + "/inscripcion"
	if h.cfg.PublicURL == "" {
		registrationURL = "https://scuderiastage.com/inscripcion"
	}
	replyText := fmt.Sprintf(
		"Te comparto el link para que puedas ver más información del curso y puedas inscribirte: %s",
		registrationURL,
	)

	// Send audio and text concurrently to minimise reply latency.
	type result struct{ err error }
	audioCh := make(chan result, 1)
	textCh := make(chan result, 1)

	if h.cfg.BotAudioURL != "" {
		go func() {
			audioCh <- result{h.msgr.SendAudio(senderID, h.cfg.BotAudioURL)}
		}()
	} else {
		audioCh <- result{nil}
	}

	go func() {
		textCh <- result{h.msgr.SendText(senderID, replyText)}
	}()

	if r := <-audioCh; r.err != nil {
		h.logger.Error("bot: failed to send audio",
			zap.String("sender", senderID),
			zap.Error(r.err),
		)
	}
	if r := <-textCh; r.err != nil {
		h.logger.Error("bot: failed to send text",
			zap.String("sender", senderID),
			zap.Error(r.err),
		)
		return
	}

	h.respondedTo[senderID] = time.Now()
}

// verifySignature checks the X-Hub-Signature-256 header against the body.
func (h *Handler) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	// Signature format: "sha256=<hex>"
	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.cfg.AppSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(parts[1]))
}

// RawBody is a middleware that reads the body into c.Locals for signature verification.
func RawBody(c *fiber.Ctx) error {
	body, err := io.ReadAll(c.Request().BodyStream())
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	c.Request().SetBody(body)
	return c.Next()
}
