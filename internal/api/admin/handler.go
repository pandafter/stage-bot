package admin

import (
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/media"
	"github.com/kart-academy/instagram-bot/internal/storage"
)

type Handler struct {
	cfg           *config.Config
	Auth          *AuthHandler
	CMS           *CMSHandler
	Media         *MediaHandler
	FormConfig    *FormConfigHandler
	Inscripciones *InscripcionesAdminHandler
	Theme         *ThemeHandler
	Audit         *storage.AuditRepo
	logger        *zap.Logger
}

func NewHandler(
	cfg *config.Config,
	users *storage.AdminUsersRepo,
	cms *storage.CMSRepo,
	formConfig *storage.FormConfigRepo,
	inscripcionesRepo *storage.InscripcionesRepo,
	mediaStore media.Store,
	tenantsRepo *storage.TenantsRepo,
	audit *storage.AuditRepo,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		cfg:           cfg,
		Auth:          NewAuthHandler(cfg, users, logger),
		CMS:           NewCMSHandler(cms, audit, logger),
		Media:         NewMediaHandler(mediaStore, cms, logger),
		FormConfig:    NewFormConfigHandler(formConfig, logger),
		Inscripciones: NewInscripcionesAdminHandler(inscripcionesRepo, audit, logger),
		Theme:         NewThemeHandler(tenantsRepo, logger),
		Audit:         audit,
		logger:        logger,
	}
}
