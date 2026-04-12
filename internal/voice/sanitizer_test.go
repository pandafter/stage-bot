package voice

import "testing"

func TestSanitizeForTTS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
	}{
		{
			name:  "strips emojis",
			input: "¡Qué más! 🏁 Todo bien 😊",
			want:  "¡Qué más! Todo bien",
		},
		{
			name:  "spells prices in pesos",
			input: "El valor es $890.000 y manejamos varios métodos.",
			want:  "El valor es ochocientos noventa mil pesos y manejamos varios métodos.",
		},
		{
			name:  "spells million prices",
			input: "El plan sube a $1.200.000 este mes.",
			want:  "El plan sube a un millón doscientos mil pesos este mes.",
		},
		{
			name:  "converts em dash to comma",
			input: "El curso dura 2 días — el primero es teoría.",
			want:  "El curso dura 2 días , el primero es teoría.",
		},
		{
			name:  "unwraps parenthesis",
			input: "Incluye todo (kart, indumentaria y coaching).",
			want:  "Incluye todo , kart, indumentaria y coaching, .",
		},
		{
			name:  "replaces url",
			input: "Paga aquí: https://pagos.ejemplo.com/curso",
			want:  "Paga aquí: el link que te paso abajo",
		},
		{
			name:  "reads percent",
			input: "Tienes un 10% de descuento.",
			want:  "Tienes un 10 por ciento de descuento.",
		},
		{
			name:  "reads time on the hour",
			input: "Nos vemos a las 9:00.",
			want:  "Nos vemos a las nueve en punto.",
		},
		{
			name:  "reads half hour",
			input: "Arrancamos a las 3:30.",
			want:  "Arrancamos a las tres y media.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeForTTS(tt.input)
			if got != tt.want {
				t.Errorf("\ninput: %q\nwant:  %q\ngot:   %q", tt.input, tt.want, got)
			}
		})
	}
}

func TestSpellInt(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "cero"},
		{1, "uno"},
		{15, "quince"},
		{100, "cien"},
		{101, "ciento uno"},
		{450, "cuatrocientos cincuenta"},
		{890000, "ochocientos noventa mil"},
		{1200000, "un millón doscientos mil"},
		{1500000, "un millón quinientos mil"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := spellInt(tt.n)
			if got != tt.want {
				t.Errorf("spellInt(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}
