package inscripcion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/storage"
)

// SheetsRow is the JSON payload posted to the Apps Script web app.
// Field order is also the row order the script will use for appendRow().
type SheetsRow struct {
	Token            string `json:"token"`
	Timestamp        string `json:"timestamp"`
	ID               string `json:"id"`
	Status           string `json:"status"`
	Modalidad        string `json:"modalidad"`
	Plan             string `json:"plan"`
	MetodoPago       string `json:"metodo_pago"`
	MontoCOP         int    `json:"monto_cop"`
	FechaCurso       string `json:"fecha_curso"`
	NombrePiloto     string `json:"nombre_piloto"`
	Edad             int    `json:"edad"`
	TipoDocumento    string `json:"tipo_documento"`
	NumeroDocumento  string `json:"numero_documento"`
	Telefono         string `json:"telefono"`
	Email            string `json:"email"`
	Ciudad           string `json:"ciudad"`
	EPS              string `json:"eps"`
	GrupoSanguineo   string `json:"grupo_sanguineo"`
	InstagramUser    string `json:"instagram_user"`
	FamiliarNombre   string `json:"familiar_nombre"`
	FamiliarTelefono string `json:"familiar_telefono"`
	ComprobanteURL   string `json:"comprobante_url"`
	BoldCheckoutURL  string `json:"bold_checkout_url"`
}

// pushToSheets POSTs a row to the configured Google Apps Script web app. No-op
// when SHEETS_WEBHOOK_URL is empty.
func (h *Handler) pushToSheets(ctx context.Context, rec storage.InscripcionRecord, modalidad, boldURL string) {
	if h.cfg.SheetsWebhookURL == "" {
		return
	}
	row := SheetsRow{
		Token:            h.cfg.SheetsSharedToken,
		Timestamp:        time.Now().Format(time.RFC3339),
		ID:               rec.ID,
		Status:           rec.Status,
		Modalidad:        modalidad,
		Plan:             rec.Plan,
		MetodoPago:       rec.MetodoPago,
		MontoCOP:         rec.MontoCOP,
		FechaCurso:       rec.FechaCurso,
		NombrePiloto:     rec.NombrePiloto,
		Edad:             rec.Edad,
		TipoDocumento:    rec.TipoDocumento,
		NumeroDocumento:  rec.NumeroDocumento,
		Telefono:         rec.Telefono,
		Email:            rec.Email,
		Ciudad:           rec.Ciudad,
		EPS:              rec.EPS,
		GrupoSanguineo:   rec.GrupoSanguineo,
		InstagramUser:    rec.InstagramUser,
		FamiliarNombre:   rec.FamiliarNombre,
		FamiliarTelefono: rec.FamiliarTelefono,
		ComprobanteURL:   rec.ComprobantePath,
		BoldCheckoutURL:  boldURL,
	}
	body, err := json.Marshal(row)
	if err != nil {
		h.logger.Error("sheets marshal", zap.Error(err))
		return
	}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, "POST", h.cfg.SheetsWebhookURL, bytes.NewReader(body))
	if err != nil {
		h.logger.Error("sheets request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.logger.Error("sheets post", zap.Error(err))
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		h.logger.Error("sheets non-2xx", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return
	}
	h.logger.Info("inscripcion enviada a Google Sheets", zap.String("id", rec.ID))
}

// boldDescription generates a human-readable description for the Bold link.
func boldDescription(modalidad string, edad int) string {
	switch modalidad {
	case "completo":
		return fmt.Sprintf("Curso piloto Scuderia St4ge — Pago completo (%d años)", edad)
	default:
		return fmt.Sprintf("Curso piloto Scuderia St4ge — Reserva de cupo (%d años)", edad)
	}
}

// SheetStatusUpdate is the payload posted to the Apps Script when we want to
// update an existing row instead of appending a new one. The script
// distinguishes by the `op` field.
type SheetStatusUpdate struct {
	Op             string `json:"op"`
	Token          string `json:"token"`
	ID             string `json:"id"`
	Status         string `json:"status"`
	BoldPaymentID  string `json:"bold_payment_id"`
	BoldAmount     int    `json:"bold_amount"`
	BoldMethod     string `json:"bold_method"`
	BoldFranchise  string `json:"bold_franchise"`
	BoldUpdatedAt  string `json:"bold_updated_at"`
}

// pushStatusUpdate notifies the Apps Script that a row's status changed (e.g.
// after Bold confirms the payment). No-op when the webhook URL is empty.
func (h *Handler) pushStatusUpdate(ctx context.Context, upd SheetStatusUpdate) {
	if h.cfg.SheetsWebhookURL == "" {
		return
	}
	upd.Op = "update"
	upd.Token = h.cfg.SheetsSharedToken
	if upd.BoldUpdatedAt == "" {
		upd.BoldUpdatedAt = time.Now().Format(time.RFC3339)
	}
	body, err := json.Marshal(upd)
	if err != nil {
		h.logger.Error("sheets update marshal", zap.Error(err))
		return
	}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, "POST", h.cfg.SheetsWebhookURL, bytes.NewReader(body))
	if err != nil {
		h.logger.Error("sheets update request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.logger.Error("sheets update post", zap.Error(err))
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		h.logger.Error("sheets update non-2xx", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return
	}
	h.logger.Info("inscripcion status actualizado en Sheets", zap.String("id", upd.ID), zap.String("status", upd.Status))
}
