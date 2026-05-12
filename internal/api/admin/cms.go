package admin

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/storage"
)

var validSectionKeys = map[string]bool{
	"hero": true, "stats": true, "program": true,
	"instructores": true, "gallery": true, "faq": true, "cta": true,
}

type CMSHandler struct {
	cms    *storage.CMSRepo
	logger *zap.Logger
}

func NewCMSHandler(cms *storage.CMSRepo, logger *zap.Logger) *CMSHandler {
	return &CMSHandler{cms: cms, logger: logger}
}

func (h *CMSHandler) GetSections(c *fiber.Ctx) error {
	tenantID := TenantIDFromCtx(c)
	sections, err := h.cms.GetSections(c.Context(), tenantID)
	if err != nil {
		h.logger.Error("cms: get sections", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	resp := make([]SectionResponse, 0, len(sections))
	for _, s := range sections {
		resp = append(resp, SectionResponse{
			TenantID:    s.TenantID,
			SectionKey:  s.SectionKey,
			Data:        s.Data,
			IsPublished: s.IsPublished,
			Version:     s.Version,
			UpdatedAt:   s.UpdatedAt,
		})
	}
	return c.JSON(resp)
}

func (h *CMSHandler) GetSection(c *fiber.Ctx) error {
	key := c.Params("key")
	if !validSectionKeys[key] {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "section not found"})
	}
	tenantID := TenantIDFromCtx(c)
	s, err := h.cms.GetSection(c.Context(), tenantID, key)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "section not found"})
	}
	return c.JSON(SectionResponse{
		TenantID:    s.TenantID,
		SectionKey:  s.SectionKey,
		Data:        s.Data,
		IsPublished: s.IsPublished,
		Version:     s.Version,
		UpdatedAt:   s.UpdatedAt,
	})
}

func (h *CMSHandler) UpsertSection(c *fiber.Ctx) error {
	key := c.Params("key")
	if !validSectionKeys[key] {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "section not found"})
	}

	var req UpsertSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if req.Data == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data is required"})
	}

	isPublished := true
	if req.IsPublished != nil {
		isPublished = *req.IsPublished
	}

	tenantID := TenantIDFromCtx(c)
	adminID := AdminIDFromCtx(c)

	if err := h.cms.UpsertSection(c.Context(), tenantID, key, req.Data, isPublished, adminID); err != nil {
		h.logger.Error("cms: upsert section", zap.String("key", key), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	s, _ := h.cms.GetSection(c.Context(), tenantID, key)
	if s == nil {
		return c.JSON(fiber.Map{"ok": true})
	}
	return c.JSON(SectionResponse{
		TenantID:    s.TenantID,
		SectionKey:  s.SectionKey,
		Data:        s.Data,
		IsPublished: s.IsPublished,
		Version:     s.Version,
		UpdatedAt:   s.UpdatedAt,
	})
}
