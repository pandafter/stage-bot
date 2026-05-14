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

const graphAPIBase = "https://graph.instagram.com/v25.0/me/messages"

// Messenger sends messages via the Instagram Messaging API.
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

// sendRequest sends a message payload to the Instagram Messaging API.
func (m *Messenger) sendRequest(payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	url := graphAPIBase + "?access_token=" + m.token
	resp, err := m.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		m.logger.Error("messenger: API error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)),
		)
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
		"recipient":      map[string]string{"id": recipientID},
		"messaging_type": "RESPONSE",
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
