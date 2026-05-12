package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

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
