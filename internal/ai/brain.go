package ai

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/domain"
	"github.com/kart-academy/instagram-bot/internal/knowledge"
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
	salesData    *knowledge.SalesDataset
	leads        *storage.LeadsRepo
	conversation *storage.ConversationRepo
	playbook     *Playbook
	logger       *zap.Logger
}

type scriptedKnowledge interface {
	PrimaryMessages() []string
	DirectorContactMessage() string
	FollowUpMessage(attempt int) string
}

type scriptedDecision struct {
	reply           string
	assistantIntent string
}

func New(
	apiKey string,
	ks domain.KnowledgeBase,
	sd *knowledge.SalesDataset,
	leads *storage.LeadsRepo,
	conv *storage.ConversationRepo,
	pb *Playbook,
	logger *zap.Logger,
) *Brain {
	return &Brain{
		claude:       NewClaude(apiKey, logger),
		knowledge:    ks,
		salesData:    sd,
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

	// If the lead was followed up and now responded, reset the counter
	if rec.FollowupCount > 0 {
		if err := b.leads.ResetFollowup(ctx, senderID); err != nil {
			b.logger.Warn("reset followup failed", zap.Error(err))
		}
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

	msgs, err := b.loadHistory(ctx, senderID)
	if err != nil {
		b.logger.Warn("history load failed, continuing without", zap.Error(err))
	}
	decision := b.scriptedReply(msgs, text)
	if decision.reply == "" {
		b.logger.Info("scripted flow: no reply for non yes/no input",
			zap.String("sender", senderID),
		)
	}

	chat := ChatResult{Text: decision.reply}
	b.persist(ctx, senderID, text, chat, intent, strategy, scoreDelta, rec, decision.assistantIntent)
	return decision.reply, nil
}

// loadHistory pulls the last N messages from DB.
func (b *Brain) loadHistory(ctx context.Context, senderID string) ([]storage.ConversationMessage, error) {
	dbCtx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	history, err := b.conversation.LastN(dbCtx, senderID, historyWindow)
	if err != nil {
		return nil, err
	}
	return history, nil
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
	assistantIntent string,
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

	if strings.TrimSpace(chat.Text) != "" {
		if assistantIntent == "" {
			assistantIntent = string(intent)
		}
		assistantMsg := storage.ConversationMessage{
			LeadID:      senderID,
			Role:        storage.RoleAssistant,
			Content:     chat.Text,
			ContentType: "text",
			Intent:      assistantIntent,
			Strategy:    string(strategy),
			ScoreAfter:  rec.LeadScore,
			TokensIn:    chat.InputTokens,
			TokensOut:   chat.OutputTokens,
			Model:       chat.Model,
		}
		if err := b.conversation.Append(dbCtx, assistantMsg); err != nil {
			b.logger.Error("persist assistant message", zap.Error(err))
		}
	}

	if err := b.leads.UpdateScore(dbCtx, rec); err != nil {
		b.logger.Error("persist lead score", zap.Error(err))
	}
}

func (b *Brain) scriptedReply(history []storage.ConversationMessage, text string) scriptedDecision {
	provider, _ := b.knowledge.(scriptedKnowledge)
	primaryText := b.primaryMessage(provider)
	directorText := b.directorMessage(provider)

	if primaryText == "" {
		primaryText = "Hola, te comparto la información principal de Scuderia St4ge.\n\n¿Quieres el link de inscripción con las fechas y los medios de pago para reservar tu cupo con el descuento que tienes aprobado por tiempo limitado?"
	}

	hasPrimary := false
	hasDirector := false
	for _, msg := range history {
		if msg.Role != storage.RoleAssistant {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(msg.Intent)) {
		case "SCRIPT_PRIMARY":
			hasPrimary = true
		case "SCRIPT_DIRECTOR":
			hasDirector = true
		}
	}

	if asksKartSale(text) && directorText != "" {
		return scriptedDecision{reply: directorText, assistantIntent: "SCRIPT_DIRECTOR"}
	}

	if !hasPrimary {
		return scriptedDecision{reply: primaryText, assistantIntent: "SCRIPT_PRIMARY"}
	}

	if hasDirector {
		return scriptedDecision{}
	}

	switch classifyYesNo(text) {
	case "yes":
		if directorText == "" {
			return scriptedDecision{}
		}
		return scriptedDecision{reply: directorText, assistantIntent: "SCRIPT_DIRECTOR"}
	case "no":
		return scriptedDecision{}
	default:
		return scriptedDecision{}
	}
}

func (b *Brain) primaryMessage(provider scriptedKnowledge) string {
	if provider == nil {
		return ""
	}
	parts := provider.PrimaryMessages()
	if len(parts) == 0 {
		return ""
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "\n\n")
}

func (b *Brain) directorMessage(provider scriptedKnowledge) string {
	if provider == nil {
		return ""
	}
	return strings.TrimSpace(provider.DirectorContactMessage())
}

func (b *Brain) FollowUpMessage(attempt int, lead *storage.LeadRecord) string {
	if provider, ok := b.knowledge.(scriptedKnowledge); ok {
		if msg := strings.TrimSpace(provider.FollowUpMessage(attempt)); msg != "" {
			return msg
		}
	}
	return simpleFollowupMessage(attempt, lead)
}

func classifyYesNo(text string) string {
	normalized := normalizeForMatch(text)
	normalized = strings.TrimSpace(normalized)
	normalized = strings.Trim(normalized, ".,!?¡¿;:")
	if normalized == "" {
		return ""
	}

	yesPhrases := []string{"si quiero", "me interesa", "de una"}
	for _, phrase := range yesPhrases {
		if normalized == phrase || strings.HasPrefix(normalized, phrase+" ") {
			return "yes"
		}
	}

	noPhrases := []string{"no quiero", "no gracias", "ahora no"}
	for _, phrase := range noPhrases {
		if normalized == phrase || strings.HasPrefix(normalized, phrase+" ") {
			return "no"
		}
	}

	yesWords := map[string]struct{}{
		"si": {}, "ok": {}, "dale": {}, "claro": {}, "yes": {},
	}
	noWords := map[string]struct{}{
		"no": {}, "nop": {}, "negativo": {},
	}
	if _, ok := yesWords[normalized]; ok {
		return "yes"
	}
	if _, ok := noWords[normalized]; ok {
		return "no"
	}

	return ""
}

func asksKartSale(text string) bool {
	n := normalizeForMatch(text)
	if !strings.Contains(n, "kart") {
		return false
	}
	return containsAny(n, "venta", "comprar", "compro", "venden", "precio", "cuanto", "cuesta")
}

// buildSystemPrompt creates the full prompt with knowledge, strategy, lead profile,
// sales methodology, and learned patterns from previous conversations.
func (b *Brain) buildSystemPrompt(strategy domain.Strategy, intent domain.Intent, rec *storage.LeadRecord) string {
	prompt := basePrompt
	prompt += "\n\n" + strategyInstruction(strategy, rec)

	// Lead profile: what the bot already knows about this person
	prompt += "\n\n" + leadProfile(rec)

	if b.knowledge != nil && b.knowledge.Enabled() {
		if ctx := b.knowledge.FormatContext(); ctx != "" {
			prompt += "\n\nBASA TUS RESPUESTAS EN ESTA INFORMACIÓN REAL DEL NEGOCIO:" + ctx
			prompt += "\n\nUSA la sección de EJEMPLOS DE VENTAS para imitar el tono y estilo real del equipo."
		}
	}

	// Sales methodology: relevant articles based on current strategy and intent
	if b.salesData != nil {
		if methodology := b.salesData.ForStrategy(string(strategy), string(intent)); methodology != "" {
			prompt += "\n\n" + methodology
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
// Uses random selection so it doesn't sound like the same robot every time.
func fallbackReply(text string) string {
	lower := strings.ToLower(text)

	switch {
	case containsAny(lower, "hola", "hey", "buenas", "buenos", "hi", "hello", "qué tal", "que tal"):
		return pick(
			"hola! cómo estás? en qué andas interesado?",
			"hola, cómo vas? te puedo ayudar con algo?",
			"hola! qué te trae por acá?",
		)

	case containsAny(lower, "precio", "costo", "cuánto", "cuanto", "vale", "valor", "tarifa"):
		return pick(
			"claro, deja confirmo disponibilidad y te paso precios. sería para ti o para alguien más?",
			"ya te averiguo eso. es para adulto o para un niño?",
			"sí, te cuento los precios. para qué nivel sería?",
		)

	case containsAny(lower, "curso", "clases", "clase", "programa", "aprender", "enseñan"):
		return pick(
			"tenemos para todos los niveles. ya has manejado kart antes?",
			"sí, hay varios cursos dependiendo del nivel. has tenido experiencia con karts o sería la primera vez?",
			"claro! tienes algo de experiencia o arrancarías desde cero?",
		)

	case containsAny(lower, "horario", "cuando", "cuándo", "disponibilidad", "hora"):
		return pick(
			"hay entre semana y fines de semana. qué días te quedan mejor?",
			"manejamos varios horarios. te queda mejor entre semana o fin de semana?",
		)

	case containsAny(lower, "edad", "años", "niño", "niños", "hijo", "hija", "pequeño"):
		return pick(
			"desde los 8 años pueden arrancar, con karts adaptados y un supervisor al lado. cuántos años tiene?",
			"sí, tenemos para niños desde los 8 años con equipo adaptado. cuántos años tiene el tuyo?",
		)

	case containsAny(lower, "ubicación", "ubicacion", "dónde", "donde", "dirección", "direccion", "llegar", "queda"):
		return pick(
			"estamos en Tocancipá. te paso la ubicación?",
			"queda en Tocancipá, te mando la ubi?",
		)

	case containsAny(lower, "gracias", "thank", "genial", "perfecto", "dale", "listo"):
		return pick(
			"listo! cualquier cosa me escribes",
			"dale, acá estamos para lo que necesites",
			"con gusto, me dices si te surge algo más",
		)

	case containsAny(lower, "seguro", "seguridad", "peligro", "riesgo"):
		return "la seguridad es lo primero. tienen equipo completo, karts con todas las medidas y los instructores están certificados"

	case containsAny(lower, "inscri", "registr", "empezar", "inicio", "comenzar"):
		return pick(
			"listo! me pasas tu nombre y me dices para qué nivel?",
			"dale! cómo te llamas y qué nivel te interesa?",
		)

	case containsAny(normalizeForMatch(lower), offTopicKeywords...):
		return pick(
			"de ese tema no te puedo ayudar bien por acá. si quieres te ayudo con los cursos: prefieres que te cuente precios o fechas?",
			"me enfoco en info de los cursos de kart. quieres que empecemos por horarios o por costos?",
		)

	default:
		return pick(
			"hola! te ayudo con los cursos. prefieres que te cuente precios o fechas?",
			"si quieres avanzamos rápido: te paso primero horarios o costos?",
		)
	}
}

func pick(options ...string) string {
	return options[rand.Intn(len(options))]
}
