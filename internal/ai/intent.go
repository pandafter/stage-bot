package ai

import (
	"strings"

	"github.com/kart-academy/instagram-bot/internal/domain"
)

// offTopicKeywords must stay unaccented because matching runs on normalizeForMatch output.
var offTopicKeywords = []string{
	"clima", "tiempo hoy", "presidente", "futbol", "partido",
	"musica", "receta", "pelicula", "serie", "noticias",
}

// normalizeForMatch lowercases first and then applies this replacer,
// so only lowercase accented runes are needed here.
var accentNormalizer = strings.NewReplacer(
	"á", "a",
	"é", "e",
	"í", "i",
	"ó", "o",
	"ú", "u",
	"ü", "u",
)

// DetectIntent classifies a user message into a sales-relevant intent.
// Uses keyword matching for speed — runs before Claude to select strategy.
func DetectIntent(text string) domain.Intent {
	lower := strings.ToLower(text)

	// Order matters: more specific intents first

	// Greeting (check early — "buenas tardes" should be greeting, not schedule)
	if containsAny(lower, "hola", "hey", "buenas", "buenos", "hello",
		"qué tal", "que tal", "qué más", "que mas", "epa") ||
		lower == "hi" || lower == "ey" || strings.HasPrefix(lower, "hi ") || strings.HasPrefix(lower, "ey ") {
		return domain.IntentGreeting
	}

	// Payment / buy signals
	if containsAny(lower, "cómo pago", "como pago", "link de pago", "pagar", "transferencia",
		"nequi", "daviplata", "efectivo", "tarjeta", "pse") {
		return domain.IntentPaymentConfirm
	}

	if containsAny(lower, "quiero inscribirme", "me inscribo", "reservo", "reservar",
		"me apunto", "listo para", "dale hagámosle", "dale hagamosle",
		"voy a tomar", "quiero el curso", "lo quiero", "me interesa comprar",
		"quiero comprar", "dónde pago", "donde pago", "cuándo empiezo", "cuando empiezo") {
		return domain.IntentBuySignal
	}

	// Objections
	if containsAny(lower, "caro", "costoso", "muy caro", "no tengo plata", "no me alcanza",
		"mucha plata", "descuento", "rebaja", "promoción", "promocion", "oferta",
		"más barato", "mas barato", "no puedo pagar") {
		return domain.IntentObjectionPrice
	}

	if containsAny(lower, "no tengo tiempo", "no me queda", "estoy ocupado", "muy lejos",
		"me queda lejos", "no puedo ir") {
		return domain.IntentObjectionTime
	}

	if containsAny(lower, "no sé si", "no se si", "no estoy seguro", "será que", "sera que",
		"tengo miedo", "es peligroso", "me da miedo", "es seguro", "accidente") {
		return domain.IntentObjectionDoubt
	}

	// Price inquiry
	if containsAny(lower, "precio", "costo", "cuánto", "cuanto", "vale", "valor",
		"tarifa", "cuánto cuesta", "cuanto cuesta", "qué precio", "que precio") {
		return domain.IntentPriceInquiry
	}

	// Schedule inquiry
	if containsAny(lower, "horario", "hora", "cuándo", "cuando", "disponibilidad",
		"qué días", "que dias", "fin de semana", "entre semana", "sábado", "sabado",
		"domingo", "mañana", "tarde") {
		return domain.IntentScheduleInquiry
	}

	// Location
	if containsAny(lower, "ubicación", "ubicacion", "dónde", "donde queda", "dirección",
		"direccion", "cómo llego", "como llego", "mapa", "tocancipá", "tocancipa") {
		return domain.IntentLocationInquiry
	}

	// Requirements (age, gear, etc.)
	if containsAny(lower, "edad", "años", "niño", "niña", "hijo", "hija", "requisito",
		"necesito llevar", "qué necesito", "que necesito", "equipo", "casco", "licencia") {
		return domain.IntentRequirements
	}

	// Course inquiry
	if containsAny(lower, "curso", "cursos", "clase", "clases", "programa", "niveles",
		"nivel", "aprender", "enseñan", "entrenar", "entrenamiento", "kart", "karting",
		"kartismo", "pista", "competir", "competencia", "carrera") {
		return domain.IntentCourseInquiry
	}

	// Thanks
	if containsAny(lower, "gracias", "thank", "genial", "perfecto", "excelente",
		"chévere", "chevere", "bacano", "brutal") {
		return domain.IntentThanks
	}

	// Off-topic (outside sales flow/context)
	if containsAny(normalizeForMatch(lower), offTopicKeywords...) {
		return domain.IntentOffTopic
	}

	return domain.IntentUnknown
}

// IntentSalesWeight returns how much a given intent contributes to lead scoring.
func IntentSalesWeight(intent domain.Intent) int {
	switch intent {
	case domain.IntentGreeting:
		return 5
	case domain.IntentCourseInquiry:
		return 10
	case domain.IntentPriceInquiry:
		return 20
	case domain.IntentScheduleInquiry:
		return 15
	case domain.IntentRequirements:
		return 10
	case domain.IntentLocationInquiry:
		return 12
	case domain.IntentBuySignal:
		return 30
	case domain.IntentPaymentConfirm:
		return 35
	case domain.IntentObjectionPrice, domain.IntentObjectionTime, domain.IntentObjectionDoubt, domain.IntentObjectionOther:
		return 10
	case domain.IntentThanks:
		return 5
	case domain.IntentOffTopic:
		return 0
	case domain.IntentUnknown:
		return 3
	default:
		return 0
	}
}

func containsAny(text string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func normalizeForMatch(s string) string {
	return accentNormalizer.Replace(strings.ToLower(s))
}
