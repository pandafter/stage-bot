package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/domain"
	"github.com/kart-academy/instagram-bot/internal/storage"
)

const (
	// How often the follow-up loop checks for stale leads.
	followupCheckInterval = 5 * time.Minute

	// Fixed follow-up schedule from the user's last inbound message.
	followupAfter1 = 20 * time.Minute
	followupAfter2 = 3 * time.Hour
	followupAfter3 = 12 * time.Hour
	followupAfter4 = 24 * time.Hour

	// Max follow-up attempts per lead within one 24h window.
	followupMaxAttempts = 4

	// Max leads to process per cycle (don't blast everyone at once).
	followupBatchSize = 5
)

// FollowUp runs a background loop that re-engages stale leads with a natural,
// non-spammy message. It uses Claude to craft each message based on the
// conversation history so every follow-up feels personal.
type FollowUp struct {
	brain     *Brain
	messenger domain.Messenger
	leads     *storage.LeadsRepo
	conv      *storage.ConversationRepo
	logger    *zap.Logger
}

func NewFollowUp(
	brain *Brain,
	messenger domain.Messenger,
	leads *storage.LeadsRepo,
	conv *storage.ConversationRepo,
	logger *zap.Logger,
) *FollowUp {
	return &FollowUp{
		brain:     brain,
		messenger: messenger,
		leads:     leads,
		conv:      conv,
		logger:    logger,
	}
}

// Start kicks off the background follow-up loop.
func (f *FollowUp) Start(ctx context.Context) {
	go func() {
		// Wait a bit on startup before first check
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Minute):
		}

		f.runCycle(ctx)

		ticker := time.NewTicker(followupCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				f.runCycle(ctx)
			}
		}
	}()
}

func (f *FollowUp) runCycle(ctx context.Context) {
	dbCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Auto-mark leads as abandoned if they got all follow-ups and still didn't respond after 7 days
	if n, err := f.leads.AutoAbandon(dbCtx, 7*24*time.Hour); err != nil {
		f.logger.Warn("followup: auto-abandon failed", zap.Error(err))
	} else if n > 0 {
		f.logger.Info("followup: auto-abandoned stale leads", zap.Int("count", n))
	}

	stale, err := f.leads.StaleLeads(dbCtx, followupAfter1, followupAfter2, followupAfter3, followupAfter4, followupBatchSize)
	if err != nil {
		f.logger.Warn("followup: failed to fetch stale leads", zap.Error(err))
		return
	}

	if len(stale) == 0 {
		return
	}

	f.logger.Info("followup: found stale leads", zap.Int("count", len(stale)))

	for _, lead := range stale {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := f.sendFollowUp(ctx, lead); err != nil {
			f.logger.Error("followup: failed to send",
				zap.String("lead", lead.ID),
				zap.Error(err),
			)
			continue
		}

		// Space out messages so they don't all land at once
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
		}
	}
}

func (f *FollowUp) sendFollowUp(ctx context.Context, lead *storage.LeadRecord) error {
	// Load last few messages for context
	dbCtx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	msgs, err := f.conv.LastN(dbCtx, lead.ID, 6)
	if err != nil {
		return fmt.Errorf("load history: %w", err)
	}

	// Build the follow-up prompt
	prompt := f.buildFollowUpPrompt(lead, msgs)

	history := []ClaudeMessage{
		{Role: "user", Content: "[SISTEMA: genera el mensaje de follow-up]"},
	}

	chat, err := f.brain.claude.Chat(prompt, history)
	if err != nil {
		return fmt.Errorf("claude: %w", err)
	}

	// Clean up — Claude sometimes wraps in quotes
	text := strings.TrimSpace(chat.Text)
	text = strings.Trim(text, "\"")

	// Send it
	if err := f.messenger.SetTypingOn(lead.ID); err != nil {
		f.logger.Debug("followup: typing indicator failed", zap.Error(err))
	}

	time.Sleep(2 * time.Second)

	if err := f.messenger.SendText(lead.ID, text); err != nil {
		// If Instagram rejected it because the 24h window closed,
		// mark max follow-ups so we stop trying this lead.
		if strings.Contains(err.Error(), "outside of allowed window") ||
			strings.Contains(err.Error(), "allowed window") {
			f.logger.Warn("followup: 24h window closed, stopping attempts for this lead",
				zap.String("lead", lead.ID))
			// Set followup_count to max so StaleLeads won't pick it up again
			for i := lead.FollowupCount; i < followupMaxAttempts; i++ {
				_ = f.leads.MarkFollowup(dbCtx, lead.ID)
			}
			return nil // not an error, just a window miss
		}
		return fmt.Errorf("send: %w", err)
	}

	// Persist the follow-up as a conversation message
	fuMsg := storage.ConversationMessage{
		LeadID:      lead.ID,
		Role:        storage.RoleAssistant,
		Content:     text,
		ContentType: "text",
		Intent:      "FOLLOWUP",
		Strategy:    followupStrategy(lead),
		TokensIn:    chat.InputTokens,
		TokensOut:   chat.OutputTokens,
		Model:       chat.Model,
		ScoreAfter:  lead.LeadScore,
	}
	if err := f.conv.Append(dbCtx, fuMsg); err != nil {
		f.logger.Error("followup: persist failed", zap.Error(err))
	}

	// Mark it
	if err := f.leads.MarkFollowup(dbCtx, lead.ID); err != nil {
		f.logger.Error("followup: mark failed", zap.Error(err))
	}

	f.logger.Info("followup: sent",
		zap.String("lead", lead.ID),
		zap.Int("attempt", lead.FollowupCount+1),
		zap.Int("score", lead.LeadScore),
	)

	return nil
}

