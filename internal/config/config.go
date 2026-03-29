package config

import (
	"fmt"
	"os"
	"strconv"

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
	ElevenLabsAPIKey string
	ElevenLabsVoiceID string

	// OpenAI (Whisper)
	OpenAIAPIKey string

	// Database
	DatabaseURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:               getIntEnv("PORT", 8080),
		Env:                getEnv("ENV", "development"),
		AppID:              getEnv("APP_ID", ""),
		AppSecret:          getEnv("APP_SECRET", ""),
		PageAccessToken:    getEnv("PAGE_ACCESS_TOKEN", ""),
		InstagramAccountID: getEnv("INSTAGRAM_ACCOUNT_ID", ""),
		WebhookVerifyToken: getEnv("WEBHOOK_VERIFY_TOKEN", ""),
		AnthropicAPIKey:    getEnv("ANTHROPIC_API_KEY", ""),
		ElevenLabsAPIKey:   getEnv("ELEVENLABS_API_KEY", ""),
		ElevenLabsVoiceID:  getEnv("ELEVENLABS_VOICE_ID", ""),
		OpenAIAPIKey:       getEnv("OPENAI_API_KEY", ""),
		DatabaseURL:        getEnv("DATABASE_URL", "data/bot.db"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) validate() error {
	required := map[string]string{
		"WEBHOOK_VERIFY_TOKEN": c.WebhookVerifyToken,
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
