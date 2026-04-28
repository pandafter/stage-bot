package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/admin"
	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/domain"
	"github.com/kart-academy/instagram-bot/internal/inscripcion"
	"github.com/kart-academy/instagram-bot/internal/mcp"
	"github.com/kart-academy/instagram-bot/internal/queue"
	"github.com/kart-academy/instagram-bot/internal/storage"
	"github.com/kart-academy/instagram-bot/internal/webhook"
)

type Server struct {
	app    *fiber.App
	cfg    *config.Config
	logger *zap.Logger
}

type Dependencies struct {
	Messenger      domain.Messenger
	AI             domain.AIEngine
	Voice          domain.VoiceService
	AudioStore     domain.AudioStore
	Leads          *storage.LeadsRepo
	Conversation   *storage.ConversationRepo
	Inscripciones  *storage.InscripcionesRepo
	Queue          queue.Queue
	WebhookHandler *webhook.Handler
}

func New(cfg *config.Config, deps Dependencies, logger *zap.Logger) *Server {
	app := fiber.New(fiber.Config{
		AppName:               "kart-academy-bot",
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           30 * time.Second,
		DisableStartupMessage: false,
	})

	app.Use(recover.New())

	s := &Server{
		app:    app,
		cfg:    cfg,
		logger: logger,
	}

	s.setupRoutes(deps)

	return s
}

func (s *Server) setupRoutes(deps Dependencies) {
	s.app.Get("/health", s.healthHandler)

	// Audio file serving endpoint
	s.app.Get("/audio/:id", func(c *fiber.Ctx) error {
		data, ok := deps.AudioStore.Get(c.Params("id"))
		if !ok {
			return c.SendStatus(fiber.StatusNotFound)
		}
		c.Set("Content-Type", "audio/mp4")
		return c.Send(data)
	})

	if s.cfg.BotEnabled {
		wh := deps.WebhookHandler
		if wh == nil {
			wh = webhook.NewHandler(s.cfg, deps.Messenger, deps.AI, deps.Voice, deps.AudioStore, deps.Queue, s.logger)
		}
		s.app.Get("/webhook", wh.Verify)
		s.app.Post("/webhook", wh.Receive)
	} else {
		s.logger.Info("bot disabled — Meta /webhook routes NOT registered (full disconnect)")
	}

	mcpHandler := mcp.NewHandler(s.cfg, s.logger)
	s.app.Post("/mcp/token", mcpHandler.Token)
	s.app.Get("/authorize", mcpHandler.Authorize)
	s.app.Post("/authorize", mcpHandler.AuthorizeSubmit)
	s.app.Get("/mcp", mcpHandler.Auth, mcpHandler.Stream)
	s.app.Post("/mcp", mcpHandler.Auth, mcpHandler.HandleJSONRPC)

	// OAuth discovery endpoints required by Claude mobile / Claude Desktop
	// to register this MCP server as a Custom Connector.
	s.app.Get("/.well-known/oauth-authorization-server", mcpHandler.OAuthMetadata)
	s.app.Get("/.well-known/oauth-protected-resource", mcpHandler.ProtectedResourceMetadata)
	// Some clients append the MCP path to the well-known URL; alias both.
	s.app.Get("/.well-known/oauth-authorization-server/mcp", mcpHandler.OAuthMetadata)
	s.app.Get("/.well-known/oauth-protected-resource/mcp", mcpHandler.ProtectedResourceMetadata)
	// MCP spec also asks for dynamic client registration metadata; we accept any client.
	s.app.Post("/mcp/register", mcpHandler.DynamicClientRegister)

 	if deps.Inscripciones != nil {
 		ih := inscripcion.NewHandler(inscripcion.Config{
 			UploadsDir:       s.cfg.UploadsDir,
 			PublicURL:        s.cfg.PublicURL,
 			TelegramBotToken:  s.cfg.TelegramBotToken,
 			TelegramChatID:    s.cfg.TelegramChatID,
 			AdminToken:        s.cfg.AdminToken,
 			BoldWebhookSecret: s.cfg.BoldWebhookSecret,
 			BoldAPIKey:        s.cfg.BoldAPIKey,
 			SheetsWebhookURL:  s.cfg.SheetsWebhookURL,
 			SheetsSharedToken: s.cfg.SheetsSharedToken,
 		}, deps.Inscripciones, s.logger)
 		s.app.Get("/inscripcion", ih.ServeForm)
 		s.app.Post("/inscripcion", ih.Submit)
 		s.app.Get("/inscripcion/logo.jpg", ih.ServeLogo)
 		s.app.Get("/inscripcion/assets/bancolombiaLogo.png", ih.ServeBancolombiaLogo)
 		s.app.Get("/inscripcion/assets/nequiLogo.webp", ih.ServeNequiLogo)
 		s.app.Get("/inscripcion/telegram-test", ih.TelegramTest)
 		s.app.Get("/inscripcion/telegram-debug", ih.TelegramDebug)
 		s.app.Get("/inscripcion/callback", ih.Callback)
 		s.app.Get("/inscripcion/status", ih.Status)
 		s.app.Post("/webhook/bold", ih.BoldWebhook)
 		s.app.Get("/inscripcion/bold-debug", ih.BoldDebug)
 		s.app.Get("/", ih.ServeLanding)
 	}

	if deps.Leads != nil && deps.Conversation != nil {
		ah := admin.NewHandler(s.cfg, deps.Leads, deps.Conversation, deps.Messenger, deps.AI, deps.Queue, s.logger)
		adminGroup := s.app.Group("/admin", ah.Auth)
		adminGroup.Get("/", ah.Dashboard)
		adminGroup.Get("", ah.Dashboard)
		adminGroup.Get("/lead/:id", ah.LeadDetail)
		adminGroup.Post("/lead/:id/retake", ah.RetakeLead)
		adminGroup.Post("/lead/:id/outcome", ah.SetOutcome)
		adminGroup.Get("/instagram/conversations", ah.InstagramConversations)
		adminGroup.Get("/ping/claude", ah.PingClaude)
		adminGroup.Get("/ping/elevenlabs", ah.PingElevenLabs)
		adminGroup.Get("/ping/instagram", ah.PingInstagram)
		adminGroup.Get("/queue", ah.QueueView)
		adminGroup.Post("/queue/retry/:id", ah.QueueRetry)
		adminGroup.Post("/send", ah.SendToUser)
	}
}

func (s *Server) healthHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
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
