package main

import (
	"context"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/api"
	adminpkg "github.com/kart-academy/instagram-bot/internal/api/admin"
	"github.com/kart-academy/instagram-bot/internal/bot"
	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/media"
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
	botLeadsRepo := storage.NewBotLeadsRepo(db)
	botHandler := bot.NewHandler(cfg, botLeadsRepo, logger)

	cmsRepo := storage.NewCMSRepo(db)
	formConfigRepo := storage.NewFormConfigRepo(db)
	adminUsersRepo := storage.NewAdminUsersRepo(db)
	tenantsRepo := storage.NewTenantsRepo(db)

	// Attach CMS repo to public API so /api/config includes published sections.
	apiHandler.WithCMSRepo(cmsRepo)
	// Attach tenants repo so /api/config includes the tenant theme.
	apiHandler.WithTenantsRepo(tenantsRepo)
	// Attach form config repo so /api/config serves DB-managed dates.
	apiHandler.WithFormConfigRepo(formConfigRepo)

	// Bootstrap: create first admin if ADMIN_PASSWORD_HASH is set and no users exist for the tenant.
	if cfg.AdminPasswordHash != "" {
		count, err := adminUsersRepo.CountByTenant(ctx, cfg.DefaultTenant)
		if err != nil {
			logger.Error("admin bootstrap check", zap.Error(err))
		} else if count == 0 {
			_, err = adminUsersRepo.Create(ctx, cfg.DefaultTenant, cfg.AdminUsername, cfg.AdminPasswordHash, "owner")
			if err != nil {
				logger.Error("admin bootstrap create", zap.Error(err))
			} else {
				logger.Info("admin bootstrap: created initial admin", zap.String("username", cfg.AdminUsername))
			}
		}
	}

	// Seed CMS sections with default content if they don't exist yet.
	if err := storage.SeedCMSSections(ctx, cmsRepo, cfg.DefaultTenant); err != nil {
		logger.Error("cms seed", zap.Error(err))
	} else {
		logger.Info("cms seed: OK")
	}

	// Seed form config (dates, plans, payment methods) if empty.
	if err := storage.SeedFormConfig(ctx, formConfigRepo, cfg.DefaultTenant); err != nil {
		logger.Error("form config seed", zap.Error(err))
	} else {
		logger.Info("form config seed: OK")
	}

	mediaStore, err := media.NewFromConfig(cfg)
	if err != nil {
		logger.Fatal("media store init", zap.Error(err))
	}

	auditRepo := storage.NewAuditRepo(db)

	adminHandler := adminpkg.NewHandler(cfg, adminUsersRepo, cmsRepo, formConfigRepo, repo, mediaStore, tenantsRepo, auditRepo, logger)

	srv := server.New(cfg, server.Dependencies{API: apiHandler, Bot: botHandler, Admin: adminHandler}, logger)
	if err := srv.Start(); err != nil {
		logger.Fatal("server", zap.Error(err))
	}
}
