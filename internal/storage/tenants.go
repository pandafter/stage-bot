package storage

import (
	"context"
	"fmt"
)

type TenantsRepo struct{ db *DB }

func NewTenantsRepo(db *DB) *TenantsRepo { return &TenantsRepo{db: db} }

type Tenant struct {
	ID       string
	Name     string
	Domain   string
	Theme    []byte // raw JSONB
	Features []byte // raw JSONB
}

func (r *TenantsRepo) Get(ctx context.Context, id string) (*Tenant, error) {
	row := r.db.Pool.QueryRow(ctx,
		`SELECT id, name, COALESCE(domain,''), theme, features FROM tenants WHERE id=$1`, id)
	var t Tenant
	if err := row.Scan(&t.ID, &t.Name, &t.Domain, &t.Theme, &t.Features); err != nil {
		return nil, fmt.Errorf("tenants.Get: %w", err)
	}
	return &t, nil
}

func (r *TenantsRepo) UpdateTheme(ctx context.Context, id string, theme []byte) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE tenants SET theme=$1, updated_at=NOW() WHERE id=$2`, theme, id)
	return err
}

// Unused in V1 — prepared for multi-tenant resolution by domain
func (r *TenantsRepo) GetByDomain(ctx context.Context, domain string) (*Tenant, error) {
	row := r.db.Pool.QueryRow(ctx,
		`SELECT id, name, COALESCE(domain,''), theme, features FROM tenants WHERE domain=$1`, domain)
	var t Tenant
	if err := row.Scan(&t.ID, &t.Name, &t.Domain, &t.Theme, &t.Features); err != nil {
		return nil, fmt.Errorf("tenants.GetByDomain: %w", err)
	}
	return &t, nil
}
