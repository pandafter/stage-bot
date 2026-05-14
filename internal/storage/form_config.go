package storage

import (
	"context"
	"encoding/json"
	"time"
)

// ── Form dates ───────────────────────────────────────────────────

type FormDateRecord struct {
	ID        int64
	TenantID  string
	Label     string
	StartsOn  string // "2006-01-02"
	IsEnabled bool
	Capacity  *int
	SortOrder int
	CreatedAt time.Time
}

type FormConfigRepo struct{ db *DB }

func NewFormConfigRepo(db *DB) *FormConfigRepo { return &FormConfigRepo{db: db} }

func (r *FormConfigRepo) ListDates(ctx context.Context, tenantID string) ([]FormDateRecord, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, tenant_id, label, starts_on::TEXT, is_enabled, capacity, sort_order, created_at
		 FROM form_dates WHERE tenant_id=$1 ORDER BY sort_order, starts_on`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FormDateRecord
	for rows.Next() {
		var d FormDateRecord
		if err := rows.Scan(&d.ID, &d.TenantID, &d.Label, &d.StartsOn, &d.IsEnabled, &d.Capacity, &d.SortOrder, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *FormConfigRepo) CreateDate(ctx context.Context, tenantID, label, startsOn string, isEnabled bool, capacity *int, sortOrder int) (*FormDateRecord, error) {
	var d FormDateRecord
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO form_dates (tenant_id, label, starts_on, is_enabled, capacity, sort_order)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING id, tenant_id, label, starts_on::TEXT, is_enabled, capacity, sort_order, created_at`,
		tenantID, label, startsOn, isEnabled, capacity, sortOrder,
	).Scan(&d.ID, &d.TenantID, &d.Label, &d.StartsOn, &d.IsEnabled, &d.Capacity, &d.SortOrder, &d.CreatedAt)
	return &d, err
}

func (r *FormConfigRepo) UpdateDate(ctx context.Context, id int64, tenantID string, fields map[string]any) error {
	// Build partial update — only passed fields
	// Simplification: accept all fields, use COALESCE pattern
	label, _ := fields["label"].(string)
	startsOn, _ := fields["starts_on"].(string)
	isEnabled, _ := fields["is_enabled"].(bool)
	sortOrder, _ := fields["sort_order"].(int)

	_, err := r.db.Pool.Exec(ctx,
		`UPDATE form_dates SET
		  label=COALESCE(NULLIF($3,''), label),
		  starts_on=COALESCE(NULLIF($4,'')::date, starts_on),
		  is_enabled=$5,
		  sort_order=$6
		 WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, label, startsOn, isEnabled, sortOrder,
	)
	return err
}

func (r *FormConfigRepo) DeleteDate(ctx context.Context, id int64, tenantID string) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM form_dates WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	return err
}

// ── Pricing plans ─────────────────────────────────────────────────

type PricingPlanRecord struct {
	ID          int64
	TenantID    string
	Key         string
	Name        string
	Description string
	PriceCOP    int
	Features    []string
	IsEnabled   bool
	SortOrder   int
}

func (r *FormConfigRepo) ListPlans(ctx context.Context, tenantID string) ([]PricingPlanRecord, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, tenant_id, key, name, description, price_cop, features, is_enabled, sort_order
		 FROM pricing_plans WHERE tenant_id=$1 ORDER BY sort_order`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PricingPlanRecord
	for rows.Next() {
		var p PricingPlanRecord
		var featRaw []byte
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Key, &p.Name, &p.Description, &p.PriceCOP, &featRaw, &p.IsEnabled, &p.SortOrder); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(featRaw, &p.Features)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *FormConfigRepo) CreatePlan(ctx context.Context, p PricingPlanRecord) (*PricingPlanRecord, error) {
	raw, _ := json.Marshal(p.Features)
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO pricing_plans (tenant_id, key, name, description, price_cop, features, is_enabled, sort_order)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id, tenant_id, key, name, description, price_cop, features, is_enabled, sort_order`,
		p.TenantID, p.Key, p.Name, p.Description, p.PriceCOP, raw, p.IsEnabled, p.SortOrder,
	).Scan(&p.ID, &p.TenantID, &p.Key, &p.Name, &p.Description, &p.PriceCOP, &raw, &p.IsEnabled, &p.SortOrder)
	_ = json.Unmarshal(raw, &p.Features)
	return &p, err
}

func (r *FormConfigRepo) UpdatePlan(ctx context.Context, id int64, tenantID string, fields map[string]any) error {
	raw, _ := json.Marshal(fields["features"])
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE pricing_plans SET
		  name=COALESCE(NULLIF($3,''), name),
		  description=COALESCE($4, description),
		  price_cop=COALESCE($5, price_cop),
		  features=COALESCE($6, features),
		  is_enabled=$7,
		  sort_order=$8
		 WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		fields["name"], fields["description"], fields["price_cop"], raw,
		fields["is_enabled"], fields["sort_order"],
	)
	return err
}

func (r *FormConfigRepo) DeletePlan(ctx context.Context, id int64, tenantID string) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM pricing_plans WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	return err
}

// ── Payment methods ───────────────────────────────────────────────

type PaymentMethodRecord struct {
	ID           int64
	TenantID     string
	Key          string
	Label        string
	IsEnabled    bool
	SurchargePct float64
	SortOrder    int
}

func (r *FormConfigRepo) ListMethods(ctx context.Context, tenantID string) ([]PaymentMethodRecord, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, tenant_id, key, label, is_enabled, surcharge_pct, sort_order
		 FROM payment_methods WHERE tenant_id=$1 ORDER BY sort_order`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PaymentMethodRecord
	for rows.Next() {
		var m PaymentMethodRecord
		if err := rows.Scan(&m.ID, &m.TenantID, &m.Key, &m.Label, &m.IsEnabled, &m.SurchargePct, &m.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *FormConfigRepo) UpdateMethod(ctx context.Context, id int64, tenantID string, fields map[string]any) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE payment_methods SET
		  label=COALESCE(NULLIF($3,''), label),
		  is_enabled=$4,
		  surcharge_pct=$5,
		  sort_order=$6
		 WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		fields["label"], fields["is_enabled"], fields["surcharge_pct"], fields["sort_order"],
	)
	return err
}
