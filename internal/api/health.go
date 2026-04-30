package api

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// Health is the probe endpoint Railway uses to verify the container is up.
func Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}
