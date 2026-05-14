package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const graphAPIBase = "https://graph.facebook.com/v21.0/me/messages"

// Messenger sends messages via the Instagram Messaging API (Messenger Platform).
type Messenger struct {
	token  string
	client *http.Client
	logger *zap.Logger
}

func NewMessenger(pageAccessToken string, logger *zap.Logger) *Messenger {
	return &Messenger{
		token: pageAccessToken,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		logger: logger,
	}
}

func (m *Messenger) sendRequest(payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, graphAPIBase, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.token)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("messenger: API error status=%d body=%s", resp.StatusCode, string(respBody))
		m.logger.Error(errMsg)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SendText sends a plain text message to the recipient.
func (m *Messenger) SendText(recipientID, text string) error {
	payload := map[string]any{
		"recipient":      map[string]string{"id": recipientID},
		"messaging_type": "RESPONSE",
		"message":        map[string]string{"text": text},
	}
	return m.sendRequest(payload)
}

// SendAudio sends an audio attachment (by URL) to the recipient.
func (m *Messenger) SendAudio(recipientID, audioURL string) error {
	payload := map[string]any{
		"recipient": map[string]string{"id": recipientID},
		"message": map[string]any{
			"attachment": map[string]any{
				"type": "audio",
				"payload": map[string]any{
					"url":         audioURL,
					"is_reusable": true,
				},
			},
		},
	}
	return m.sendRequest(payload)
}
