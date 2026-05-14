package admin

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/media"
	"github.com/kart-academy/instagram-bot/internal/storage"
)

const maxUploadBytes = 5 * 1024 * 1024 // 5 MB

type MediaHandler struct {
	store  media.Store
	repo   *storage.CMSRepo
	logger *zap.Logger
}

func NewMediaHandler(store media.Store, repo *storage.CMSRepo, logger *zap.Logger) *MediaHandler {
	return &MediaHandler{store: store, repo: repo, logger: logger}
}

func (h *MediaHandler) List(c *fiber.Ctx) error {
	tenantID := TenantIDFromCtx(c)
	items, err := h.repo.ListMedia(c.Context(), tenantID, c.Query("tag"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	if items == nil {
		items = []storage.MediaRecord{}
	}
	resp := make([]MediaItem, len(items))
	for i, m := range items {
		resp[i] = mediaRecordToDTO(m)
	}
	return c.JSON(resp)
}

func (h *MediaHandler) Upload(c *fiber.Ctx) error {
	tenantID := TenantIDFromCtx(c)

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file required"})
	}
	if file.Size > maxUploadBytes {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file too large (max 5MB)"})
	}

	mimeType := file.Header.Get("Content-Type")
	if !strings.HasPrefix(mimeType, "image/") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "only images allowed"})
	}

	ext := filepath.Ext(file.Filename)
	if ext == "" {
		exts, _ := mime.ExtensionsByType(mimeType)
		if len(exts) > 0 {
			ext = exts[0]
		}
	}

	key := fmt.Sprintf("%s/%d%s", tenantID, time.Now().UnixNano(), ext)

	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot open file"})
	}
	defer f.Close()

	url, err := h.store.Put(c.Context(), key, f, mimeType, file.Size)
	if err != nil {
		h.logger.Error("media: upload", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "upload failed"})
	}

	altText := c.FormValue("alt_text")
	record, err := h.repo.CreateMedia(c.Context(), tenantID, file.Filename, url, key, mimeType, file.Size, altText)
	if err != nil {
		h.logger.Error("media: save record", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(mediaRecordToDTO(*record))
}

func (h *MediaHandler) Delete(c *fiber.Ctx) error {
	tenantID := TenantIDFromCtx(c)
	idStr := c.Params("id")
	var id int64
	if _, err := fmt.Sscan(idStr, &id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	record, err := h.repo.GetMedia(c.Context(), tenantID, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}

	if err := h.store.Delete(c.Context(), record.StorageKey); err != nil {
		h.logger.Warn("media: delete from store", zap.Error(err))
	}

	if err := h.repo.DeleteMedia(c.Context(), tenantID, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *MediaHandler) Update(c *fiber.Ctx) error {
	tenantID := TenantIDFromCtx(c)
	idStr := c.Params("id")
	var id int64
	if _, err := fmt.Sscan(idStr, &id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req UpdateMediaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.repo.UpdateMedia(c.Context(), tenantID, id, req.AltText, req.Tags); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	record, _ := h.repo.GetMedia(c.Context(), tenantID, id)
	if record == nil {
		return c.JSON(fiber.Map{"ok": true})
	}
	return c.JSON(mediaRecordToDTO(*record))
}

func mediaRecordToDTO(m storage.MediaRecord) MediaItem {
	tags := m.Tags
	if tags == nil {
		tags = []string{}
	}
	return MediaItem{
		ID:         m.ID,
		Filename:   m.Filename,
		URL:        m.URL,
		StorageKey: m.StorageKey,
		MimeType:   m.MimeType,
		SizeBytes:  m.SizeBytes,
		AltText:    m.AltText,
		Tags:       tags,
		CreatedAt:  m.CreatedAt,
	}
}
