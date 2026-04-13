package ai

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/domain"
	"github.com/kart-academy/instagram-bot/internal/storage"
)

const (
	historyWindow = 20
	dbTimeout     = 5 * time.Second
)

// Brain orchestrates AI responses for conversations, backed by Postgres.
type Brain struct {
	claude       *Claude
	knowledge    domain.KnowledgeBase
	leads        *storage.LeadsRepo
	conversation *storage.ConversationRepo
	playbook     *Playbook
	logger       *zap.Logger
}

func New(
	apiKey string,
	ks domain.KnowledgeBase,
	leads *storage.LeadsRepo,
	conv *storage.ConversationRepo,
	pb *Playbook,
	logger *zap.Logger,
) *Brain {
	return &Brain{
		claude:       NewClaude(apiKey, logger),
		knowledge:    ks,
		leads:        leads,
		conversation: conv,
		playbook:     pb,
		logger:       logger,
	}
}

// Process takes a user message and returns the AI response.
// Pipeline: ensure lead -> detect intent -> update score -> select strategy -> generate -> persist.
func (b *Brain) Process(senderID, text string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rec, err := b.leads.Ensure(ctx, senderID)
	if err != nil {
		b.logger.Error("lead ensure failed", zap.String("sender", senderID), zap.Error(err))
		return b.fallback(text), nil
	}

	intent := DetectIntent(text)
	scoreDelta := ApplyIntent(rec, intent)
	strategy := SelectStrategy(intent, rec)

	b.logger.Info("pipeline",
		zap.String("sender", senderID),
		zap.String("intent", string(intent)),
		zap.Int("score", rec.LeadScore),
		zap.Int("score_delta", scoreDelta),
		zap.String("state", string(rec.State)),
		zap.String("strategy", string(strategy)),
	)

	history, err := b.loadHistory(ctx, senderID)
	if err != nil {
		b.logger.Warn("history load failed, continuing without", zap.Error(err))
	}
	history = append(history, ClaudeMessage{Role: "user", Content: text})

	systemPrompt := b.buildSystemPrompt(strategy, rec)
	chat, err := b.claude.Chat(systemPrompt, history)
	if err != nil {
		b.logger.Warn("claude unavailable, using fallback", zap.Error(err))
		chat = ChatResult{Text: b.fallback(text)}
	}

	b.persist(ctx, senderID, text, chat, intent, strategy, scoreDelta, rec)
	return chat.Text, nil
}

// loadHistory pulls the last N messages from DB and formats them for Claude.
func (b *Brain) loadHistory(ctx context.Context, senderID string) ([]ClaudeMessage, error) {
	dbCtx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	msgs, err := b.conversation.LastN(dbCtx, senderID, historyWindow)
	if err != nil {
		return nil, err
	}

	out := make([]ClaudeMessage, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, ClaudeMessage{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}
	return out, nil
}

// persist writes the user and assistant messages plus updated lead state.
// Failures are logged but not returned — we've already replied to the user.
func (b *Brain) persist(
	ctx context.Context,
	senderID, userText string,
	chat ChatResult,
	intent domain.Intent,
	strategy domain.Strategy,
	scoreDelta int,
	rec *storage.LeadRecord,
) {
	dbCtx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	userMsg := storage.ConversationMessage{
		LeadID:      senderID,
		Role:        storage.RoleUser,
		Content:     userText,
		ContentType: "text",
		Intent:      string(intent),
		Strategy:    string(strategy),
		ScoreDelta:  scoreDelta,
		ScoreAfter:  rec.LeadScore,
	}
	if err := b.conversation.Append(dbCtx, userMsg); err != nil {
		b.logger.Error("persist user message", zap.Error(err))
	}

	assistantMsg := storage.ConversationMessage{
		LeadID:      senderID,
		Role:        storage.RoleAssistant,
		Content:     chat.Text,
		ContentType: "text",
		Intent:      string(intent),
		Strategy:    string(strategy),
		ScoreAfter:  rec.LeadScore,
		TokensIn:    chat.InputTokens,
		TokensOut:   chat.OutputTokens,
		Model:       chat.Model,
	}
	if err := b.conversation.Append(dbCtx, assistantMsg); err != nil {
		b.logger.Error("persist assistant message", zap.Error(err))
	}

	if err := b.leads.UpdateScore(dbCtx, rec); err != nil {
		b.logger.Error("persist lead score", zap.Error(err))
	}
}

// buildSystemPrompt creates the full prompt with knowledge, strategy and learned patterns.
func (b *Brain) buildSystemPrompt(strategy domain.Strategy, rec *storage.LeadRecord) string {
	prompt := basePrompt
	prompt += "\n\n" + strategyInstruction(strategy, rec)

	if b.knowledge != nil && b.knowledge.Enabled() {
		if ctx := b.knowledge.FormatContext(); ctx != "" {
			prompt += "\n\nBASA TUS RESPUESTAS EN ESTA INFORMACIÓN REAL DEL NEGOCIO:" + ctx
			prompt += "\n\nUSA la sección de EJEMPLOS DE VENTAS para imitar el tono y estilo real del equipo."
		}
	}

	if b.playbook != nil {
		if learned := b.playbook.Snapshot(); learned != "" {
			prompt += "\n\n" + learned
		}
	}

	return prompt
}

func (b *Brain) fallback(text string) string {
	return fallbackReply(text)
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
