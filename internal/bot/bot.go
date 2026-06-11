// Package bot implements a simple Instagram DM auto-responder.
//
// Flow:
//  1. User sends their first DM to the Page's Instagram account.
//  2. Meta's webhook delivers the message to POST /webhook/instagram.
//  3. The bot replies with the greeting message.
//  4. Subsequent messages from the same sender are silently ignored.
//  5. A background ticker sends the same greeting every 3 days as follow-up
//     (up to maxFollowups times, staying within Meta's 7-day messaging window).
package bot

import (
	"context"
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
	"github.com/kart-academy/instagram-bot/internal/storage"
)

const (
	replyText    = "Hola, ¿cómo estás?"
	followupEvery = 3 * 24 * time.Hour // 3 days
	maxFollowups  = 2                  // day-3 and day-6 follow-ups (within Meta's 7-day window)
	tickerInterval = 1 * time.Hour     // how often the ticker checks for due follow-ups
)

// Handler exposes the webhook routes for the Instagram Messaging API.
type Handler struct {
	cfg    *config.Config
	msgr   *Messenger
	leads  *storage.BotLeadsRepo
	logger *zap.Logger
	quit   chan struct{}
}

// NewHandler creates a bot handler. If required config is missing, returns nil.
func NewHandler(cfg *config.Config, leads *storage.BotLeadsRepo, logger *zap.Logger) *Handler {
	if cfg.PageAccessToken == "" || cfg.IGAccountID == "" {
		logger.Info("bot: PAGE_ACCESS_TOKEN or INSTAGRAM_ACCOUNT_ID not set — bot disabled")
		return nil
	}
	h := &Handler{
		cfg:    cfg,
		msgr:   NewMessenger(cfg.PageAccessToken, logger),
		leads:  leads,
		logger: logger,
		quit:   make(chan struct{}),
	}
	go h.followupTicker()
	return h
}

// Stop shuts down the background follow-up ticker.
func (h *Handler) Stop() { close(h.quit) }

// ── Webhook verification (GET) ──────────────────────────────────

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

func (h *Handler) Receive(c *fiber.Ctx) error {
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

	// Don't respond to our own messages.
	if senderID == h.cfg.IGAccountID {
		return
	}

	h.logger.Info("bot: DM received",
		zap.String("sender", senderID),
		zap.String("text", msg.Message.Text),
	)

	// If TEST_SENDER_ID is set, only process messages from that sender.
	if h.cfg.TestSenderID != "" && senderID != h.cfg.TestSenderID {
		h.logger.Info("bot: skipping — not TEST_SENDER_ID", zap.String("sender", senderID))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Only reply to the first-ever message; ignore all subsequent ones.
	known, err := h.leads.IsKnown(ctx, senderID)
	if err != nil {
		h.logger.Error("bot: leads.IsKnown", zap.Error(err))
		return
	}
	if known {
		h.logger.Debug("bot: sender already known — ignoring message", zap.String("sender", senderID))
		return
	}

	// First message — reply and persist the lead.
	if err := h.msgr.SendText(senderID, replyText); err != nil {
		h.logger.Error("bot: send first reply failed", zap.String("sender", senderID), zap.Error(err))
		return
	}
	if err := h.leads.Upsert(ctx, senderID); err != nil {
		h.logger.Error("bot: leads.Upsert", zap.Error(err))
	}
	h.logger.Info("bot: first reply sent", zap.String("sender", senderID))
}

// ── Follow-up ticker ───────────────────────────────────────────

// followupTicker runs in a goroutine and sends the greeting message to leads
// that haven't been contacted in the last 3 days (up to maxFollowups times).
func (h *Handler) followupTicker() {
	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.sendFollowups()
		case <-h.quit:
			return
		}
	}
}

func (h *Handler) sendFollowups() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	leads, err := h.leads.DueForFollowup(ctx, followupEvery, maxFollowups)
	if err != nil {
		h.logger.Error("bot: followup query failed", zap.Error(err))
		return
	}
	for _, lead := range leads {
		if err := h.msgr.SendText(lead.SenderID, replyText); err != nil {
			h.logger.Error("bot: followup send failed",
				zap.String("sender", lead.SenderID),
				zap.Error(err),
			)
			continue
		}
		if err := h.leads.RecordFollowup(ctx, lead.SenderID); err != nil {
			h.logger.Error("bot: followup record failed",
				zap.String("sender", lead.SenderID),
				zap.Error(err),
			)
		}
		h.logger.Info("bot: follow-up sent",
			zap.String("sender", lead.SenderID),
			zap.Int("followup_count", lead.FollowupCount+1),
		)
	}
}

// ── Helpers ────────────────────────────────────────────────────

func (h *Handler) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}
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
