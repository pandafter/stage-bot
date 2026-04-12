package voice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	elevenLabsBase = "https://api.elevenlabs.io/v1"
)

// ElevenLabs implements domain.VoiceService using the ElevenLabs API.
type ElevenLabs struct {
	apiKey     string
	voiceID    string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewElevenLabs(apiKey, voiceID string, logger *zap.Logger) *ElevenLabs {
	return &ElevenLabs{
		apiKey:  apiKey,
		voiceID: voiceID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Enabled returns true if the voice client is properly configured.
func (c *ElevenLabs) Enabled() bool {
	return c.apiKey != "" && c.voiceID != ""
}

// SpeechToText transcribes audio bytes to text using ElevenLabs STT.
func (c *ElevenLabs) SpeechToText(audioData []byte) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("model_id", "scribe_v1"); err != nil {
		return "", fmt.Errorf("write model_id field: %w", err)
	}
	if err := w.WriteField("language_code", "es"); err != nil {
		return "", fmt.Errorf("write language field: %w", err)
	}

	fw, err := w.CreateFormFile("file", "audio.mp4")
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := fw.Write(audioData); err != nil {
		return "", fmt.Errorf("write audio data: %w", err)
	}

	w.Close()

	url := fmt.Sprintf("%s/speech-to-text", elevenLabsBase)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return "", fmt.Errorf("create STT request: %w", err)
	}
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("STT request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read STT response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("elevenlabs STT error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)),
		)
		return "", fmt.Errorf("STT error: status %d: %s", resp.StatusCode, string(respBody))
	}

	var sttResp struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBody, &sttResp); err != nil {
		return "", fmt.Errorf("unmarshal STT response: %w", err)
	}

	c.logger.Debug("STT transcription", zap.String("text", sttResp.Text))
	return sttResp.Text, nil
}

// TextToSpeech converts text to audio bytes (M4A) using the configured cloned voice.
func (c *ElevenLabs) TextToSpeech(text string) ([]byte, error) {
	payload := map[string]interface{}{
		"text":     text,
		"model_id": "eleven_multilingual_v2",
		"voice_settings": map[string]interface{}{
			"stability":        0.5,
			"similarity_boost": 0.8,
			"style":            0.3,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal TTS request: %w", err)
	}

	url := fmt.Sprintf("%s/text-to-speech/%s?output_format=mp3_22050_32", elevenLabsBase, c.voiceID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create TTS request: %w", err)
	}
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("TTS request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read TTS response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("elevenlabs TTS error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)),
		)
		return nil, fmt.Errorf("TTS error: status %d: %s", resp.StatusCode, string(respBody))
	}

	// Convert MP3 to M4A (AAC) for Instagram compatibility
	m4a, err := ConvertMP3toM4A(respBody)
	if err != nil {
		c.logger.Error("ffmpeg conversion failed, returning mp3", zap.Error(err))
		return respBody, nil
	}

	c.logger.Debug("TTS generated", zap.Int("mp3_bytes", len(respBody)), zap.Int("m4a_bytes", len(m4a)))
	return m4a, nil
}
