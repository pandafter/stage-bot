package api

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/storage"
)

// TelegramClient sends notifications to the director's chat.
type TelegramClient struct {
	token  string
	chatID string
	logger *zap.Logger
}

func NewTelegramClient(token, chatID string, logger *zap.Logger) *TelegramClient {
	return &TelegramClient{token: token, chatID: chatID, logger: logger}
}

func (t *TelegramClient) Configured() bool {
	return t.token != "" && t.chatID != ""
}

func (t *TelegramClient) sendMessage(text string) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)
	body := url.Values{}
	body.Set("chat_id", t.chatID)
	body.Set("text", text)
	body.Set("parse_mode", "Markdown")
	resp, err := http.PostForm(endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram %d: %s", resp.StatusCode, string(out))
	}
	return nil
}

func (t *TelegramClient) sendPhoto(path, caption string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("chat_id", t.chatID)
	_ = mw.WriteField("caption", caption)
	_ = mw.WriteField("parse_mode", "Markdown")
	fw, err := mw.CreateFormFile("photo", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return err
	}
	mw.Close()

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", t.token)
	req, err := http.NewRequest("POST", endpoint, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram %d: %s", resp.StatusCode, string(out))
	}
	return nil
}

// NotifyNewInscripcion sends the director a markdown summary of a new sign-up.
func (t *TelegramClient) NotifyNewInscripcion(rec storage.InscripcionRecord, modalidad Modalidad, method PaymentMethod, checkoutURL string) {
	if !t.Configured() {
		t.logger.Info("inscripcion received (no Telegram configured)",
			zap.String("id", rec.ID),
			zap.String("piloto", rec.NombrePiloto),
			zap.String("email", rec.Email),
			zap.String("modalidad", modalidad.ID),
			zap.Int("monto", rec.MontoCOP))
		return
	}

	caption := buildTelegramMessage(rec, modalidad, method, checkoutURL)

	if rec.ComprobantePath != "" {
		if err := t.sendPhoto(rec.ComprobantePath, caption); err != nil {
			t.logger.Error("telegram sendPhoto failed, falling back to text", zap.Error(err))
			if err := t.sendMessage(caption); err != nil {
				t.logger.Error("telegram sendMessage fallback failed", zap.Error(err))
			}
		}
		return
	}

	if err := t.sendMessage(caption); err != nil {
		t.logger.Error("telegram sendMessage failed", zap.Error(err))
	}
}

// NotifyBoldEvent informs the director of a Bold webhook event (approved/rejected/voided).
func (t *TelegramClient) NotifyBoldEvent(eventType, transactionID, paymentID, methodLabel, tsLabel, newStatus string, amount int, rec *storage.InscripcionRecord, reference string) {
	if !t.Configured() {
		return
	}

	header := ""
	switch eventType {
	case "SALE_APPROVED":
		header = "🟢 *PAGO CONFIRMADO* — Bold"
	case "SALE_REJECTED":
		header = "🔴 *PAGO RECHAZADO* — Bold"
	case "VOID_APPROVED":
		header = "↩️ *Pago anulado* — Bold"
	default:
		header = "💳 *Evento Bold:* " + tgEscape(eventType)
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
	fmt.Fprintf(&b, "*Inscripción:* `%s`\n", tgEscape(reference))
	fmt.Fprintf(&b, "*ID transacción Bold:* `%s`\n", tgEscape(transactionID))
	if paymentID != "" {
		fmt.Fprintf(&b, "*Payment ID:* `%s`\n", tgEscape(paymentID))
	}
	fmt.Fprintf(&b, "*Monto pagado:* $%s COP\n", tgEscape(formatCOP(amount)))
	fmt.Fprintf(&b, "*Método:* %s\n", tgEscape(methodLabel))
	fmt.Fprintf(&b, "*Fecha y hora:* %s\n", tgEscape(tsLabel))
	fmt.Fprintf(&b, "*Estado actualizado:* %s\n", tgEscape(newStatus))
	fmt.Fprintf(&b, "\n_Notificación oficial recibida vía webhook firmado por Bold._")

	if err := t.sendMessage(b.String()); err != nil {
		t.logger.Error("telegram bold notification failed", zap.Error(err))
	}
}

func buildTelegramMessage(rec storage.InscripcionRecord, modalidad Modalidad, method PaymentMethod, checkoutURL string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🏁 *Nueva inscripción* — %s\n\n", tgEscape(modalidad.Label))
	fmt.Fprintf(&b, "*ID:* `%s`\n", tgEscape(rec.ID))
	fmt.Fprintf(&b, "*Estado:* %s\n", tgEscape(rec.Status))
	fmt.Fprintf(&b, "*Monto:* $%s COP\n", tgEscape(formatCOP(rec.MontoCOP)))
	fmt.Fprintf(&b, "*Pago:* %s\n", tgEscape(method.Label))
	fmt.Fprintf(&b, "*Fecha curso:* %s\n", tgEscape(rec.FechaCurso))
	if checkoutURL != "" {
		fmt.Fprintf(&b, "*Link Bold:* %s\n", checkoutURL)
	}
	b.WriteString("\n👤 *Piloto*\n")
	fmt.Fprintf(&b, "%s, %d años\n", tgEscape(rec.NombrePiloto), rec.Edad)
	fmt.Fprintf(&b, "%s %s\n", tgEscape(rec.TipoDocumento), tgEscape(rec.NumeroDocumento))
	fmt.Fprintf(&b, "📱 %s\n", tgEscape(rec.Telefono))
	fmt.Fprintf(&b, "✉️ %s\n", tgEscape(rec.Email))
	if rec.Ciudad != "" {
		fmt.Fprintf(&b, "🏙 %s\n", tgEscape(rec.Ciudad))
	}
	if rec.InstagramUser != "" {
		fmt.Fprintf(&b, "📸 IG: %s\n", tgEscape(rec.InstagramUser))
	}
	fmt.Fprintf(&b, "\n🩺 EPS: %s · Sangre: %s\n", tgEscape(rec.EPS), tgEscape(rec.GrupoSanguineo))
	fmt.Fprintf(&b, "🆘 Emergencia: %s — %s", tgEscape(rec.FamiliarNombre), tgEscape(rec.FamiliarTelefono))
	return b.String()
}

func tgEscape(s string) string {
	r := strings.NewReplacer("_", `\_`, "*", `\*`, "[", `\[`, "`", `\`+"`")
	return r.Replace(s)
}

func formatCOP(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, r := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, r)
	}
	return string(out)
}
