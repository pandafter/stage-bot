package media

import (
	"context"
	"io"
)

// Store abstracts file storage for media assets.
// V1 supports filesystem or Cloudflare R2.
type Store interface {
	Put(ctx context.Context, key string, r io.Reader, mimeType string, size int64) (url string, err error)
	Delete(ctx context.Context, key string) error
	URL(key string) string
}
