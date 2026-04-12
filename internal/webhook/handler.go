package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/domain"
)

type Handler struct {
	cfg        *config.Config
	messenger  domain.Messenger
	ai         domain.AIEngine
	voice      domain.VoiceService
	audioStore domain.AudioStore
	logger     *zap.Logger
}

func NewHandler(cfg *config.Config, m domain.Messenger, ai domain.AIEngine, vc domain.VoiceService, as domain.AudioStore, logger *zap.Logger) *Handler {
	return &Handler{
		cfg:        cfg,
		messenger:  m,
		ai:         ai,
		voice:      vc,
		audioStore: as,
		logger:     logger,
	}
}

// Verify handles GET /webhook — Meta's webhook verification challenge.
func (h *Handler) Verify(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && token == h.cfg.WebhookVerifyToken {
		h.logger.Info("webhook verified successfully")
		return c.SendString(challenge)
	}

	h.logger.Warn("webhook verification failed",
		zap.String("mode", mode),
		zap.String("token_received", token),
	)
	return c.SendStatus(fiber.StatusForbidden)
}

// Receive handles POST /webhook — incoming messages from Instagram.
func (h *Handler) Receive(c *fiber.Ctx) error {
	body := c.Body()

	// Validate HMAC signature (skip in development for testing)
	if h.cfg.AppSecret != "" && !h.cfg.IsDevelopment() {
		signature := c.Get("X-Hub-Signature-256")
		if !validateSignature(body, signature, h.cfg.AppSecret) {
			h.logger.Warn("invalid webhook signature")
			return c.SendStatus(fiber.StatusForbidden)
		}
	}

	// Parse payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse webhook payload", zap.Error(err))
		return c.SendStatus(fiber.StatusBadRequest)
	}

	// Extract messages
	messages := ParseMessages(&payload)

	// Process in background — respond 200 immediately to avoid Instagram timeout
	if len(messages) > 0 {
		go h.processMessages(messages)
	}

	return c.SendStatus(fiber.StatusOK)
}

// processMessages handles incoming messages asynchronously.
func (h *Handler) processMessages(messages []IncomingMessage) {
	for _, msg := range messages {
		h.logger.Info("message received",
			zap.String("sender_id", msg.SenderID),
			zap.String("message_id", msg.MessageID),
			zap.String("type", string(msg.Type)),
			zap.String("text", msg.Text),
			zap.String("media_url", msg.MediaURL),
			zap.Int64("timestamp", msg.Timestamp),
		)

		// Only respond to test user in development
		if h.cfg.TestSenderID != "" && msg.SenderID != h.cfg.TestSenderID {
			h.logger.Info("ignoring message from non-test user",
				zap.String("sender_id", msg.SenderID),
			)
			continue
		}

		h.messenger.MarkSeen(msg.SenderID)
		h.messenger.SetTypingOn(msg.SenderID)

		// Determine if incoming message is audio
		respondWithAudio := false
		var inputText string

		switch msg.Type {
		case MessageTypeAudio:
			// Audio in -> transcribe -> respond with audio
			if h.voice == nil || !h.voice.Enabled() || msg.MediaURL == "" {
				continue
			}
			audioData, err := h.messenger.DownloadMedia(msg.MediaURL)
			if err != nil {
				h.logger.Error("failed to download audio", zap.Error(err))
				continue
			}

			transcription, err := h.voice.SpeechToText(audioData)
			if err != nil {
				h.logger.Error("STT failed", zap.Error(err))
				continue
			}

			h.logger.Info("audio transcribed",
				zap.String("sender_id", msg.SenderID),
				zap.String("transcription", transcription),
			)

			inputText = transcription
			respondWithAudio = true

		case MessageTypeText:
			if msg.Text == "" {
				continue
			}
			inputText = msg.Text
			// If user explicitly asks for audio response
			if wantsAudio(msg.Text) {
				respondWithAudio = true
			}

		default:
			continue
		}

		// Generate response with AI
		reply, err := h.ai.Process(msg.SenderID, inputText)
		if err != nil {
			h.logger.Error("brain processing failed",
				zap.String("sender_id", msg.SenderID),
				zap.Error(err),
			)
			continue
		}

		// Send response
		if respondWithAudio && h.voice != nil && h.voice.Enabled() {
			if err := h.sendAudioReply(msg.SenderID, reply); err != nil {
				h.logger.Error("failed to send audio reply, falling back to text",
					zap.String("sender_id", msg.SenderID),
					zap.Error(err),
				)
				// Fallback to text if audio fails
				h.messenger.SendText(msg.SenderID, reply)
			}
		} else {
			if err := h.messenger.SendText(msg.SenderID, reply); err != nil {
				h.logger.Error("failed to send reply",
					zap.String("sender_id", msg.SenderID),
					zap.Error(err),
				)
			}
		}

		h.logger.Info("reply sent",
			zap.String("sender_id", msg.SenderID),
			zap.String("reply", reply),
			zap.Bool("audio", respondWithAudio),
		)
	}
}

// sendAudioReply converts text to speech and sends it as an audio message via public URL.
func (h *Handler) sendAudioReply(recipientID, text string) error {
	audioData, err := h.voice.TextToSpeech(text)
	if err != nil {
		return fmt.Errorf("TTS: %w", err)
	}

	audioID := h.audioStore.Put(audioData)
	audioURL := fmt.Sprintf("%s/audio/%s", h.cfg.PublicURL, audioID)

	h.logger.Debug("audio URL generated", zap.String("url", audioURL), zap.Int("bytes", len(audioData)))

	return h.messenger.SendAudio(recipientID, audioURL)
}

// wantsAudio detects if the user is asking for an audio/voice response.
func wantsAudio(text string) bool {
	lower := strings.ToLower(text)
	keywords := []string{
		"audio", "nota de voz", "voz", "voice", "escuchar",
		"háblame", "hablame", "dime con voz", "manda audio",
		"envía audio", "envia audio", "grabame", "grábame",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// validateSignature verifies the X-Hub-Signature-256 header using HMAC SHA256.
func validateSignature(body []byte, signature string, appSecret string) bool {
	if signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}
