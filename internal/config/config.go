package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      int
	Env       string
	PublicURL string

	DatabaseURL string

	// Bold (pasarela de pago)
	BoldAPIKey        string
	BoldWebhookSecret string

	// Telegram (notificaciones al director)
	TelegramBotToken string
	TelegramChatID   string

	// Storage
	UploadsDir string

	// Admin (endpoints diagnósticos opcionales)
	AdminToken string

	// Instagram Basic Display API — opcional, para la sección de posts en el landing.
	// Si está vacío, el endpoint /api/instagram devuelve [] y el frontend muestra fallback.
	InstagramAccessToken string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:              getIntEnv("PORT", 3000),
		Env:               getEnv("ENV", "development"),
		PublicURL:         normalizePublicURL(getEnv("PUBLIC_URL", "")),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		BoldAPIKey:        getEnv("BOLD_API_KEY", ""),
		BoldWebhookSecret: getEnv("BOLD_WEBHOOK_SECRET", ""),
		TelegramBotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:    getEnv("TELEGRAM_CHAT_ID", ""),
		UploadsDir:           getEnv("UPLOADS_DIR", "./uploads"),
		AdminToken:           getEnv("ADMIN_TOKEN", ""),
		InstagramAccessToken: getEnv("INSTAGRAM_ACCESS_TOKEN", ""),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) IsDevelopment() bool { return c.Env == "development" }

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("missing required env var: DATABASE_URL")
	}
	return nil
}

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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
