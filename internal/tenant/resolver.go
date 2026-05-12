package tenant

import (
	"github.com/gofiber/fiber/v2"
)

// Middleware inyecta el tenant_id en c.Locals("tenant_id").
// V1: siempre usa el tenant por defecto desde config.
// V1.1: leerá el Host header y resolverá por dominio en BD.
func Middleware(defaultTenant string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("tenant_id", defaultTenant)
		return c.Next()
	}
}

func FromCtx(c *fiber.Ctx) string {
	if t, ok := c.Locals("tenant_id").(string); ok && t != "" {
		return t
	}
	return "st4ge"
}
