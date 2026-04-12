package messenger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	graphAPIBase   = "https://graph.instagram.com/v21.0"
	defaultTimeout = 10 * time.Second
)

// Instagram implements domain.Messenger using the Instagram Graph API.
type Instagram struct {
	pageToken  string
	accountID  string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewInstagram(pageToken string, accountID string, logger *zap.Logger) *Instagram {
	return &Instagram{
		pageToken: pageToken,
		accountID: accountID,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		logger: logger,
	}
}

// SendText sends a text message to a recipient.
func (c *Instagram) SendText(recipientID, text string) error {
	req := SendRequest{
		Recipient: Recipient{ID: recipientID},
		Message:   SendMsg{Text: text},
	}
	return c.sendMessage(req)
}

// SendAudio sends an audio attachment via a public URL.
func (c *Instagram) SendAudio(recipientID, audioURL string) error {
	req := SendRequest{
		Recipient: Recipient{ID: recipientID},
		Message: SendMsg{
			Attachment: &SendAttach{
				Type:    "audio",
				Payload: SendPayload{URL: audioURL},
			},
		},
	}
	return c.sendMessage(req)
}

// SendQuickReplies sends a message with quick reply buttons.
func (c *Instagram) SendQuickReplies(recipientID, text string, replies []QuickReply) error {
	req := SendRequest{
		Recipient: Recipient{ID: recipientID},
		Message: SendMsg{
			Text:         text,
			QuickReplies: replies,
		},
	}
	return c.sendMessage(req)
}

// SetTypingOn shows the typing indicator to the user.
func (c *Instagram) SetTypingOn(recipientID string) error {
	return c.sendAction(recipientID, "typing_on")
}

// SetTypingOff hides the typing indicator.
func (c *Instagram) SetTypingOff(recipientID string) error {
	return c.sendAction(recipientID, "typing_off")
}

// MarkSeen marks the last message as seen.
func (c *Instagram) MarkSeen(recipientID string) error {
	return c.sendAction(recipientID, "mark_seen")
}

// DownloadMedia downloads media from an Instagram media URL.
func (c *Instagram) DownloadMedia(mediaURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", mediaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create media request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.pageToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download media: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read media body: %w", err)
	}

	return data, nil
}

func (c *Instagram) sendMessage(req SendRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal send request: %w", err)
	}

	url := fmt.Sprintf("%s/me/messages?access_token=%s", graphAPIBase, c.pageToken)

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		_ = json.Unmarshal(respBody, &apiErr)
		c.logger.Error("instagram API error",
			zap.Int("status", resp.StatusCode),
			zap.String("error", apiErr.Error.Message),
			zap.Int("code", apiErr.Error.Code),
		)
		return fmt.Errorf("instagram API error %d: %s", apiErr.Error.Code, apiErr.Error.Message)
	}

	var sendResp SendResponse
	_ = json.Unmarshal(respBody, &sendResp)

	c.logger.Debug("message sent",
		zap.String("recipient_id", sendResp.RecipientID),
		zap.String("message_id", sendResp.MessageID),
	)

	return nil
}

func (c *Instagram) sendAction(recipientID, action string) error {
	req := SenderActionRequest{
		Recipient:    Recipient{ID: recipientID},
		SenderAction: action,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal action request: %w", err)
	}

	url := fmt.Sprintf("%s/me/messages?access_token=%s", graphAPIBase, c.pageToken)

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send action: %w", err)
	}
	defer resp.Body.Close()

	return nil
}
