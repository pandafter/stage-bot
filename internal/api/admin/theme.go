package admin

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/storage"
)

type ThemeHandler struct {
	tenants *storage.TenantsRepo
	logger  *zap.Logger
}

func NewThemeHandler(tenants *storage.TenantsRepo, logger *zap.Logger) *ThemeHandler {
	return &ThemeHandler{tenants: tenants, logger: logger}
}

func (h *ThemeHandler) Get(c *fiber.Ctx) error {
	tenantID := TenantIDFromCtx(c)
	tenant, err := h.tenants.Get(c.Context(), tenantID)
	if err != nil {
		h.logger.Error("theme: get tenant", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	var theme map[string]any
	if err := json.Unmarshal(tenant.Theme, &theme); err != nil {
		theme = map[string]any{}
	}
	return c.JSON(theme)
}

func (h *ThemeHandler) Update(c *fiber.Ctx) error {
	tenantID := TenantIDFromCtx(c)
	var theme map[string]any
	if err := c.BodyParser(&theme); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	raw, err := json.Marshal(theme)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid theme"})
	}
	if err := h.tenants.UpdateTheme(c.Context(), tenantID, raw); err != nil {
		h.logger.Error("theme: update", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(theme)
}
