package knowledge

import (
	"strings"
	"testing"
)

func TestParseEmpresaSheet_real(t *testing.T) {
	// Mirrors the live sheet structure as of 2026-04-26.
	rows := [][]string{
		{"nombre", "SCUDERIA ST4GE", "NOTAS IMPORTANTES IA", "follow-ups", "charla_incluye"},
		{"descripcion", "EQUIPO PROFESIONAL", "Aplica notas siempre",
			"6 horas: Hola hola! como estas?\n24 horas : hello! como vas?\n48 horas: Quieres cumplir tu sueño?\n120 horas: Sigo super pendiente",
			"Hola!, Bienvenido a St4ge"},
		{"ubicacion", "Tocancipá", "Si te mandan piloto en comentario, responde", "",
			"¿Te gustaría el link de inscripción?"},
		{"telefono", "3059456266", "", "", ""},
		{"instagram", "@scuderia_stage4", "Solo predeterminados", "", ""},
		{"horario_atencion", "Lun-Vie 8am-11pm", "", "", ""},
	}
	got := parseEmpresaSheet(rows)

	if got.Scripted.FollowUp6H == "" || !strings.Contains(got.Scripted.FollowUp6H, "Hola hola") {
		t.Errorf("FollowUp6H wrong: %q", got.Scripted.FollowUp6H)
	}
	if got.Scripted.FollowUp24H == "" || !strings.Contains(got.Scripted.FollowUp24H, "hello!") {
		t.Errorf("FollowUp24H wrong: %q", got.Scripted.FollowUp24H)
	}
	if got.Scripted.FollowUp48H == "" || !strings.Contains(got.Scripted.FollowUp48H, "Quieres cumplir") {
		t.Errorf("FollowUp48H wrong: %q", got.Scripted.FollowUp48H)
	}
	if got.Scripted.FollowUp120H == "" || !strings.Contains(got.Scripted.FollowUp120H, "Sigo super") {
		t.Errorf("FollowUp120H wrong: %q", got.Scripted.FollowUp120H)
	}

	if got.Scripted.Message1 != "Hola!, Bienvenido a St4ge" {
		t.Errorf("Message1 wrong: %q", got.Scripted.Message1)
	}
	if got.Scripted.Message2 != "¿Te gustaría el link de inscripción?" {
		t.Errorf("Message2 wrong: %q", got.Scripted.Message2)
	}

	if got.WhatsApp != "3059456266" {
		t.Errorf("WhatsApp wrong: %q", got.WhatsApp)
	}
	if got.KeyValues["nombre"] != "SCUDERIA ST4GE" {
		t.Errorf("nombre KV wrong: %q", got.KeyValues["nombre"])
	}
	if got.KeyValues["instagram"] != "@scuderia_stage4" {
		t.Errorf("instagram KV wrong: %q", got.KeyValues["instagram"])
	}

	if len(got.Notas) < 2 {
		t.Errorf("expected multiple notas, got %d", len(got.Notas))
	}
}

func TestParseFollowUpsBlock_singleCell(t *testing.T) {
	block := "6 horas: A\n24 horas : B\n48 horas:  C\n120 horas: D"
	got := parseFollowUpsBlock(block)
	for h, want := range map[int]string{6: "A", 24: "B", 48: "C", 120: "D"} {
		if got[h] != want {
			t.Errorf("hour %d: got %q want %q", h, got[h], want)
		}
	}
}

func TestSplitMessageAndQuestion(t *testing.T) {
	body := "Hola, info\n\nPrecio: $100\n\n¿Quieres el link?"
	intro, q := splitMessageAndQuestion(body)
	if !strings.Contains(intro, "Precio: $100") {
		t.Errorf("intro lost content: %q", intro)
	}
	if q != "¿Quieres el link?" {
		t.Errorf("question wrong: %q", q)
	}
}
