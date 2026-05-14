package media

import "github.com/kart-academy/instagram-bot/internal/config"

func NewFromConfig(cfg *config.Config) (Store, error) {
	if cfg.R2Bucket != "" && cfg.R2AccountID != "" {
		return NewR2Store(cfg.R2AccountID, cfg.R2AccessKey, cfg.R2SecretKey, cfg.R2Bucket, cfg.R2PublicURL)
	}
	publicBase := cfg.PublicURL + "/uploads"
	return NewFSStore(cfg.UploadsDir, publicBase), nil
}
