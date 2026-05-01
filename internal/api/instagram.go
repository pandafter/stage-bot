package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// IGPost is one Instagram media item returned to the frontend.
type IGPost struct {
	ID        string `json:"id"`
	MediaURL  string `json:"media_url"`
	Caption   string `json:"caption,omitempty"`
	Permalink string `json:"permalink"`
}

type igAPIResponse struct {
	Data []struct {
		ID        string `json:"id"`
		MediaType string `json:"media_type"`
		MediaURL  string `json:"media_url"`
		Thumbnail string `json:"thumbnail_url,omitempty"`
		Caption   string `json:"caption,omitempty"`
		Permalink string `json:"permalink"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

// GetInstagramPosts handles GET /api/instagram
// Returns up to 9 recent image posts from the configured Instagram account.
// Returns an empty array (never an error) when the token is not configured or
// the API call fails — frontend shows a static fallback in that case.
func (h *Handler) GetInstagramPosts(c *fiber.Ctx) error {
	token := h.cfg.InstagramAccessToken
	if token == "" {
		h.logger.Debug("instagram: no access token configured, returning empty")
		return c.JSON([]IGPost{})
	}

	client := &http.Client{Timeout: 8 * time.Second}
	url := fmt.Sprintf(
		"https://graph.instagram.com/me/media?fields=id,media_type,media_url,thumbnail_url,caption,permalink&limit=9&access_token=%s",
		token,
	)

	resp, err := client.Get(url)
	if err != nil {
		h.logger.Error("instagram: fetch failed", zap.Error(err))
		return c.JSON([]IGPost{})
	}
	defer resp.Body.Close()

	var igResp igAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&igResp); err != nil {
		h.logger.Error("instagram: decode failed", zap.Error(err))
		return c.JSON([]IGPost{})
	}

	if igResp.Error != nil {
		h.logger.Warn("instagram: API error", zap.String("msg", igResp.Error.Message), zap.Int("code", igResp.Error.Code))
		return c.JSON([]IGPost{})
	}

	posts := make([]IGPost, 0, len(igResp.Data))
	for _, m := range igResp.Data {
		if m.MediaType != "IMAGE" && m.MediaType != "CAROUSEL_ALBUM" {
			continue // skip videos without a proper image URL
		}
		mediaURL := m.MediaURL
		if mediaURL == "" {
			mediaURL = m.Thumbnail
		}
		if mediaURL == "" {
			continue
		}
		posts = append(posts, IGPost{
			ID:        m.ID,
			MediaURL:  mediaURL,
			Caption:   m.Caption,
			Permalink: m.Permalink,
		})
	}

	// 5-minute public cache — Instagram CDN URLs include their own expiry
	c.Set("Cache-Control", "public, max-age=300")
	return c.JSON(posts)
}
