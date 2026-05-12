package admin

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/storage"
)

type InscripcionesAdminHandler struct {
	repo   *storage.InscripcionesRepo
	logger *zap.Logger
}

func NewInscripcionesAdminHandler(repo *storage.InscripcionesRepo, logger *zap.Logger) *InscripcionesAdminHandler {
	return &InscripcionesAdminHandler{repo: repo, logger: logger}
}

func (h *InscripcionesAdminHandler) List(c *fiber.Ctx) error {
	tenantID := TenantIDFromCtx(c)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	items, total, err := h.repo.List(
		c.Context(),
		tenantID,
		c.Query("status"),
		c.Query("date_from"),
		c.Query("date_to"),
		c.Query("plan"),
		c.Query("search"),
		page,
		limit,
	)
	if err != nil {
		h.logger.Error("admin: list inscripciones", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	if items == nil {
		items = []storage.InscripcionRecord{}
	}
	return c.JSON(fiber.Map{
		"items": items,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *InscripcionesAdminHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	ins, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	if ins == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(ins)
}

func (h *InscripcionesAdminHandler) UpdateStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	var req UpdateInscripcionStatusRequest
	if err := c.BodyParser(&req); err != nil || req.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status required"})
	}

	validStatuses := map[string]bool{
		"pendiente": true, "pagado": true, "comprobante_pendiente": true,
		"rechazado": true, "cancelado": true,
	}
	if !validStatuses[req.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
	}

	if err := h.repo.UpdateStatus(c.Context(), id, req.Status); err != nil {
		h.logger.Error("admin: update inscripcion status", zap.String("id", id), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(fiber.Map{"ok": true, "id": id, "status": req.Status})
}

func (h *InscripcionesAdminHandler) ExportCSV(c *fiber.Ctx) error {
	tenantID := TenantIDFromCtx(c)
	items, _, err := h.repo.List(c.Context(), tenantID, "", "", "", "", "", 1, 1000)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	_ = w.Write([]string{
		"ID", "Email", "Nombre", "Plan", "Método Pago", "Monto COP",
		"Fecha Curso", "Status", "Teléfono", "Ciudad", "Documento", "Creado",
	})

	for _, ins := range items {
		_ = w.Write([]string{
			ins.ID, ins.Email, ins.NombrePiloto, ins.Plan, ins.MetodoPago,
			strconv.Itoa(ins.MontoCOP), ins.FechaCurso, ins.Status,
			ins.Telefono, ins.Ciudad, ins.NumeroDocumento,
			ins.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()

	filename := fmt.Sprintf("inscripciones_%s.csv", time.Now().Format("2006-01-02"))
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.Send(buf.Bytes())
}
