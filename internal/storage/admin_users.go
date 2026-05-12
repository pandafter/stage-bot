package storage

import (
	"context"
	"fmt"
	"time"
)

type AdminUserRecord struct {
	ID           int64
	TenantID     string
	Username     string
	PasswordHash string
	Role         string
	LastLoginAt  *time.Time
	CreatedAt    time.Time
}

type AdminUsersRepo struct{ db *DB }

func NewAdminUsersRepo(db *DB) *AdminUsersRepo { return &AdminUsersRepo{db: db} }

func (r *AdminUsersRepo) GetByUsername(ctx context.Context, tenantID, username string) (*AdminUserRecord, error) {
	row := r.db.Pool.QueryRow(ctx,
		`SELECT id, tenant_id, username, password_hash, role, last_login_at, created_at
		 FROM admin_users WHERE tenant_id=$1 AND username=$2`,
		tenantID, username)
	var u AdminUserRecord
	if err := row.Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.Role, &u.LastLoginAt, &u.CreatedAt); err != nil {
		return nil, fmt.Errorf("admin_users.GetByUsername: %w", err)
	}
	return &u, nil
}

func (r *AdminUsersRepo) Create(ctx context.Context, tenantID, username, passwordHash, role string) (*AdminUserRecord, error) {
	var u AdminUserRecord
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO admin_users (tenant_id, username, password_hash, role)
		 VALUES ($1,$2,$3,$4) RETURNING id, tenant_id, username, password_hash, role, last_login_at, created_at`,
		tenantID, username, passwordHash, role,
	).Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.Role, &u.LastLoginAt, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("admin_users.Create: %w", err)
	}
	return &u, nil
}

func (r *AdminUsersRepo) UpdateLastLogin(ctx context.Context, id int64) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE admin_users SET last_login_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *AdminUsersRepo) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_users WHERE tenant_id=$1`, tenantID).Scan(&count)
	return count, err
}
