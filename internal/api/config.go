package api

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

// GetConfig exposes the catalog (modalidades, métodos, fechas, recargos).
// If a CMS repo is attached, published sections are included under the "cms" key.
func (h *Handler) GetConfig(c *fiber.Ctx) error {
	response := fiber.Map{
		"modalidades":      Modalidades,
		"metodos":          Metodos,
		"fechas":           Fechas,
		"card_surcharge_pct": CardSurchargePct,
		"reserva_cop":      ReservaCOP,
		"precio_completo":  PrecioCompleto,
		"precio_descuento": PrecioDescuento,
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
