package api

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

// GetConfig exposes the catalog (modalidades, métodos, fechas, recargos).
// If a CMS repo is attached, published sections are included under the "cms" key.
func (h *Handler) GetConfig(c *fiber.Ctx) error {
	// Fetch dates from DB when available; fall back to hardcoded defaults.
	fechas := Fechas
	if h.formConfigRepo != nil {
		if dates, err := h.formConfigRepo.ListDates(c.Context(), h.defaultTenant); err == nil {
			var dbLabels []string
			for _, d := range dates {
				if d.IsEnabled {
					dbLabels = append(dbLabels, d.Label)
				}
			}
			if len(dbLabels) > 0 {
				fechas = dbLabels
			}
		}
	}

	// Fetch plans from DB when available; fall back to hardcoded defaults.
	modalidades := Modalidades
	if h.formConfigRepo != nil {
		if plans, err := h.formConfigRepo.ListPlans(c.Context(), h.defaultTenant); err == nil {
			var dbModalidades []Modalidad
			for _, p := range plans {
				if p.IsEnabled {
					dbModalidades = append(dbModalidades, Modalidad{ID: p.Key, Label: p.Name, PriceCOP: p.PriceCOP})
				}
			}
			if len(dbModalidades) > 0 {
				modalidades = dbModalidades
			}
		}
	}

	// Fetch payment methods from DB; fall back to hardcoded defaults.
	metodos := Metodos
	cardSurchargePct := CardSurchargePct
	if h.formConfigRepo != nil {
		if methods, err := h.formConfigRepo.ListMethods(c.Context(), h.defaultTenant); err == nil {
			var dbMetodos []PaymentMethod
			for _, m := range methods {
				if m.IsEnabled {
					dbMetodos = append(dbMetodos, PaymentMethod{ID: m.Key, Label: m.Label})
					if m.Key == "tarjeta" {
						cardSurchargePct = int(m.SurchargePct)
					}
				}
			}
			if len(dbMetodos) > 0 {
				metodos = dbMetodos
			}
		}
	}

	response := fiber.Map{
		"modalidades":        modalidades,
		"metodos":            metodos,
		"fechas":             fechas,
		"card_surcharge_pct": cardSurchargePct,
		"reserva_cop":        ReservaCOP,
		"precio_completo":    PrecioCompleto,
		"precio_descuento":   PrecioDescuento,
	}

	if h.cmsRepo != nil {
		sections, err := h.cmsRepo.GetSections(c.Context(), h.defaultTenant)
		if err == nil && len(sections) > 0 {
			cmsData := make(map[string]any, len(sections))
			for _, s := range sections {
				if s.IsPublished {
					cmsData[s.SectionKey] = s.Data
				}
			}
			if len(cmsData) > 0 {
				response["cms"] = cmsData
			}
		}
	}

	if h.tenantsRepo != nil {
		tenant, err := h.tenantsRepo.Get(c.Context(), h.defaultTenant)
		if err == nil && len(tenant.Theme) > 0 {
			var theme map[string]any
			if json.Unmarshal(tenant.Theme, &theme) == nil && len(theme) > 0 {
				response["theme"] = theme
			}
		}
	}

	return c.JSON(response)
}
