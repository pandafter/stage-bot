package inscripcion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BoldPaymentMethod is one of the codes Bold accepts in the `payment_methods`
// array of the link creation API.
type BoldPaymentMethod string

const (
	BoldMethodCard        BoldPaymentMethod = "CREDIT_CARD"
	BoldMethodNequi       BoldPaymentMethod = "NEQUI"
	BoldMethodBancolombia BoldPaymentMethod = "BOTON_BANCOLOMBIA"
	BoldMethodPSE         BoldPaymentMethod = "PSE"
)

// boldLinkRequest is the shape Bold expects at POST /online/link/v1.
type boldLinkRequest struct {
	AmountType     string              `json:"amount_type"`
	Amount         boldAmount          `json:"amount"`
	Reference      string              `json:"reference,omitempty"`
	Description    string              `json:"description,omitempty"`
	PaymentMethods []BoldPaymentMethod `json:"payment_methods,omitempty"`
	CallbackURL    string              `json:"callback_url,omitempty"`
	PayerEmail     string              `json:"payer_email,omitempty"`
}

type boldAmount struct {
	Currency    string `json:"currency"`
	TotalAmount int    `json:"total_amount"`
	TipAmount   int    `json:"tip_amount"`
}

type boldLinkResponse struct {
	Payload struct {
		PaymentLink string `json:"payment_link"`
		URL         string `json:"url"`
	} `json:"payload"`
	Errors []any `json:"errors"`
}

// CreateBoldLink calls Bold's API to mint a single-use checkout URL pre-set
// with method, amount and reference. Empty `methods` means "let user choose".
func (h *Handler) CreateBoldLink(ctx context.Context, method BoldPaymentMethod, amountCOP int, reference, description, payerEmail string) (string, error) {
	if h.cfg.BoldAPIKey == "" {
		return "", fmt.Errorf("BOLD_API_KEY no configurado")
	}

	body := boldLinkRequest{
		AmountType: "CLOSE",
		Amount: boldAmount{
			Currency:    "COP",
			TotalAmount: amountCOP,
			TipAmount:   0,
		},
		Reference:   reference,
		Description: description,
		PayerEmail:  payerEmail,
	}
	if method != "" {
		body.PaymentMethods = []BoldPaymentMethod{method}
	}
	if h.cfg.PublicURL != "" {
		body.CallbackURL = h.cfg.PublicURL + "/inscripcion/callback?ref=" + reference
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "POST",
		"https://integrations.api.bold.co/online/link/v1", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "x-api-key "+h.cfg.BoldAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("bold api: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("bold api %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed boldLinkResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("bold api decode: %w (body=%s)", err, string(respBody))
	}
	if parsed.Payload.URL == "" {
		return "", fmt.Errorf("bold api returned empty url: %s", string(respBody))
	}
	return parsed.Payload.URL, nil
}