func (f *FollowUp) buildFollowUpPrompt(lead *storage.LeadRecord, msgs []storage.ConversationMessage) string {
	var b strings.Builder

	b.WriteString(`Eres alguien del equipo de una academia de karting. Escribes UN solo mensaje de seguimiento pasivo para retomar una conversación. Escribes como persona, no como vendedor agresivo ni como bot.

CONTEXTO:
`)
	fmt.Fprintf(&b, "- Llevan %d mensajes en la conversación\n", lead.TotalMessages)
	fmt.Fprintf(&b, "- Score de interés: %d\n", lead.LeadScore)
	fmt.Fprintf(&b, "- Es el follow-up #%d\n", lead.FollowupCount+1)

	if lead.PriceAsked {
		b.WriteString("- Ya había preguntado por precios\n")
	}
	if lead.ScheduleAsked {
		b.WriteString("- Ya había preguntado por fechas/horarios\n")
	}
	if lead.BuySignal {
		b.WriteString("- Había mostrado intención de compra\n")
	}

	// Include conversation tail
	if len(msgs) > 0 {
		b.WriteString("\nÚLTIMOS MENSAJES DE LA CONVERSACIÓN:\n")
		for _, m := range msgs {
			who := "Cliente"
			if m.Role == storage.RoleAssistant {
				who = "Tú"
			}
			content := oneLine(m.Content)
			fmt.Fprintf(&b, "%s: %s\n", who, content)
		}
	}

	// Strategy based on attempt number
	switch lead.FollowupCount {
	case 0:
		b.WriteString(`
	TIPO: primer follow-up (20 min). Muy suave.
	- Menciona algo de la conversación que tuvieron. Que se note que te acuerdas de lo que habló
	- Estructura: 1) contexto breve, 2) pregunta con 2 opciones para avanzar
	- Máximo 2 oraciones. Cero presión`)

	case 1:
		b.WriteString(`
	TIPO: segundo follow-up (3 horas). Pasivo y útil.
	- Mantén tono amable y sin urgencia
	- Estructura: 1) dato útil corto, 2) elección concreta (ej: precios o fechas)
	- Máximo 2 oraciones`)

	case 2:
		b.WriteString(`
	TIPO: tercer follow-up (12 horas). Reconducción al flujo.
	- No reclames ni insinúes compromiso previo
	- Estructura: 1) validación breve, 2) opción para continuar por flujo de inscripción
	- Si aplica, pregunta si le compartes el link de inscripción
	- Máximo 2 oraciones`)

	default:
		b.WriteString(`
	TIPO: cuarto follow-up (1 día). Último toque, muy respetuoso.
	- Da salida elegante si no es el momento
	- Estructura: 1) cierre amable, 2) opción simple para retomar cuando quiera
	- Si está listo para avanzar, prioriza link de inscripción antes de cualquier link de pago total
	- Máximo 2 oraciones`)
	}

	b.WriteString(`

REGLAS:
- Escribe SOLO el mensaje. Sin comillas, sin explicaciones, sin encabezados
- Suena a persona real escribiendo desde el celular. Imperfecto, rápido, natural
- NO ataques al cliente, NO reclames, NO uses frases tipo "quedamos en..." o "te comprometiste"
- NO uses "qué más", "cuéntame", "Excelente", "no te lo pierdas", "oferta", "aprovecha"
- NO copies frases de estas instrucciones. Inventa algo propio basado en la conversación
- Mantén formato guiado por opciones para llevar al flujo
- Si ya está listo para pagar, primero prioriza enviar el link de inscripción
- El link de pago total va después y solo si corresponde (ej: tarjeta), según la info del contexto
- Máximo 1 emoji. Casi siempre mejor sin emoji`)

	return b.String()
}

func followupStrategy(lead *storage.LeadRecord) string {
	if lead.BuySignal || lead.LeadScore >= 61 {
		return "FOLLOWUP_HOT"
	}
	if lead.PriceAsked || lead.ScheduleAsked {
		return "FOLLOWUP_WARM"
	}
	return "FOLLOWUP_MILD"
}
