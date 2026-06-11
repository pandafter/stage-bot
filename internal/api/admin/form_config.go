package admin

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/storage"
)

type FormConfigHandler struct {
	repo   *storage.FormConfigRepo
	logger *zap.Logger
}

func NewFormConfigHandler(repo *storage.FormConfigRepo, logger *zap.Logger) *FormConfigHandler {
	return &FormConfigHandler{repo: repo, logger: logger}
}

// ── Dates ─────────────────────────────────────────────────────────

func (h *FormConfigHandler) ListDates(c *fiber.Ctx) error {
	tenantID := TenantIDFromCtx(c)
	dates, err := h.repo.ListDates(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	resp := make([]FormDate, len(dates))
	for i, d := range dates {
		resp[i] = formDateToDTO(d)
	}
	return c.JSON(resp)
}

func (h *FormConfigHandler) CreateDate(c *fiber.Ctx) error {
	var req CreateFormDateRequest
	if err := c.BodyParser(&req); err != nil || req.Label == "" || req.StartsOn == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "label and starts_on required"})
	}
	tenantID := TenantIDFromCtx(c)
	d, err := h.repo.CreateDate(c.Context(), tenantID, req.Label, req.StartsOn, req.IsEnabled, req.Capacity, req.SortOrder)
	if err != nil {
		h.logger.Error("form: create date", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.Status(fiber.StatusCreated).JSON(formDateToDTO(*d))
}

func (h *FormConfigHandler) UpdateDate(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var req UpdateFormDateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	tenantID := TenantIDFromCtx(c)
	fields := map[string]any{}
	if req.Label != nil {
		fields["label"] = *req.Label
	}
	if req.StartsOn != nil {
		fields["starts_on"] = *req.StartsOn
	}
	if req.IsEnabled != nil {
		fields["is_enabled"] = *req.IsEnabled
	}
	if req.SortOrder != nil {
		fields["sort_order"] = *req.SortOrder
	}
	if req.Capacity != nil {
		fields["capacity"] = *req.Capacity
	}
	if err := h.repo.UpdateDate(c.Context(), id, tenantID, fields); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *FormConfigHandler) DeleteDate(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	tenantID := TenantIDFromCtx(c)
	if err := h.repo.DeleteDate(c.Context(), id, tenantID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *FormConfigHandler) ReorderDates(c *fiber.Ctx) error {
	var req ReorderRequest
	if err := c.BodyParser(&req); err != nil || len(req.IDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ids required"})
	}
	tenantID := TenantIDFromCtx(c)
	for i, id := range req.IDs {
		fields := map[string]any{"sort_order": i}
		_ = h.repo.UpdateDate(c.Context(), id, tenantID, fields)
	}
	return c.JSON(fiber.Map{"ok": true})
}

// ── Plans ──────────────────────────────────────────────────────────

func (h *FormConfigHandler) ListPlans(c *fiber.Ctx) error {
	tenantID := TenantIDFromCtx(c)
	plans, err := h.repo.ListPlans(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	resp := make([]PricingPlan, len(plans))
	for i, p := range plans {
		resp[i] = pricingPlanToDTO(p)
	}
	return c.JSON(resp)
}

func (h *FormConfigHandler) CreatePlan(c *fiber.Ctx) error {
	var req CreatePlanRequest
	if err := c.BodyParser(&req); err != nil || req.Key == "" || req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "key and name required"})
	}
	tenantID := TenantIDFromCtx(c)
	record := storage.PricingPlanRecord{
		TenantID:    tenantID,
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		PriceCOP:    req.PriceCOP,
		Features:    req.Features,
		BoldLink:    req.BoldLink,
		IsEnabled:   req.IsEnabled,
		SortOrder:   req.SortOrder,
	}
	p, err := h.repo.CreatePlan(c.Context(), record)
	if err != nil {
		h.logger.Error("form: create plan", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.Status(fiber.StatusCreated).JSON(pricingPlanToDTO(*p))
}

func (h *FormConfigHandler) UpdatePlan(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	tenantID := TenantIDFromCtx(c)
	var fields map[string]any
	if err := c.BodyParser(&fields); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := h.repo.UpdatePlan(c.Context(), id, tenantID, fields); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *FormConfigHandler) DeletePlan(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	tenantID := TenantIDFromCtx(c)
	if err := h.repo.DeletePlan(c.Context(), id, tenantID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ── Methods ────────────────────────────────────────────────────────

func (h *FormConfigHandler) ListMethods(c *fiber.Ctx) error {
	tenantID := TenantIDFromCtx(c)
	methods, err := h.repo.ListMethods(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	resp := make([]PaymentMethod, len(methods))
	for i, m := range methods {
		resp[i] = paymentMethodToDTO(m)
	}
	return c.JSON(resp)
}

func (h *FormConfigHandler) UpdateMethod(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	tenantID := TenantIDFromCtx(c)
	var fields map[string]any
	if err := c.BodyParser(&fields); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := h.repo.UpdateMethod(c.Context(), id, tenantID, fields); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// ── Helpers ────────────────────────────────────────────────────────

func formDateToDTO(d storage.FormDateRecord) FormDate {
	return FormDate{
		ID:        d.ID,
		TenantID:  d.TenantID,
		Label:     d.Label,
		StartsOn:  d.StartsOn,
		IsEnabled: d.IsEnabled,
		Capacity:  d.Capacity,
		SortOrder: d.SortOrder,
		CreatedAt: d.CreatedAt,
	}
}

func pricingPlanToDTO(p storage.PricingPlanRecord) PricingPlan {
	return PricingPlan{
		ID:          p.ID,
		TenantID:    p.TenantID,
		Key:         p.Key,
		Name:        p.Name,
		Description: p.Description,
		PriceCOP:    p.PriceCOP,
		Features:    p.Features,
		BoldLink:    p.BoldLink,
		IsEnabled:   p.IsEnabled,
		SortOrder:   p.SortOrder,
	}
}

func paymentMethodToDTO(m storage.PaymentMethodRecord) PaymentMethod {
	return PaymentMethod{
		ID:           m.ID,
		TenantID:     m.TenantID,
		Key:          m.Key,
		Label:        m.Label,
		IsEnabled:    m.IsEnabled,
		SurchargePct: m.SurchargePct,
		SortOrder:    m.SortOrder,
	}
}
