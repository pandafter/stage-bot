package main

import (
	"log"

	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/ai"
	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/knowledge"
	"github.com/kart-academy/instagram-bot/internal/messenger"
	"github.com/kart-academy/instagram-bot/internal/server"
	"github.com/kart-academy/instagram-bot/internal/storage"
	"github.com/kart-academy/instagram-bot/internal/voice"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger
	var logger *zap.Logger
	if cfg.IsDevelopment() {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	// Initialize database
	db, err := storage.NewDB(cfg.DatabaseURL, logger)
	if err != nil {
		logger.Fatal("failed to init database", zap.Error(err))
	}
	defer db.Close()

	// Build dependencies
	ig := messenger.NewInstagram(cfg.PageAccessToken, cfg.InstagramAccountID, logger)
	ks := knowledge.NewStore(cfg.GoogleSheetID, logger)
	brain := ai.New(cfg.AnthropicAPIKey, ks, logger)
	vc := voice.NewElevenLabs(cfg.ElevenLabsAPIKey, cfg.ElevenLabsVoiceID, logger)
	audioStore := voice.NewAudioStore()

	deps := server.Dependencies{
		Messenger:  ig,
		AI:         brain,
		Voice:      vc,
		AudioStore: audioStore,
	}

	// Start server
	srv := server.New(cfg, deps, logger)
	if err := srv.Start(); err != nil {
		logger.Fatal("server error", zap.Error(err))
	}
}
