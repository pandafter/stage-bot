package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/ai"
	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/knowledge"
	"github.com/kart-academy/instagram-bot/internal/messenger"
	"github.com/kart-academy/instagram-bot/internal/queue"
	"github.com/kart-academy/instagram-bot/internal/server"
	"github.com/kart-academy/instagram-bot/internal/storage"
	"github.com/kart-academy/instagram-bot/internal/voice"
	"github.com/kart-academy/instagram-bot/internal/webhook"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

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

	db, err := storage.NewDB(cfg.DatabaseURL, logger)
	if err != nil {
		logger.Fatal("failed to init database", zap.Error(err))
	}
	defer db.Close()

	leadsRepo := storage.NewLeadsRepo(db)
	convRepo := storage.NewConversationRepo(db)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	playbook := ai.NewPlaybook(leadsRepo, convRepo, logger)
	playbook.Start(ctx)

	ig := messenger.NewInstagram(cfg.PageAccessToken, cfg.InstagramAccountID, logger)
	ks := knowledge.NewStore(cfg.GoogleSheetID, logger)

	salesDataset, err := knowledge.LoadSalesDataset("internal/knowledge/dataset_ventas_instagram.json", logger)
	if err != nil {
		logger.Warn("sales dataset not loaded, continuing without", zap.Error(err))
	}

	brain := ai.New(cfg.AnthropicAPIKey, ks, salesDataset, leadsRepo, convRepo, playbook, logger)
	vc := voice.NewElevenLabs(cfg.ElevenLabsAPIKey, cfg.ElevenLabsVoiceID, logger)
	audioStore := voice.NewAudioStore()

	followup := ai.NewFollowUp(brain, ig, leadsRepo, convRepo, logger)
	followup.Start(ctx)

	jobQueue := queue.NewPostgres(db.Conn())

	webhookHandler := webhook.NewHandler(cfg, ig, brain, vc, audioStore, jobQueue, logger)

	pool := queue.NewPool(jobQueue, webhookHandler, queue.Config{
		Concurrency: cfg.WorkerConcurrency,
		MaxAttempts: cfg.WorkerMaxAttempts,
	}, logger)
	pool.Start(ctx)

	deps := server.Dependencies{
		Messenger:      ig,
		AI:             brain,
		Voice:          vc,
		AudioStore:     audioStore,
		Leads:          leadsRepo,
		Conversation:   convRepo,
		Queue:          jobQueue,
		WebhookHandler: webhookHandler,
	}

	srv := server.New(cfg, deps, logger)
	if err := srv.Start(); err != nil {
		logger.Fatal("server error", zap.Error(err))
	}
}
