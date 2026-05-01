package api

import "github.com/gofiber/fiber/v2"

// GetConfig exposes the catalog (modalidades, métodos, fechas, recargos).
func (h *Handler) GetConfig(c *fiber.Ctx) error {
	return c.JSON(ConfigResponse{
		Modalidades:      Modalidades,
		Metodos:          Metodos,
		Fechas:           Fechas,
		CardSurchargePct: CardSurchargePct,
		ReservaCOP:       ReservaCOP,
		PrecioCompleto:   PrecioCompleto,
		PrecioDescuento:  PrecioDescuento,
	})
}
