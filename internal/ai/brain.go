package ai

import (
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/domain"
)

// Brain orchestrates AI responses for conversations.
type Brain struct {
	claude    *Claude
	knowledge domain.KnowledgeBase
	scorer    *LeadScorer
	logger    *zap.Logger
	history   map[string][]ClaudeMessage
	mu        sync.RWMutex
}

func New(apiKey string, ks domain.KnowledgeBase, logger *zap.Logger) *Brain {
	return &Brain{
		claude:    NewClaude(apiKey, logger),
		knowledge: ks,
		scorer:    NewLeadScorer(),
		logger:    logger,
		history:   make(map[string][]ClaudeMessage),
	}
}

// buildSystemPrompt creates the full prompt with knowledge context and sales strategy.
func (b *Brain) buildSystemPrompt(strategy domain.Strategy, score *domain.LeadScore) string {
	prompt := basePrompt

	// Inject strategy-specific instructions
	prompt += "\n\n" + strategyInstruction(strategy, score)

	if b.knowledge != nil && b.knowledge.Enabled() {
		ctx := b.knowledge.FormatContext()
		if ctx != "" {
			prompt += "\n\nBASA TUS RESPUESTAS EN ESTA INFORMACIÓN REAL DEL NEGOCIO:" + ctx
			prompt += "\n\nUSA la sección de EJEMPLOS DE VENTAS para imitar el tono y estilo real del equipo."
		}
	}

	return prompt
}

// Process takes a user message and returns the AI response.
// Pipeline: detect intent -> update score -> select strategy -> generate response.
// Falls back to keyword-based replies when Claude API is unavailable.
func (b *Brain) Process(senderID, text string) (string, error) {
	// Step 1: Detect intent (no lock needed, pure function)
	intent := DetectIntent(text)

	// Step 2: Update lead score
	score := b.scorer.Update(senderID, intent)

	// Step 3: Select strategy
	strategy := SelectStrategy(intent, score)

	b.logger.Info("pipeline",
		zap.String("sender", senderID),
		zap.String("intent", string(intent)),
		zap.Int("score", score.Total),
		zap.String("state", string(score.State)),
		zap.String("strategy", string(strategy)),
	)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Append user message to history
	b.history[senderID] = append(b.history[senderID], ClaudeMessage{
		Role:    "user",
		Content: text,
	})

	// Keep only last 20 messages to control token usage
	if len(b.history[senderID]) > 20 {
		b.history[senderID] = b.history[senderID][len(b.history[senderID])-20:]
	}

	// Step 4: Build strategy-aware prompt and generate response
	systemPrompt := b.buildSystemPrompt(strategy, score)
	response, err := b.claude.Chat(systemPrompt, b.history[senderID])
	if err != nil {
		b.logger.Warn("claude unavailable, using fallback", zap.Error(err))
		response = fallbackReply(text)
	}

	// Append assistant response to history
	b.history[senderID] = append(b.history[senderID], ClaudeMessage{
		Role:    "assistant",
		Content: response,
	})

	return response, nil
}

// fallbackReply returns a keyword-based response when Claude API is unavailable.
func fallbackReply(text string) string {
	lower := strings.ToLower(text)

	switch {
	case containsAny(lower, "hola", "hey", "buenas", "buenos", "hi", "hello", "qué tal", "que tal"):
		return "¡Ey, qué más! Bienvenido 🏁 ¿Qué te gustaría saber sobre nuestros cursos?"

	case containsAny(lower, "precio", "costo", "cuánto", "cuanto", "vale", "valor", "tarifa"):
		return "Claro, déjame confirmo disponibilidad y te paso los precios. ¿Para quién sería el curso?"

	case containsAny(lower, "curso", "clases", "clase", "programa", "aprender", "enseñan"):
		return "Tenemos cursos para todos los niveles 🏎️ ¿Ya has manejado kart antes o arrancarías desde cero?"

	case containsAny(lower, "horario", "cuando", "cuándo", "disponibilidad", "hora"):
		return "Manejamos horarios entre semana y fines de semana. ¿Qué días te quedan mejor?"

	case containsAny(lower, "edad", "años", "niño", "niños", "hijo", "hija", "pequeño"):
		return "Desde los 8 años, los peques arrancan con karts adaptados y supervisor directo. ¿Cuántos años tiene?"

	case containsAny(lower, "ubicación", "ubicacion", "dónde", "donde", "dirección", "direccion", "llegar", "queda"):
		return "Estamos en Tocancipá. ¿Quieres que te pase la ubicación exacta?"

	case containsAny(lower, "gracias", "thank", "genial", "perfecto", "dale", "listo"):
		return "¡De una! Cualquier otra duda me escribes 🙌"

	case containsAny(lower, "seguro", "seguridad", "peligro", "riesgo"):
		return "Tranqui, la seguridad es lo primero. Equipo completo, karts con seguridad y instructores certificados. ¿Alguna otra duda?"

	case containsAny(lower, "inscri", "registr", "empezar", "inicio", "comenzar"):
		return "¡Dale que es! 🏎️ ¿Me dices tu nombre y para qué nivel te interesa?"

	default:
		return "¡Hola! Cuéntame, ¿qué te gustaría saber sobre nuestros cursos de karting?"
	}
}
