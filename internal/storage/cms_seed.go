package storage

import (
	"context"
)

// SeedCMSSections inserts default content for any section that doesn't exist yet.
// Safe to call on every startup — only inserts missing sections, never overwrites.
func SeedCMSSections(ctx context.Context, repo *CMSRepo, tenantID string) error {
	existing, err := repo.GetSections(ctx, tenantID)
	if err != nil {
		return err
	}
	has := make(map[string]bool, len(existing))
	for _, s := range existing {
		has[s.SectionKey] = true
	}

	seeds := defaultSections()
	for key, data := range seeds {
		if has[key] {
			continue
		}
		if err := repo.UpsertSection(ctx, tenantID, key, data, true, 0); err != nil {
			return err
		}
	}
	return nil
}

func defaultSections() map[string]map[string]any {
	return map[string]map[string]any{
		"hero": {
			"title":    "Conviértete en Piloto de Karts",
			"subtitle": "Curso intensivo de fin de semana en el kartódromo de Tocancipá. Mecánica, técnica de manejo y carrera real con instructores activos en competencia.",
			"cta_text": "Reserva tu cupo →",
			"cta_href": "/inscripcion",
			"bg_url":   "",
		},
		"stats": {
			"items": []map[string]any{
				{"value": "+1000", "label": "Pilotos formados"},
				{"value": "10 hrs", "label": "En pista por estudiante"},
				{"value": "2:1", "label": "Ratio piloto/instructor"},
				{"value": "100%", "label": "Satisfacción del cliente"},
			},
		},
		"instructores": {
			"items": []map[string]any{
				{
					"name":      "Andrés Melo",
					"role":      "Director técnico — Piloto activo",
					"bio_html":  "<p>Más de 15 años en pista. Coach principal de la academia.</p>",
					"photo_url": "/assets/AndresMelo.png",
				},
				{
					"name":      "Santi Gutiérrez",
					"role":      "Instructor en pista",
					"bio_html":  "<p>Especialista en trazadas y frenado tardío.</p>",
					"photo_url": "/assets/SantiGutierrez.png",
				},
				{
					"name":      "Equipo mecánico",
					"role":      "Soporte total durante el curso",
					"bio_html":  "<p>Karts revisados antes y entre cada tanda.</p>",
					"photo_url": "/assets/EquipoMecanico.png",
				},
			},
		},
		"faq": {
			"items": []map[string]any{
				{
					"q":     "¿Necesito experiencia previa?",
					"a_html": "<p>No. El curso está diseñado para empezar desde cero, pero también reta a quienes ya han manejado.</p>",
				},
				{
					"q":     "¿Qué edades aplican?",
					"a_html": "<p>Recibimos pilotos desde los 8 años hasta adultos. Cada grupo se ajusta por edad y experiencia.</p>",
				},
				{
					"q":     "¿Qué incluye la inscripción?",
					"a_html": "<p>Equipo de seguridad completo, todas las tandas en pista, instrucción uno a uno y diploma final.</p>",
				},
				{
					"q":     "¿Cómo pago?",
					"a_html": "<p>Tarjeta de crédito/débito, Nequi, Bancolombia, PSE o transferencia con comprobante. Procesamos pagos vía Bold.</p>",
				},
				{
					"q":     "¿Puedo solo reservar el cupo?",
					"a_html": "<p>Sí. La reserva es de $150.000 y abona al pago completo. Te garantiza tu lugar mientras decides.</p>",
				},
			},
		},
		"cta": {
			"title":    "El kartódromo te espera",
			"subtitle": "Asegura tu cupo con la reserva de $150.000. Te confirmamos por WhatsApp y te enviamos toda la info logística del fin de semana.",
			"cta_text": "Inscribirme ahora →",
			"cta_href": "/inscripcion",
			"bg_url":   "",
		},
	}
}
