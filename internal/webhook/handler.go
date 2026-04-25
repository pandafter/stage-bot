package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/domain"
	"github.com/kart-academy/instagram-bot/internal/queue"
)

const graphAPIBase = "https://graph.instagram.com/v21.0"
const graphAPITimeout = 5 * time.Second

type Handler struct {
	cfg        *config.Config
	messenger  domain.Messenger
	ai         domain.AIEngine
	voice      domain.VoiceService
	audioStore domain.AudioStore
	queue      queue.Queue
	graphHTTP  *http.Client
	keyword    string
	logger     *zap.Logger
}

func NewHandler(
	cfg *config.Config,
	m domain.Messenger,
	ai domain.AIEngine,
	vc domain.VoiceService,
	as domain.AudioStore,
	q queue.Queue,
	logger *zap.Logger,
) *Handler {
	keyword := strings.TrimSpace(cfg.CommentTriggerKeyword)
	if keyword == "" {
		keyword = "piloto"
	}

	return &Handler{
		cfg:        cfg,
		messenger:  m,
		ai:         ai,
		voice:      vc,
		audioStore: as,
		queue:      q,
		graphHTTP: &http.Client{
			Timeout: graphAPITimeout,
			Transport: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		keyword: keyword,
		logger:  logger,
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

// Receive handles POST /webhook — validates signature, parses messages,
// and enqueues each one for asynchronous processing by the worker pool.
func (h *Handler) Receive(c *fiber.Ctx) error {
	body := c.Body()

	if h.cfg.AppSecret != "" && !h.cfg.IsDevelopment() {
		signature := c.Get("X-Hub-Signature-256")
		if !validateSignature(body, signature, h.cfg.AppSecret) {
			expected := computeSignature(body, h.cfg.AppSecret)
			h.logger.Warn("invalid webhook signature",
				zap.Int("secret_len", len(h.cfg.AppSecret)),
				zap.Int("body_len", len(body)),
				zap.String("received_sig_prefix", safePrefix(signature, 14)),
				zap.String("computed_sig_prefix", safePrefix(expected, 14)),
				zap.String("client_ua", c.Get("User-Agent")),
				zap.String("src_ip", c.IP()),
			)
			return c.SendStatus(fiber.StatusForbidden)
		}
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse webhook payload", zap.Error(err))
		return c.SendStatus(fiber.StatusBadRequest)
	}

	messages := ParseMessages(&payload)
	if HasCommentChanges(&payload) {
		latestMediaID, err := h.fetchLatestMediaID(c.Context())
		if err != nil {
			h.logger.Warn("failed to resolve latest media for comment trigger", zap.Error(err))
		} else {
			commentTriggers := ParseCommentTriggers(&payload, latestMediaID, h.keyword)
			messages = append(messages, commentTriggers...)
		}
	}

	for _, msg := range messages {
		h.logger.Info("message received",
			zap.String("sender_id", msg.SenderID),
			zap.String("message_id", msg.MessageID),
			zap.String("type", string(msg.Type)),
			zap.String("text", msg.Text),
		)

		// Dev-mode filter: if TEST_SENDER_ID set, skip others before enqueuing.
		if h.cfg.TestSenderID != "" && msg.SenderID != h.cfg.TestSenderID {
			h.logger.Info("ignoring message (TEST_SENDER_ID filter)",
				zap.String("sender_id", msg.SenderID))
			continue
		}

		if h.queue == nil {
			// Fallback: old fire-and-forget if queue not wired (dev safety net).
			go h.process(context.Background(), msg)
			continue
		}

		jsonBytes, err := json.Marshal(msg)
		if err != nil {
			h.logger.Error("failed to marshal message for queue", zap.Error(err))
			continue
		}

		enqCtx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		jobID, err := h.queue.Enqueue(enqCtx, msg.SenderID, jsonBytes)
		cancel()
		if err != nil {
			h.logger.Error("enqueue failed",
				zap.String("sender_id", msg.SenderID),
				zap.Error(err))
			continue
		}
		h.logger.Info("message enqueued",
			zap.String("sender_id", msg.SenderID),
			zap.Int64("job_id", jobID))
	}

	return c.SendStatus(fiber.StatusOK)
}

// Process implements queue.Processor — runs the full pipeline for one message.
func (h *Handler) Process(ctx context.Context, senderID string, payload []byte) error {
	var msg IncomingMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	return h.process(ctx, msg)
}

// process runs the full per-message pipeline (STT if audio, Brain, TTS if needed, send).
func (h *Handler) process(ctx context.Context, msg IncomingMessage) error {
	h.messenger.MarkSeen(msg.SenderID)
	h.messenger.SetTypingOn(msg.SenderID)

	respondWithAudio := false
	var inputText string

	switch msg.Type {
	case MessageTypeAudio:
		if h.voice == nil || !h.voice.Enabled() || msg.MediaURL == "" {
			return nil
		}
		audioData, err := h.messenger.DownloadMedia(msg.MediaURL)
		if err != nil {
			return fmt.Errorf("download audio: %w", err)
		}
		transcription, err := h.voice.SpeechToText(audioData)
		if err != nil {
			return fmt.Errorf("STT: %w", err)
		}
		h.logger.Info("audio transcribed",
			zap.String("sender_id", msg.SenderID),
			zap.String("transcription", transcription),
		)
		inputText = transcription
		respondWithAudio = true

	case MessageTypeText:
		if msg.Text == "" {
			return nil
		}
		inputText = msg.Text
		if wantsAudio(msg.Text) {
			respondWithAudio = true
		}

	default:
		return nil
	}

	reply, err := h.ai.Process(msg.SenderID, inputText)
	if err != nil {
		return fmt.Errorf("brain: %w", err)
	}
	if strings.TrimSpace(reply.Text) == "" && strings.TrimSpace(reply.AudioScript) == "" {
		h.logger.Info("no reply generated by flow", zap.String("sender_id", msg.SenderID))
		return nil
	}

	// Pequeño delay para que la respuesta no sea instantánea (evita marcar visto
	// si el usuario ya cerró la conversación).
	select {
	case <-time.After(2 * time.Second):
	case <-ctx.Done():
		return ctx.Err()
	}

	// Trust-building audio (sent BEFORE the text, when the brain decided so).
	if reply.AudioScript != "" && h.voice != nil && h.voice.Enabled() {
		if err := h.sendAudioReply(msg.SenderID, reply.AudioScript); err != nil {
			h.logger.Error("trust audio failed, continuing with text only",
				zap.String("sender_id", msg.SenderID), zap.Error(err))
		} else {
			// Small gap so the audio lands first in the chat.
			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if respondWithAudio && h.voice != nil && h.voice.Enabled() {
		if err := h.sendAudioReply(msg.SenderID, reply.Text); err != nil {
			h.logger.Error("audio reply failed, fallback to text",
				zap.String("sender_id", msg.SenderID), zap.Error(err))
			if terr := h.messenger.SendText(msg.SenderID, reply.Text); terr != nil {
				return fmt.Errorf("send text fallback: %w", terr)
			}
		}
	} else if strings.TrimSpace(reply.Text) != "" {
		if err := h.messenger.SendText(msg.SenderID, reply.Text); err != nil {
			return fmt.Errorf("send text: %w", err)
		}
	}

	h.logger.Info("reply sent",
		zap.String("sender_id", msg.SenderID),
		zap.String("reply", reply.Text),
		zap.Bool("audio", respondWithAudio),
		zap.Bool("trust_audio", reply.AudioScript != ""),
	)
	return nil
}

// sendAudioReply converts text to speech and sends it as an audio message via public URL.
func (h *Handler) sendAudioReply(recipientID, text string) error {
	audioData, err := h.voice.TextToSpeech(text)
	if err != nil {
		return fmt.Errorf("TTS: %w", err)
	}

	audioID := h.audioStore.Put(audioData)
	audioURL := fmt.Sprintf("%s/audio/%s", h.cfg.PublicURL, audioID)

	h.logger.Debug("audio URL generated",
		zap.String("url", audioURL),
		zap.Int("bytes", len(audioData)),
	)

	return h.messenger.SendAudio(recipientID, audioURL)
}

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

func validateSignature(body []byte, signature string, appSecret string) bool {
	if signature == "" {
		return false
	}
	expected := computeSignature(body, appSecret)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func computeSignature(body []byte, appSecret string) string {
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

type mediaListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (h *Handler) fetchLatestMediaID(ctx context.Context) (string, error) {
	if h.cfg.InstagramAccountID == "" || h.cfg.PageAccessToken == "" {
		return "", fmt.Errorf("instagram credentials not configured")
	}

	url := fmt.Sprintf("%s/%s/media?fields=id&limit=1", graphAPIBase, h.cfg.InstagramAccountID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create latest media request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.PageAccessToken)

	resp, err := h.graphHTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest media: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			h.logger.Warn("failed to close latest media response body", zap.Error(cerr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			h.logger.Warn("failed to read latest media error response body", zap.Error(readErr))
		}
		return "", fmt.Errorf("fetch latest media: status %d body=%q", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var mediaResp mediaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&mediaResp); err != nil {
		return "", fmt.Errorf("decode latest media response: %w", err)
	}
	if len(mediaResp.Data) == 0 || mediaResp.Data[0].ID == "" {
		return "", fmt.Errorf("latest media not found")
	}

	return mediaResp.Data[0].ID, nil
}
