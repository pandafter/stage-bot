package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/storage"
)

const (
	playbookRefreshInterval = 1 * time.Hour
	playbookSampleSize      = 5
	playbookMaxExcerpt      = 8 // messages per conversation excerpt
)

// Playbook builds a learning snapshot from won and lost deals and exposes it
// as a prompt fragment that Brain injects into Claude's system prompt.
type Playbook struct {
	leads *storage.LeadsRepo
	conv  *storage.ConversationRepo
	log   *zap.Logger

	mu       sync.RWMutex
	snapshot string
}

func NewPlaybook(leads *storage.LeadsRepo, conv *storage.ConversationRepo, log *zap.Logger) *Playbook {
	return &Playbook{leads: leads, conv: conv, log: log}
}

// Snapshot returns the cached prompt fragment. Empty if never refreshed.
func (p *Playbook) Snapshot() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.snapshot
}

// Start kicks off a background refresh loop. Safe to call once at startup.
func (p *Playbook) Start(ctx context.Context) {
	go func() {
		if err := p.Refresh(ctx); err != nil {
			p.log.Warn("playbook initial refresh failed", zap.Error(err))
		}
		ticker := time.NewTicker(playbookRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := p.Refresh(ctx); err != nil {
					p.log.Warn("playbook refresh failed", zap.Error(err))
				}
			}
		}
	}()
}

// Refresh rebuilds the snapshot from recent won and lost deals.
func (p *Playbook) Refresh(ctx context.Context) error {
	dbCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	wins, err := p.leads.ListByOutcome(dbCtx, storage.OutcomeWon, playbookSampleSize)
	if err != nil {
		return fmt.Errorf("list wins: %w", err)
	}
	losses, err := p.leads.ListByOutcome(dbCtx, storage.OutcomeLost, playbookSampleSize)
	if err != nil {
		return fmt.Errorf("list losses: %w", err)
	}

	if len(wins) == 0 && len(losses) == 0 {
		p.mu.Lock()
		p.snapshot = ""
		p.mu.Unlock()
		return nil
	}

	var b strings.Builder
	b.WriteString("APRENDIZAJE DE CONVERSACIONES ANTERIORES:\n")

	if len(wins) > 0 {
		b.WriteString("\nLO QUE HA FUNCIONADO (ventas cerradas) — imita el estilo y los patrones:\n")
		for _, lead := range wins {
			excerpt, err := p.buildExcerpt(dbCtx, lead.ID)
			if err != nil {
				continue
			}
			if excerpt != "" {
				b.WriteString(excerpt)
			}
		}
	}

	if len(losses) > 0 {
		b.WriteString("\nLO QUE NO HA FUNCIONADO (ventas perdidas) — evita estos patrones:\n")
		for _, lead := range losses {
			excerpt, err := p.buildExcerpt(dbCtx, lead.ID)
			if err != nil {
				continue
			}
			if excerpt != "" {
				b.WriteString(excerpt)
			}
		}
	}

	p.mu.Lock()
	p.snapshot = b.String()
	p.mu.Unlock()

	p.log.Info("playbook refreshed",
		zap.Int("wins", len(wins)),
		zap.Int("losses", len(losses)),
		zap.Int("bytes", len(p.snapshot)),
	)
	return nil
}

func (p *Playbook) buildExcerpt(ctx context.Context, leadID string) (string, error) {
	msgs, err := p.conv.FullForLead(ctx, leadID)
	if err != nil {
		return "", err
	}
	if len(msgs) == 0 {
		return "", nil
	}

	// Keep the tail of the conversation — that's where the close (or drop) happens.
	if len(msgs) > playbookMaxExcerpt {
		msgs = msgs[len(msgs)-playbookMaxExcerpt:]
	}

	var b strings.Builder
	b.WriteString("---\n")
	for _, m := range msgs {
		who := "Cliente"
		if m.Role == storage.RoleAssistant {
			who = "Asesor"
		}
		b.WriteString(who)
		b.WriteString(": ")
		b.WriteString(oneLine(m.Content))
		b.WriteString("\n")
	}
	return b.String(), nil
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 240 {
		s = s[:240] + "…"
	}
	return s
}
