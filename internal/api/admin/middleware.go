package admin

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/kart-academy/instagram-bot/internal/auth"
	"github.com/kart-academy/instagram-bot/internal/config"
)

func RequireJWT(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.VerifyToken(tokenStr, cfg.JWTSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		c.Locals("admin_id", claims.AdminID)
		c.Locals("tenant_id", claims.TenantID)
		c.Locals("admin_role", claims.Role)
		return c.Next()
	}
}

func AdminIDFromCtx(c *fiber.Ctx) int64 {
	v, _ := c.Locals("admin_id").(int64)
	return v
}

func TenantIDFromCtx(c *fiber.Ctx) string {
	if t, ok := c.Locals("tenant_id").(string); ok && t != "" {
		return t
	}
	return "st4ge"
}
