package ai

import (
	"testing"

	"github.com/kart-academy/instagram-bot/internal/domain"
)

func TestDetectIntent(t *testing.T) {
	tests := []struct {
		input string
		want  domain.Intent
	}{
		// Greetings
		{"Hola", domain.IntentGreeting},
		{"hey qué más", domain.IntentGreeting},
		{"buenas tardes", domain.IntentGreeting},
		{"hello", domain.IntentGreeting},
		{"epa", domain.IntentGreeting},

		// Price inquiry
		{"cuánto cuesta el curso?", domain.IntentPriceInquiry},
		{"qué precio tiene?", domain.IntentPriceInquiry},
		{"cuánto vale?", domain.IntentPriceInquiry},
		{"cuál es la tarifa?", domain.IntentPriceInquiry},
		{"me puedes pasar el costo?", domain.IntentPriceInquiry},

		// Schedule inquiry
		{"qué horarios manejan?", domain.IntentScheduleInquiry},
		{"tienen clases los sábados?", domain.IntentScheduleInquiry},
		{"hay disponibilidad en la tarde?", domain.IntentScheduleInquiry},
		{"qué días tienen disponibles?", domain.IntentScheduleInquiry},
		{"tienen fin de semana?", domain.IntentScheduleInquiry},

		// Location
		{"dónde queda la pista?", domain.IntentLocationInquiry},
		{"cómo llego?", domain.IntentLocationInquiry},
		{"están en Tocancipá?", domain.IntentLocationInquiry},
		{"me pasan la ubicación?", domain.IntentLocationInquiry},

		// Requirements
		{"qué edad mínima tienen?", domain.IntentRequirements},
		{"mi hijo tiene 9 años", domain.IntentRequirements},
		{"qué necesito llevar?", domain.IntentRequirements},
		{"necesito casco?", domain.IntentRequirements},

		// Course inquiry
		{"qué cursos tienen?", domain.IntentCourseInquiry},
		{"me cuentan sobre las clases", domain.IntentCourseInquiry},
		{"quiero aprender karting", domain.IntentCourseInquiry},
		{"tienen niveles?", domain.IntentCourseInquiry},
		{"quiero entrenar para competir", domain.IntentCourseInquiry},

		// Buy signal
		{"quiero inscribirme", domain.IntentBuySignal},
		{"me apunto", domain.IntentBuySignal},
		{"lo quiero", domain.IntentBuySignal},
		{"dale hagámosle", domain.IntentBuySignal},
		{"cuándo empiezo?", domain.IntentBuySignal},

		// Payment confirm
		{"cómo pago?", domain.IntentPaymentConfirm},
		{"puedo pagar con Nequi?", domain.IntentPaymentConfirm},
		{"aceptan transferencia?", domain.IntentPaymentConfirm},
		{"tienen Daviplata?", domain.IntentPaymentConfirm},
		{"pago con tarjeta?", domain.IntentPaymentConfirm},

		// Objection - price
		{"está muy caro", domain.IntentObjectionPrice},
		{"no tengo plata", domain.IntentObjectionPrice},
		{"hay algún descuento?", domain.IntentObjectionPrice},
		{"tienen promoción?", domain.IntentObjectionPrice},

		// Objection - time
		{"no tengo tiempo", domain.IntentObjectionTime},
		{"me queda lejos", domain.IntentObjectionTime},
		{"estoy ocupado siempre", domain.IntentObjectionTime},

		// Objection - doubt
		{"no sé si sea para mí", domain.IntentObjectionDoubt},
		{"es peligroso?", domain.IntentObjectionDoubt},
		{"me da miedo", domain.IntentObjectionDoubt},
		{"es seguro para niños?", domain.IntentObjectionDoubt},

		// Thanks
		{"gracias!", domain.IntentThanks},
		{"perfecto", domain.IntentThanks},
		{"genial", domain.IntentThanks},
		{"chevere", domain.IntentThanks},

		// Unknown / off-topic
		{"jajajajaja", domain.IntentUnknown},
		{"qué clima hace hoy?", domain.IntentUnknown},
		{"🔥🔥🔥", domain.IntentUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := DetectIntent(tt.input)
			if got != tt.want {
				t.Errorf("DetectIntent(%q) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestIntentSalesWeight(t *testing.T) {
	// Buy signals and payment should have highest weight
	if w := IntentSalesWeight(domain.IntentPaymentConfirm); w < 30 {
		t.Errorf("PaymentConfirm weight = %d, want >= 30", w)
	}
	if w := IntentSalesWeight(domain.IntentBuySignal); w < 25 {
		t.Errorf("BuySignal weight = %d, want >= 25", w)
	}

	// Off-topic should have zero weight
	if w := IntentSalesWeight(domain.IntentOffTopic); w != 0 {
		t.Errorf("OffTopic weight = %d, want 0", w)
	}

	// Objections should still have positive weight (engagement)
	if w := IntentSalesWeight(domain.IntentObjectionPrice); w <= 0 {
		t.Errorf("ObjectionPrice weight = %d, want > 0", w)
	}
}
