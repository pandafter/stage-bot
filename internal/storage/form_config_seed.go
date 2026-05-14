package storage

import (
	"context"
)

// SeedFormConfig inserts default dates, plans, and payment methods if the
// tables are empty for the given tenant. Safe to call on every startup.
func SeedFormConfig(ctx context.Context, repo *FormConfigRepo, tenantID string) error {
	if err := seedDates(ctx, repo, tenantID); err != nil {
		return err
	}
	if err := seedPlans(ctx, repo, tenantID); err != nil {
		return err
	}
	return seedMethods(ctx, repo, tenantID)
}

func seedDates(ctx context.Context, repo *FormConfigRepo, tenantID string) error {
	existing, err := repo.ListDates(ctx, tenantID)
	if err != nil || len(existing) > 0 {
		return err
	}
	cap10 := 10
	defaults := []struct {
		label    string
		startsOn string
	}{
		{"Mayo 9 y 10", "2026-05-09"},
		{"Mayo 23 y 24", "2026-05-23"},
	}
	for i, d := range defaults {
		_, err := repo.CreateDate(ctx, tenantID, d.label, d.startsOn, true, &cap10, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func seedPlans(ctx context.Context, repo *FormConfigRepo, tenantID string) error {
	existing, err := repo.ListPlans(ctx, tenantID)
	if err != nil || len(existing) > 0 {
		return err
	}
	plans := []PricingPlanRecord{
		{
			TenantID:    tenantID,
			Key:         "reserva",
			Name:        "Solo reserva del cupo",
			Description: "Reserva tu lugar y paga el resto más adelante.",
			PriceCOP:    150000,
			IsEnabled:   true,
			SortOrder:   0,
		},
		{
			TenantID:    tenantID,
			Key:         "completo",
			Name:        "Pago completo (preventa)",
			Description: "Precio especial de preventa. Incluye todo el curso.",
			PriceCOP:    730000,
			IsEnabled:   true,
			SortOrder:   1,
		},
	}
	for _, p := range plans {
		if _, err := repo.CreatePlan(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func seedMethods(ctx context.Context, repo *FormConfigRepo, tenantID string) error {
	existing, err := repo.ListMethods(ctx, tenantID)
	if err != nil || len(existing) > 0 {
		return err
	}
	methods := []struct {
		key          string
		label        string
		surchargePct float64
		sortOrder    int
	}{
		{"tarjeta", "Tarjeta de crédito / débito", 5, 0},
		{"nequi", "Nequi", 0, 1},
		{"bancolombia", "Botón Bancolombia", 0, 2},
		{"pse", "PSE — otros bancos", 0, 3},
		{"transferencia", "Transferencia manual + comprobante", 0, 4},
	}
	for _, m := range methods {
		_, err := repo.db.Pool.Exec(ctx,
			`INSERT INTO payment_methods (tenant_id, key, label, is_enabled, surcharge_pct, sort_order)
			 VALUES ($1,$2,$3,true,$4,$5)`,
			tenantID, m.key, m.label, m.surchargePct, m.sortOrder,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
