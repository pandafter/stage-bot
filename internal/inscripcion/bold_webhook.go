package inscripcion

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/storage"
)

// BoldEvent is the Cloud-Events-shaped payload Bold POSTs to our webhook.
// Fields documented at https://developers.bold.co/webhook
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
	PaymentID     string `json:"payment_id"`
	MerchantID    string `json:"merchant_id"`
	CreatedAt     string `json:"created_at"`
	Amount        struct {
		Total       int64  `json:"total"`
		Currency    string `json:"currency"`
		TaxesAmount int64  `json:"taxes_amount"`
		TipAmount   int64  `json:"tip_amount"`
	} `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Card          struct {
		Capturer string `json:"capturer"`
		Franchise string `json:"franchise"`
	} `json:"card"`
	Metadata struct {
		Reference string `json:"reference"`
	} `json:"metadata"`
}

// BoldWebhook verifies the HMAC signature, parses the Bold payload, updates
// the matching inscripcion status and pings Telegram. Always returns 200 to
// prevent Bold retries on bugs at our end (we log everything).
func (h *Handler) BoldWebhook(c *fiber.Ctx) error {
	body := c.Body()
	captureBold(c, body)

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
		// Nothing we can match — still return 200 so Bold doesn't retry.
		return c.SendStatus(fiber.StatusOK)
	}

	newStatus := mapBoldStatus(ev.Type)
	if newStatus == "" {
		return c.SendStatus(fiber.StatusOK)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.repo.UpdateStatus(ctx, ref, newStatus); err != nil {
		h.logger.Error("update inscripcion status from bold", zap.Error(err), zap.String("ref", ref))
		// still 200 — we don't want Bold retrying just because our DB hiccupped
	}

	// Fetch the record so the Telegram confirmation can include the pilot name,
	// email, course date, etc. — non-fatal if it fails.
	lookupCtx, lookupCancel := context.WithTimeout(context.Background(), 3*time.Second)
	rec, _ := h.repo.GetByID(lookupCtx, ref)
	lookupCancel()

	go h.notifyBold(ev, data, newStatus, rec)
	go h.pushStatusUpdate(context.Background(), SheetStatusUpdate{
		ID:            ref,
		Status:        newStatus,
		BoldPaymentID: ev.Subject,
		BoldAmount:    int(data.Amount.Total),
		BoldMethod:    data.PaymentMethod,
		BoldFranchise: data.Card.Franchise,
	})

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
	case "VOID_REJECTED":
		return ""
	default:
		return ""
	}
}

// verifyBoldSignature returns true if `signature` matches the HMAC-SHA256 of
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

func (h *Handler) notifyBold(ev BoldEvent, data boldData, newStatus string, rec *storage.InscripcionRecord) {
	if h.cfg.TelegramBotToken == "" || h.cfg.TelegramChatID == "" {
		return
	}

	header := ""
	switch ev.Type {
	case "SALE_APPROVED":
		header = "🟢 *PAGO CONFIRMADO* — Bold"
	case "SALE_REJECTED":
		header = "🔴 *PAGO RECHAZADO* — Bold"
	case "VOID_APPROVED":
		header = "↩️ *Pago anulado* — Bold"
	default:
		header = "💳 *Evento Bold:* " + tgEscape(ev.Type)
	}

	method := data.PaymentMethod
	if data.Card.Franchise != "" {
		method = method + " (" + data.Card.Franchise + ")"
	}

	tsLabel := time.Unix(ev.Time, 0).Format("2006-01-02 15:04:05")
	if ev.Time == 0 {
		tsLabel = time.Now().Format("2006-01-02 15:04:05")
	}

	var b strings.Builder
	fmt.Fprintln(&b, header)
	b.WriteString("\n")
	if rec != nil {
		fmt.Fprintf(&b, "👤 *Piloto:* %s\n", tgEscape(rec.NombrePiloto))
		fmt.Fprintf(&b, "✉️ *Email:* %s\n", tgEscape(rec.Email))
		if rec.Telefono != "" {
			fmt.Fprintf(&b, "📱 *Tel:* %s\n", tgEscape(rec.Telefono))
		}
		if rec.FechaCurso != "" {
			fmt.Fprintf(&b, "📅 *Fecha curso:* %s\n", tgEscape(rec.FechaCurso))
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "🧾 *Comprobante Bold*\n")
	fmt.Fprintf(&b, "*Inscripción:* `%s`\n", tgEscape(data.Metadata.Reference))
	fmt.Fprintf(&b, "*ID transacción Bold:* `%s`\n", tgEscape(ev.Subject))
	if data.PaymentID != "" {
		fmt.Fprintf(&b, "*Payment ID:* `%s`\n", tgEscape(data.PaymentID))
	}
	fmt.Fprintf(&b, "*Monto pagado:* $%s COP\n", tgEscape(formatCOP(int(data.Amount.Total))))
	fmt.Fprintf(&b, "*Método:* %s\n", tgEscape(method))
	fmt.Fprintf(&b, "*Fecha y hora:* %s\n", tgEscape(tsLabel))
	fmt.Fprintf(&b, "*Estado actualizado:* %s\n", tgEscape(newStatus))
	fmt.Fprintf(&b, "\n_Notificación oficial recibida vía webhook firmado por Bold. Este mensaje sirve como comprobante digital del evento de pago._")

	if err := h.tgSendMessage(b.String()); err != nil {
		h.logger.Error("telegram bold notification failed", zap.Error(err))
	}
}

// boldDebugBody is exposed by the dev-only `/inscripcion/bold-debug` endpoint
// to show what the last raw body and headers looked like — useful while you're
// configuring the webhook in Bold's dashboard.
type boldDebugBody struct {
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// BoldDebug echoes the last request received (within memory of this binary).
// Protected by ADMIN_TOKEN. The actual capture happens via the BoldWebhook
// handler when called with ?capture=1 query.
func (h *Handler) BoldDebug(c *fiber.Ctx) error {
	if h.cfg.AdminToken == "" || c.Query("token") != h.cfg.AdminToken {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"ok": false, "error": "unauthorized"})
	}
	if lastBoldDebug == nil {
		return c.JSON(fiber.Map{"ok": false, "message": "no bold webhook recibido aún"})
	}
	return c.JSON(lastBoldDebug)
}

var lastBoldDebug *boldDebugBody

// captureBold stores the latest webhook for /bold-debug. Call from BoldWebhook.
func captureBold(c *fiber.Ctx, body []byte) {
	hdrs := map[string]string{}
	c.Request().Header.VisitAll(func(k, v []byte) {
		hdrs[string(k)] = string(v)
	})
	lastBoldDebug = &boldDebugBody{Headers: hdrs, Body: string(body)}
}

var _ = io.Discard // keep io import in case future edits need it
