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

	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/domain"
	"github.com/kart-academy/instagram-bot/internal/webhook"
)

type Server struct {
	app    *fiber.App
	cfg    *config.Config
	logger *zap.Logger
}

type Dependencies struct {
	Messenger  domain.Messenger
	AI         domain.AIEngine
	Voice      domain.VoiceService
	AudioStore domain.AudioStore
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

	wh := webhook.NewHandler(s.cfg, deps.Messenger, deps.AI, deps.Voice, deps.AudioStore, s.logger)
	s.app.Get("/webhook", wh.Verify)
	s.app.Post("/webhook", wh.Receive)
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
