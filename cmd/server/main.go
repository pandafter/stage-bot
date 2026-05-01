package main

import (
	"context"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/api"
	"github.com/kart-academy/instagram-bot/internal/bot"
	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/server"
	"github.com/kart-academy/instagram-bot/internal/storage"
)

func main() {
	_ = godotenv.Load()

	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("config", zap.Error(err))
	}

	ctx := context.Background()
	db, err := storage.New(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		logger.Fatal("db init", zap.Error(err))
	}
	defer db.Close()

	repo := storage.NewInscripcionesRepo(db)
	telegram := api.NewTelegramClient(cfg.TelegramBotToken, cfg.TelegramChatID, logger)
	bold := api.NewBoldClient(cfg.BoldAPIKey, cfg.PublicURL, logger)
	apiHandler := api.NewHandler(cfg, repo, telegram, bold, logger)
	botHandler := bot.NewHandler(cfg, logger) // nil if PAGE_ACCESS_TOKEN not set

	srv := server.New(cfg, server.Dependencies{API: apiHandler, Bot: botHandler}, logger)
	if err := srv.Start(); err != nil {
		logger.Fatal("server", zap.Error(err))
	}
}
