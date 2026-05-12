package api

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/storage"
)

type Handler struct {
	cfg           *config.Config
	repo          *storage.InscripcionesRepo
	telegram      *TelegramClient
	bold          *BoldClient
	logger        *zap.Logger
	cmsRepo       *storage.CMSRepo
	tenantsRepo   *storage.TenantsRepo
	defaultTenant string
}

func NewHandler(cfg *config.Config, repo *storage.InscripcionesRepo, telegram *TelegramClient, bold *BoldClient, logger *zap.Logger) *Handler {
	if cfg.UploadsDir != "" {
		_ = os.MkdirAll(cfg.UploadsDir, 0o755)
	}
	return &Handler{
		cfg:           cfg,
		repo:          repo,
		telegram:      telegram,
		bold:          bold,
		logger:        logger,
		defaultTenant: cfg.DefaultTenant,
	}
}

// WithCMSRepo attaches a CMS repository so GetConfig can include published sections.
func (h *Handler) WithCMSRepo(cms *storage.CMSRepo) {
	h.cmsRepo = cms
}

// WithTenantsRepo attaches a tenants repository so GetConfig can include the tenant theme.
func (h *Handler) WithTenantsRepo(tenants *storage.TenantsRepo) {
	h.tenantsRepo = tenants
}

// Create handles POST /api/inscripciones.
func (h *Handler) Create(c *fiber.Ctx) error {
	var req CreateInscripcionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}

	modalidad, method, err := validateCreate(&req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	rec := req.toRecord()
	rec.ID = newID()
	rec.MontoCOP = computeMonto(modalidad, method.ID)
	rec.Status = "pendiente"

	ctx, cancel := context.WithTimeout(c.UserContext(), 15*time.Second)
	defer cancel()
	if err := h.repo.Insert(ctx, rec); err != nil {
		h.logger.Error("insert inscripcion", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "db error"})
	}

	var checkoutURL string
	if methodIsDigital(method.ID) {
		boldMethod := boldMethodFromID(method.ID)
		url, err := h.bold.CreateLink(ctx, boldMethod, rec.MontoCOP, rec.ID,
			boldDescription(modalidad.ID, rec.Edad), rec.Email)
		if err != nil {
			h.logger.Error("bold link create failed", zap.Error(err))
		} else {
			checkoutURL = url
		}
	}

	go h.telegram.NotifyNewInscripcion(*rec, *modalidad, *method, checkoutURL)

	return c.JSON(CreateInscripcionResponse{
		ID:                  rec.ID,
		Status:              rec.Status,
		MontoCOP:            rec.MontoCOP,
		CheckoutURL:         checkoutURL,
		RequiresComprobante: method.ID == "transferencia",
	})
}

// Get handles GET /api/inscripciones/:id.
func (h *Handler) Get(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id required"})
	}
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()
	rec, err := h.repo.GetByID(ctx, id)
	if err != nil {
		h.logger.Error("get inscripcion", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "db error"})
	}
	if rec == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(InscripcionStatusResponse{
		ID:           rec.ID,
		Status:       rec.Status,
		MontoCOP:     rec.MontoCOP,
		FechaCurso:   rec.FechaCurso,
		MetodoPago:   rec.MetodoPago,
		Modalidad:    rec.Plan,
		NombrePiloto: rec.NombrePiloto,
	})
}

// UploadComprobante handles POST /api/inscripciones/:id/comprobante (multipart).
func (h *Handler) UploadComprobante(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id required"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 10*time.Second)
	defer cancel()

	rec, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "db error"})
	}
	if rec == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}

	fh, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}
	if fh.Size > 10*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "archivo supera 10 MB"})
	}

	path, err := h.saveReceipt(rec.ID, fh)
	if err != nil {
		h.logger.Error("save receipt", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.repo.UpdateComprobante(ctx, rec.ID, path, "comprobante recibido, en validación"); err != nil {
		h.logger.Error("update comprobante", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "db error"})
	}

	rec.ComprobantePath = path
	rec.Status = "comprobante recibido, en validación"
	mod := findModalidad(rec.Plan)
	met := findMetodo(rec.MetodoPago)
	if mod != nil && met != nil {
		go h.telegram.NotifyNewInscripcion(*rec, *mod, *met, "")
	}

	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handler) saveReceipt(id string, fh *multipart.FileHeader) (string, error) {
	if h.cfg.UploadsDir == "" {
		return "", errors.New("uploads dir not configured")
	}
	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext == "" {
		ext = ".bin"
	}
	out := filepath.Join(h.cfg.UploadsDir, id+ext)
	dst, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return out, nil
}
