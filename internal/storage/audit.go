package storage

import (
	"context"
	"encoding/json"
)

type AuditRepo struct {
	db *DB
}

func NewAuditRepo(db *DB) *AuditRepo {
	return &AuditRepo{db: db}
}

func (r *AuditRepo) Log(ctx context.Context, tenantID string, adminID int64, action, entity, entityID string, diff any) {
	var raw []byte
	if diff != nil {
		raw, _ = json.Marshal(diff)
	}
	_, _ = r.db.Pool.Exec(ctx,
		`INSERT INTO admin_audit_log (tenant_id, admin_id, action, entity, entity_id, diff)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		tenantID, adminID, action, entity, entityID, raw,
	)
}
