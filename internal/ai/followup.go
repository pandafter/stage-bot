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
	followupAfter2 = 24 * time.Hour
	followupAfter3 = 48 * time.Hour

	// Max follow-up attempts per lead before stopping.
	followupMaxAttempts = 3

	// Max leads to process per cycle (don't blast everyone at once).
	followupBatchSize = 5

	// Local hour when "night" starts for promised responses.
	followupNightStartHour = 18

	// Message window for follow-up logic context.
	followupLoadWindow   = 20
	followupContextLimit = 6
)

var followupLocalTZ = func() *time.Location {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		return time.FixedZone("COT", -5*60*60)
	}
	return loc
}()

// FollowUp runs a background loop that re-engages stale leads with a simple,
// low-pressure message.
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

	stale, err := f.leads.StaleLeads(dbCtx, followupAfter1, followupAfter2, followupAfter3, followupBatchSize)
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

	msgs, err := f.conv.LastN(dbCtx, lead.ID, followupLoadWindow)
	if err != nil {
		return fmt.Errorf("load history: %w", err)
	}
	msgs = dropPastFollowupResponses(msgs)
	if len(msgs) > followupContextLimit {
		msgs = msgs[len(msgs)-followupContextLimit:]
	}

	if shouldDeferForNightPromise(msgs, time.Now()) {
		f.logger.Info("followup: deferred due to night promise",
			zap.String("lead", lead.ID),
		)
		return nil
	}

	text := simpleFollowupMessage(lead)

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

func simpleFollowupMessage(lead *storage.LeadRecord) string {
	if lead.BuySignal || lead.LeadScore >= 61 {
		return "Hola, ¿cómo estás? ¿Quieres que sigamos con la inscripción?"
	}
	if lead.PriceAsked && !lead.ScheduleAsked {
		return "Hola, ¿cómo estás? ¿Quieres que sigamos con horarios o con inscripción?"
	}
	if lead.ScheduleAsked && !lead.PriceAsked {
		return "Hola, ¿cómo estás? ¿Quieres que sigamos con precios o con inscripción?"
	}
	return "Hola, ¿cómo estás? ¿Quieres que sigamos con precios o horarios?"
}

func shouldDeferForNightPromise(msgs []storage.ConversationMessage, now time.Time) bool {
	lastUser, ok := lastUserMessage(msgs)
	if !ok {
		return false
	}

	content := normalizeFollowupText(lastUser.Content)
	if !containsAnyFollowup(content,
		"en la noche",
		"esta noche",
		"por la noche",
		"mas tarde en la noche",
		"hoy en la noche",
	) {
		return false
	}

	if !containsAnyFollowup(content,
		"te aviso",
		"aviso",
		"te escribo",
		"escribo",
		"te confirmo",
		"confirmo",
		"te respondo",
		"respondo",
		"te cuento",
		"cuento",
	) {
		return false
	}

	nowLocal := now.In(followupLocalTZ)
	msgLocal := lastUser.CreatedAt.In(followupLocalTZ)

	// Defer only until the night block of the same day the promise was made.
	if sameLocalDay(nowLocal, msgLocal) && nowLocal.Hour() < followupNightStartHour {
		return true
	}
	return false
}

func lastUserMessage(msgs []storage.ConversationMessage) (storage.ConversationMessage, bool) {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == storage.RoleUser {
			return msgs[i], true
		}
	}
	return storage.ConversationMessage{}, false
}

func normalizeFollowupText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(
		"á", "a",
		"é", "e",
		"í", "i",
		"ó", "o",
		"ú", "u",
		"ü", "u",
	)
	s = replacer.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func containsAnyFollowup(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func sameLocalDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
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

func dropPastFollowupResponses(msgs []storage.ConversationMessage) []storage.ConversationMessage {
	out := make([]storage.ConversationMessage, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == storage.RoleAssistant && strings.EqualFold(strings.TrimSpace(m.Intent), "FOLLOWUP") {
			continue
		}
		out = append(out, m)
	}
	return out
}
