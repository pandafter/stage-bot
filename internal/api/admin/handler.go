package admin

import (
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/storage"
)

// Handler is the entry point for the admin package — groups auth, CMS, media, form, inscripciones.
type Handler struct {
	cfg    *config.Config
	Auth   *AuthHandler
	logger *zap.Logger
}

func NewHandler(
	cfg *config.Config,
	users *storage.AdminUsersRepo,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		cfg:    cfg,
		Auth:   NewAuthHandler(cfg, users, logger),
		logger: logger,
	}
}
