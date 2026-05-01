package api

// Dev-only endpoints — only registered when ENV=development.
// They let you simulate payment events without going through Bold.

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// DevConfirmPayment simulates a Bold SALE_APPROVED webhook for the given
// inscription ID. Useful for end-to-end testing without a real payment.
//
// POST /dev/inscripciones/:id/confirm
func (h *Handler) DevConfirmPayment(c *fiber.Ctx) error {
	id := c.Params("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rec, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "db error"})
	}
	if rec == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "inscripcion not found"})
	}

	newStatus := "pagado"
	if err := h.repo.UpdateStatus(ctx, id, newStatus); err != nil {
		h.logger.Error("dev confirm: update status", zap.Error(err), zap.String("id", id))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "db error"})
	}

	tsLabel := time.Now().Format("2006-01-02 15:04:05")
	go h.telegram.NotifyBoldEvent(
		"SALE_APPROVED",
		"DEV-TX-"+id,
		"DEV-PAY-"+id,
		"Simulación desarrollo",
		tsLabel,
		newStatus,
		rec.MontoCOP,
		rec,
		id,
	)

	h.logger.Info("dev: payment confirmed", zap.String("id", id), zap.Int("monto", rec.MontoCOP))
	return c.JSON(fiber.Map{
		"ok":     true,
		"id":     id,
		"status": newStatus,
		"monto":  rec.MontoCOP,
	})
}

// DevRejectPayment simulates a Bold SALE_REJECTED for the given inscription.
//
// POST /dev/inscripciones/:id/reject
func (h *Handler) DevRejectPayment(c *fiber.Ctx) error {
	id := c.Params("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rec, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "db error"})
	}
	if rec == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "inscripcion not found"})
	}

	newStatus := "pago rechazado"
	if err := h.repo.UpdateStatus(ctx, id, newStatus); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "db error"})
	}

	tsLabel := time.Now().Format("2006-01-02 15:04:05")
	go h.telegram.NotifyBoldEvent(
		"SALE_REJECTED",
		"DEV-TX-"+id,
		"DEV-PAY-"+id,
		"Simulación desarrollo",
		tsLabel,
		newStatus,
		rec.MontoCOP,
		rec,
		id,
	)

	h.logger.Info("dev: payment rejected", zap.String("id", id))
	return c.JSON(fiber.Map{"ok": true, "id": id, "status": newStatus})
}
