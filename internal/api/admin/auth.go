package admin

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/auth"
	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/storage"
)

type AuthHandler struct {
	cfg    *config.Config
	users  *storage.AdminUsersRepo
	logger *zap.Logger
}

func NewAuthHandler(cfg *config.Config, users *storage.AdminUsersRepo, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{cfg: cfg, users: users, logger: logger}
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "username and password required"})
	}

	tenantID := TenantIDFromCtx(c)
	user, err := h.users.GetByUsername(c.Context(), tenantID, req.Username)
	if err != nil {
		h.logger.Warn("admin login: user not found", zap.String("username", req.Username))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	if !auth.Verify(req.Password, user.PasswordHash) {
		h.logger.Warn("admin login: wrong password", zap.String("username", req.Username))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	token, err := auth.Sign(user.ID, user.TenantID, user.Role, h.cfg.JWTSecret)
	if err != nil {
		h.logger.Error("admin login: sign token", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	_ = h.users.UpdateLastLogin(c.Context(), user.ID)

	return c.JSON(LoginResponse{
		Token: token,
		User: AdminUser{
			ID:       user.ID,
			TenantID: user.TenantID,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	adminID := AdminIDFromCtx(c)
	tenantID := TenantIDFromCtx(c)
	return c.JSON(fiber.Map{
		"admin_id":  adminID,
		"tenant_id": tenantID,
		"role":      c.Locals("admin_role"),
	})
}
