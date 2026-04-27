package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port int
	Env  string

	// Meta / Instagram
	AppID              string
	AppSecret          string
	PageAccessToken    string
	InstagramAccountID string
	WebhookVerifyToken string

	// Anthropic
	AnthropicAPIKey string

	// ElevenLabs
	ElevenLabsAPIKey  string
	ElevenLabsVoiceID string

	// OpenAI (Whisper)
	OpenAIAPIKey string

	// Google Sheets knowledge base
	GoogleSheetID string

	// Server public URL (ngrok)
	PublicURL string

	// Database
	DatabaseURL string

	// Testing — only respond to this sender in development
	TestSenderID string
	// Trigger keyword for comment-to-DM sales flow
	CommentTriggerKeyword string

	// Admin panel auth token
	AdminToken string

	// MCP / GitHub integration
	GitHubToken  string
	GitHubRepo   string
	MCPSecret    string
	BotPrompt    string
	SystemPrompt string

	// Queue worker pool
	WorkerConcurrency int
	WorkerMaxAttempts int

	// Claude pricing per million tokens (USD). Defaults to Sonnet rates.
	ClaudePriceInputPerMTok  float64
	ClaudePriceOutputPerMTok float64

	// Inscripcion (public registration form)
	UploadsDir        string
	TelegramBotToken  string
	TelegramChatID    string
	BoldWebhookSecret string
	BoldAPIKey        string // identity key for Bold's POST link creation API
	SheetsWebhookURL  string // Apps Script web app URL receiving inscripciones
	SheetsSharedToken string // shared secret echoed back to verify the call
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:                     getIntEnv("PORT", 8080),
		Env:                      getEnv("ENV", "development"),
		AppID:                    getEnv("APP_ID", ""),
		AppSecret:                getEnv("APP_SECRET", ""),
		PageAccessToken:          getEnv("PAGE_ACCESS_TOKEN", ""),
		InstagramAccountID:       getEnv("INSTAGRAM_ACCOUNT_ID", ""),
		WebhookVerifyToken:       getEnv("WEBHOOK_VERIFY_TOKEN", ""),
		AnthropicAPIKey:          getEnv("ANTHROPIC_API_KEY", ""),
		ElevenLabsAPIKey:         getEnv("ELEVENLABS_API_KEY", ""),
		ElevenLabsVoiceID:        getEnv("ELEVENLABS_VOICE_ID", ""),
		OpenAIAPIKey:             getEnv("OPENAI_API_KEY", ""),
		GoogleSheetID:            getEnv("GOOGLE_SHEET_ID", ""),
		PublicURL:                getEnv("PUBLIC_URL", ""),
		DatabaseURL:              getEnv("DATABASE_URL", ""),
		TestSenderID:             getEnv("TEST_SENDER_ID", ""),
		CommentTriggerKeyword:    getEnv("COMMENT_TRIGGER_KEYWORD", "piloto"),
		AdminToken:               getEnv("ADMIN_TOKEN", ""),
		GitHubToken:              getEnv("GITHUB_TOKEN", ""),
		GitHubRepo:               getEnv("GITHUB_REPO", ""),
		MCPSecret:                getEnv("MCP_SECRET", ""),
		BotPrompt:                getEnv("BOT_PROMPT", ""),
		SystemPrompt:             getEnv("SYSTEM_PROMPT", ""),
		WorkerConcurrency:        getIntEnv("WORKER_CONCURRENCY", 5),
		WorkerMaxAttempts:        getIntEnv("WORKER_MAX_ATTEMPTS", 3),
		ClaudePriceInputPerMTok:  getFloatEnv("CLAUDE_PRICE_INPUT_PER_MTOK", 3.0),
		ClaudePriceOutputPerMTok: getFloatEnv("CLAUDE_PRICE_OUTPUT_PER_MTOK", 15.0),
		UploadsDir:               getEnv("UPLOADS_DIR", "./uploads"),
		TelegramBotToken:         getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:           getEnv("TELEGRAM_CHAT_ID", ""),
		BoldWebhookSecret:        getEnv("BOLD_WEBHOOK_SECRET", ""),
		BoldAPIKey:               getEnv("BOLD_API_KEY", ""),
		SheetsWebhookURL:         getEnv("SHEETS_WEBHOOK_URL", ""),
		SheetsSharedToken:        getEnv("SHEETS_SHARED_TOKEN", ""),
	}

	cfg.PublicURL = normalizePublicURL(cfg.PublicURL)

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// normalizePublicURL ensures the URL has an https scheme and no trailing slash.
// Instagram requires a fully-qualified URL with scheme to fetch audio attachments.
func normalizePublicURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return u
	}
	u = strings.TrimRight(u, "/")
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}
	return u
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) validate() error {
	required := map[string]string{
		"WEBHOOK_VERIFY_TOKEN": c.WebhookVerifyToken,
		"DATABASE_URL":         c.DatabaseURL,
	}

	for name, val := range required {
		if val == "" {
			return fmt.Errorf("missing required env var: %s", name)
		}
	}

	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}

func getFloatEnv(key string, fallback float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fallback
	}
	return f
}
