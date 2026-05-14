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

	// Instagram Bot (auto-responder de DMs)
	PageAccessToken    string // Page access token with instagram_manage_messages permission
	IGAccountID        string // Instagram-scoped account ID (the page's IG identity)
	WebhookVerifyToken string // Token to verify Meta webhook subscription
	AppSecret          string // Facebook App Secret for webhook signature verification
	TestSenderID       string // In development, only respond to this sender ID
	BotAudioURL        string // URL of the director's pre-recorded audio message

	// Admin Panel (JWT + bootstrap)
	JWTSecret         string
	AdminUsername     string
	AdminPasswordHash string // bcrypt hash
	DefaultTenant     string

	// Cloudflare R2 (media storage)
	R2AccountID  string
	R2AccessKey  string
	R2SecretKey  string
	R2Bucket     string
	R2PublicURL  string
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

		// Instagram Bot
		PageAccessToken:    getEnv("PAGE_ACCESS_TOKEN", ""),
		IGAccountID:        getEnv("INSTAGRAM_ACCOUNT_ID", ""),
		WebhookVerifyToken: getEnv("WEBHOOK_VERIFY_TOKEN", ""),
		AppSecret:          getEnv("APP_SECRET", ""),
		TestSenderID:       getEnv("TEST_SENDER_ID", ""),
		BotAudioURL:        getEnv("BOT_AUDIO_URL", ""),

		// Admin Panel
		JWTSecret:         getEnv("JWT_SECRET", ""),
		AdminUsername:     getEnv("ADMIN_USERNAME", "admin"),
		AdminPasswordHash: getEnv("ADMIN_PASSWORD_HASH", ""),
		DefaultTenant:     getEnv("DEFAULT_TENANT", "st4ge"),

		// Cloudflare R2
		R2AccountID:  getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKey:  getEnv("R2_ACCESS_KEY", ""),
		R2SecretKey:  getEnv("R2_SECRET_KEY", ""),
		R2Bucket:     getEnv("R2_BUCKET", ""),
		R2PublicURL:  getEnv("R2_PUBLIC_URL", ""),
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
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
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
