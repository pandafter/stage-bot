package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// BoldEvent is the Cloud-Events shaped payload Bold POSTs to our webhook.
type BoldEvent struct {
	ID              string          `json:"id"`
	Type            string          `json:"type"`
	Subject         string          `json:"subject"`
	Source          string          `json:"source"`
	SpecVersion     string          `json:"spec_version"`
	Time            int64           `json:"time"`
	DataContentType string          `json:"datacontenttype"`
	Data            json.RawMessage `json:"data"`
}

type boldData struct {
	PaymentID  string `json:"payment_id"`
	MerchantID string `json:"merchant_id"`
	CreatedAt  string `json:"created_at"`
	Amount     struct {
		Total       int64  `json:"total"`
		Currency    string `json:"currency"`
		TaxesAmount int64  `json:"taxes_amount"`
		TipAmount   int64  `json:"tip_amount"`
	} `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Card          struct {
		Capturer  string `json:"capturer"`
		Franchise string `json:"franchise"`
	} `json:"card"`
	Metadata struct {
		Reference string `json:"reference"`
	} `json:"metadata"`
}

// BoldWebhook verifies the HMAC signature, parses the Bold payload, updates
// the matching inscripcion status and pings Telegram. Always returns 200 to
// prevent Bold retries.
func (h *Handler) BoldWebhook(c *fiber.Ctx) error {
	body := c.Body()

	if h.cfg.BoldWebhookSecret == "" {
		h.logger.Warn("bold webhook received but BOLD_WEBHOOK_SECRET not set — accepting without verification")
	} else {
		got := c.Get("x-bold-signature")
		if !verifyBoldSignature(body, got, h.cfg.BoldWebhookSecret) {
			h.logger.Warn("bold webhook signature mismatch", zap.String("got", got))
			return c.Status(fiber.StatusUnauthorized).SendString("invalid signature")
		}
	}

	var ev BoldEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		h.logger.Error("bold webhook decode failed", zap.Error(err), zap.String("body", string(body)))
		return c.Status(fiber.StatusOK).SendString("invalid json")
	}

	var data boldData
	_ = json.Unmarshal(ev.Data, &data)

	ref := strings.TrimSpace(data.Metadata.Reference)
	h.logger.Info("bold webhook received",
		zap.String("event_id", ev.ID),
		zap.String("type", ev.Type),
		zap.String("transaction_id", ev.Subject),
		zap.String("reference", ref),
		zap.Int64("amount", data.Amount.Total))

	if ref == "" {
		return c.SendStatus(fiber.StatusOK)
	}

	newStatus := mapBoldStatus(ev.Type)
	if newStatus == "" {
		return c.SendStatus(fiber.StatusOK)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Idempotencia: si el estado ya es el target, omitimos notificar para no
	// duplicar mensajes en Telegram cuando Bold reenvía el evento.
	existing, _ := h.repo.GetByID(ctx, ref)
	alreadyApplied := existing != nil && existing.Status == newStatus

	if err := h.repo.UpdateStatus(ctx, ref, newStatus); err != nil {
		h.logger.Error("update inscripcion status from bold", zap.Error(err), zap.String("ref", ref))
	}

	if !alreadyApplied {
		method := data.PaymentMethod
		if data.Card.Franchise != "" {
			method = method + " (" + data.Card.Franchise + ")"
		}
		tsLabel := time.Unix(ev.Time, 0).Format("2006-01-02 15:04:05")
		if ev.Time == 0 {
			tsLabel = time.Now().Format("2006-01-02 15:04:05")
		}
		go h.telegram.NotifyBoldEvent(ev.Type, ev.Subject, data.PaymentID, method, tsLabel,
			newStatus, int(data.Amount.Total), existing, ref)
	}

	return c.SendStatus(fiber.StatusOK)
}

func mapBoldStatus(eventType string) string {
	switch eventType {
	case "SALE_APPROVED":
		return "pagado"
	case "SALE_REJECTED":
		return "pago rechazado"
	case "VOID_APPROVED":
		return "pago anulado"
	default:
		return ""
	}
}

// verifyBoldSignature returns true if signature matches HMAC-SHA256 of
// base64(body) keyed with secret, encoded as hex.
func verifyBoldSignature(body []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}
	encoded := base64.StdEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(encoded))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(strings.TrimSpace(signature))))
}
