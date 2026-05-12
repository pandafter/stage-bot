package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/api"
	adminpkg "github.com/kart-academy/instagram-bot/internal/api/admin"
	"github.com/kart-academy/instagram-bot/internal/bot"
	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/spa"
)

type Server struct {
	app    *fiber.App
	cfg    *config.Config
	logger *zap.Logger
}

type Dependencies struct {
	API   *api.Handler
	Bot   *bot.Handler       // nil if bot is not configured
	Admin *adminpkg.Handler  // nil if admin is not configured
}

func New(cfg *config.Config, deps Dependencies, logger *zap.Logger) *Server {
	app := fiber.New(fiber.Config{
		AppName:               "scuderia-st4ge",
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           60 * time.Second,
		BodyLimit:             15 * 1024 * 1024, // accept ≤10MB uploads with headroom
		DisableStartupMessage: false,
	})

	app.Use(recover.New())

	allowOrigins := "https://www.scuderiast4ge.com,https://scuderiast4ge.com"
	if cfg.IsDevelopment() {
		allowOrigins = "http://localhost:5173,http://localhost:3000"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, x-bold-signature",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
	}))

	s := &Server{app: app, cfg: cfg, logger: logger}
	s.setupRoutes(deps)
	return s
}

func (s *Server) setupRoutes(deps Dependencies) {
	// Health check (Railway probe)
	s.app.Get("/health", api.Health)

	// API
	s.app.Get("/api/config", deps.API.GetConfig)
	s.app.Post("/api/inscripciones", deps.API.Create)
	s.app.Get("/api/inscripciones/:id", deps.API.Get)
	s.app.Post("/api/inscripciones/:id/comprobante", deps.API.UploadComprobante)
	s.app.Get("/api/instagram", deps.API.GetInstagramPosts)

	// Bold webhook (server-to-server, HMAC-signed)
	s.app.Post("/webhook/bold", deps.API.BoldWebhook)

	// Instagram bot webhook (Meta Messaging API)
	if deps.Bot != nil {
		s.app.Get("/webhook/instagram", deps.Bot.Verify)
		s.app.Post("/webhook/instagram", deps.Bot.Receive)
		s.logger.Info("🤖 Instagram bot webhook activo — /webhook/instagram")
	}

	// Dev-only: simular pagos sin pasar por Bold (solo ENV=development)
	if s.cfg.IsDevelopment() {
		s.app.Post("/dev/inscripciones/:id/confirm", deps.API.DevConfirmPayment)
		s.app.Post("/dev/inscripciones/:id/reject", deps.API.DevRejectPayment)
		s.logger.Info("⚠️  dev endpoints activos — POST /dev/inscripciones/:id/confirm|reject")
	}

	// Admin panel routes
	if deps.Admin != nil {
		adminGroup := s.app.Group("/api/admin")

		// Auth (public — no JWT required)
		adminGroup.Post("/auth/login", deps.Admin.Auth.Login)

		// Protected routes
		protected := adminGroup.Use(adminpkg.RequireJWT(s.cfg))
		protected.Get("/me", deps.Admin.Auth.Me)

		// CMS sections
		protected.Get("/cms/sections", deps.Admin.CMS.GetSections)
		protected.Get("/cms/sections/:key", deps.Admin.CMS.GetSection)
		protected.Put("/cms/sections/:key", deps.Admin.CMS.UpsertSection)

		// Media
		protected.Get("/media", deps.Admin.Media.List)
		protected.Post("/media", deps.Admin.Media.Upload)
		protected.Delete("/media/:id", deps.Admin.Media.Delete)
		protected.Patch("/media/:id", deps.Admin.Media.Update)

		// Form config — dates
		protected.Get("/form/dates", deps.Admin.FormConfig.ListDates)
		protected.Post("/form/dates", deps.Admin.FormConfig.CreateDate)
		protected.Post("/form/dates/reorder", deps.Admin.FormConfig.ReorderDates)
		protected.Patch("/form/dates/:id", deps.Admin.FormConfig.UpdateDate)
		protected.Delete("/form/dates/:id", deps.Admin.FormConfig.DeleteDate)

		// Form config — plans
		protected.Get("/form/plans", deps.Admin.FormConfig.ListPlans)
		protected.Post("/form/plans", deps.Admin.FormConfig.CreatePlan)
		protected.Patch("/form/plans/:id", deps.Admin.FormConfig.UpdatePlan)
		protected.Delete("/form/plans/:id", deps.Admin.FormConfig.DeletePlan)

		// Form config — payment methods
		protected.Get("/form/methods", deps.Admin.FormConfig.ListMethods)
		protected.Patch("/form/methods/:id", deps.Admin.FormConfig.UpdateMethod)

		// Theme
		protected.Get("/theme", deps.Admin.Theme.Get)
		protected.Put("/theme", deps.Admin.Theme.Update)

		// Inscripciones admin (export.csv must precede :id param)
		protected.Get("/inscripciones", deps.Admin.Inscripciones.List)
		protected.Get("/inscripciones/export.csv", deps.Admin.Inscripciones.ExportCSV)
		protected.Get("/inscripciones/:id", deps.Admin.Inscripciones.Get)
		protected.Patch("/inscripciones/:id/status", deps.Admin.Inscripciones.UpdateStatus)
	}

	// SPA fallback (must be last)
	if err := spa.Register(s.app); err != nil {
		s.logger.Error("spa register failed", zap.Error(err))
	}
}

func (s *Server) Start() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", s.cfg.Port)
		s.logger.Info("server starting", zap.String("addr", addr), zap.String("env", s.cfg.Env))
		errCh <- s.app.Listen(addr)
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		s.logger.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = ctx
	s.logger.Info("shutting down gracefully...")
	if err := s.app.Shutdown(); err != nil {
		s.logger.Error("server shutdown error", zap.Error(err))
		return err
	}
	s.logger.Info("server stopped")
	return nil
}
