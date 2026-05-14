package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ── Media records ─────────────────────────────────────────────────

type MediaRecord struct {
	ID         int64
	TenantID   string
	Filename   string
	URL        string
	StorageKey string
	MimeType   string
	SizeBytes  int64
	AltText    string
	Tags       []string
	CreatedAt  time.Time
}

type CMSSectionRecord struct {
	TenantID    string
	SectionKey  string
	Data        map[string]any
	IsPublished bool
	Version     int
	UpdatedAt   time.Time
}

type CMSRepo struct{ db *DB }

func NewCMSRepo(db *DB) *CMSRepo { return &CMSRepo{db: db} }

func (r *CMSRepo) GetSections(ctx context.Context, tenantID string) ([]CMSSectionRecord, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT tenant_id, section_key, data, is_published, version, updated_at
		 FROM cms_sections WHERE tenant_id=$1 ORDER BY section_key`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("cms.GetSections: %w", err)
	}
	defer rows.Close()

	var sections []CMSSectionRecord
	for rows.Next() {
		var s CMSSectionRecord
		var raw []byte
		if err := rows.Scan(&s.TenantID, &s.SectionKey, &raw, &s.IsPublished, &s.Version, &s.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &s.Data); err != nil {
			return nil, err
		}
		sections = append(sections, s)
	}
	return sections, rows.Err()
}

func (r *CMSRepo) GetSection(ctx context.Context, tenantID, key string) (*CMSSectionRecord, error) {
	var s CMSSectionRecord
	var raw []byte
	err := r.db.Pool.QueryRow(ctx,
		`SELECT tenant_id, section_key, data, is_published, version, updated_at
		 FROM cms_sections WHERE tenant_id=$1 AND section_key=$2`, tenantID, key,
	).Scan(&s.TenantID, &s.SectionKey, &raw, &s.IsPublished, &s.Version, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("cms.GetSection: %w", err)
	}
	if err := json.Unmarshal(raw, &s.Data); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *CMSRepo) UpsertSection(ctx context.Context, tenantID, key string, data map[string]any, isPublished bool, adminID int64) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("cms.UpsertSection marshal: %w", err)
	}
	_, err = r.db.Pool.Exec(ctx,
		`INSERT INTO cms_sections (tenant_id, section_key, data, is_published, updated_by, updated_at)
		 VALUES ($1,$2,$3,$4,$5,NOW())
		 ON CONFLICT (tenant_id, section_key) DO UPDATE
		 SET data=$3, is_published=$4, updated_by=$5, version=cms_sections.version+1, updated_at=NOW()`,
		tenantID, key, raw, isPublished, adminID,
	)
	return err
}

func (r *CMSRepo) MaxVersion(ctx context.Context, tenantID string) (int, error) {
	var v int
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(version),0) FROM cms_sections WHERE tenant_id=$1`, tenantID).Scan(&v)
	return v, err
}

func (r *CMSRepo) ListMedia(ctx context.Context, tenantID, tag string) ([]MediaRecord, error) {
	query := `SELECT id, tenant_id, filename, url, storage_key, mime_type, size_bytes, alt_text, tags, created_at
	           FROM cms_media WHERE tenant_id=$1`
	args := []any{tenantID}
	if tag != "" {
		query += ` AND $2=ANY(tags)`
		args = append(args, tag)
	}
	query += ` ORDER BY created_at DESC LIMIT 100`

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MediaRecord
	for rows.Next() {
		var m MediaRecord
		if err := rows.Scan(&m.ID, &m.TenantID, &m.Filename, &m.URL, &m.StorageKey, &m.MimeType, &m.SizeBytes, &m.AltText, &m.Tags, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *CMSRepo) CreateMedia(ctx context.Context, tenantID, filename, url, storageKey, mimeType string, sizeBytes int64, altText string) (*MediaRecord, error) {
	var m MediaRecord
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO cms_media (tenant_id, filename, url, storage_key, mime_type, size_bytes, alt_text)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING id, tenant_id, filename, url, storage_key, mime_type, size_bytes, alt_text, tags, created_at`,
		tenantID, filename, url, storageKey, mimeType, sizeBytes, altText,
	).Scan(&m.ID, &m.TenantID, &m.Filename, &m.URL, &m.StorageKey, &m.MimeType, &m.SizeBytes, &m.AltText, &m.Tags, &m.CreatedAt)
	return &m, err
}

func (r *CMSRepo) GetMedia(ctx context.Context, tenantID string, id int64) (*MediaRecord, error) {
	var m MediaRecord
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, tenant_id, filename, url, storage_key, mime_type, size_bytes, alt_text, tags, created_at
		 FROM cms_media WHERE tenant_id=$1 AND id=$2`, tenantID, id,
	).Scan(&m.ID, &m.TenantID, &m.Filename, &m.URL, &m.StorageKey, &m.MimeType, &m.SizeBytes, &m.AltText, &m.Tags, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *CMSRepo) DeleteMedia(ctx context.Context, tenantID string, id int64) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM cms_media WHERE tenant_id=$1 AND id=$2`, tenantID, id)
	return err
}

func (r *CMSRepo) UpdateMedia(ctx context.Context, tenantID string, id int64, altText string, tags []string) error {
	if tags == nil {
		tags = []string{}
	}
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE cms_media SET alt_text=$3, tags=$4 WHERE tenant_id=$1 AND id=$2`,
		tenantID, id, altText, tags)
	return err
}
