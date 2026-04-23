package ai

import (
	"testing"

	"github.com/kart-academy/instagram-bot/internal/storage"
)

type fakeScriptedKnowledge struct {
	primary  []string
	director string
	followup map[int]string
}

func (f fakeScriptedKnowledge) Enabled() bool {
	return true
}

func (f fakeScriptedKnowledge) FormatContext() string {
	return ""
}

func (f fakeScriptedKnowledge) PrimaryMessages() []string {
	return f.primary
}

func (f fakeScriptedKnowledge) DirectorContactMessage() string {
	return f.director
}

func (f fakeScriptedKnowledge) FollowUpMessage(attempt int) string {
	return f.followup[attempt]
}

func TestScriptedReply_FirstMessageSendsPrimary(t *testing.T) {
	b := &Brain{
		knowledge: fakeScriptedKnowledge{
			primary:  []string{"M1", "M2"},
			director: "DIR",
		},
	}

	got := b.scriptedReply(nil, "hola")
	if got.reply != "M1\n\nM2" {
		t.Fatalf("unexpected primary reply: %q", got.reply)
	}
	if got.assistantIntent != "SCRIPT_PRIMARY" {
		t.Fatalf("unexpected assistant intent: %q", got.assistantIntent)
	}
}

func TestScriptedReply_YesSendsDirector(t *testing.T) {
	b := &Brain{
		knowledge: fakeScriptedKnowledge{
			primary:  []string{"M1", "M2"},
			director: "DIR",
		},
	}
	history := []storage.ConversationMessage{
		{Role: storage.RoleAssistant, Intent: "SCRIPT_PRIMARY"},
	}

	got := b.scriptedReply(history, "sí")
	if got.reply != "DIR" {
		t.Fatalf("expected director reply, got %q", got.reply)
	}
	if got.assistantIntent != "SCRIPT_DIRECTOR" {
		t.Fatalf("unexpected assistant intent: %q", got.assistantIntent)
	}
}

func TestScriptedReply_NoStaysSilent(t *testing.T) {
	b := &Brain{
		knowledge: fakeScriptedKnowledge{
			primary:  []string{"M1", "M2"},
			director: "DIR",
		},
	}
	history := []storage.ConversationMessage{
		{Role: storage.RoleAssistant, Intent: "SCRIPT_PRIMARY"},
	}

	got := b.scriptedReply(history, "no")
	if got.reply != "" {
		t.Fatalf("expected silence on no, got %q", got.reply)
	}
}

func TestScriptedReply_KartSaleDirectsToDirector(t *testing.T) {
	b := &Brain{
		knowledge: fakeScriptedKnowledge{
			primary:  []string{"M1", "M2"},
			director: "DIR",
		},
	}

	got := b.scriptedReply(nil, "¿venden kart? quiero comprar")
	if got.reply != "DIR" {
		t.Fatalf("expected direct handoff on kart sale question, got %q", got.reply)
	}
}

func TestFollowUpMessage_UsesKnowledgeFirst(t *testing.T) {
	b := &Brain{
		knowledge: fakeScriptedKnowledge{
			followup: map[int]string{
				1: "FU6",
				2: "FU24",
				3: "FU120",
			},
		},
	}

	if got := b.FollowUpMessage(1, &storage.LeadRecord{}); got != "FU6" {
		t.Fatalf("attempt 1 got %q", got)
	}
	if got := b.FollowUpMessage(2, &storage.LeadRecord{}); got != "FU24" {
		t.Fatalf("attempt 2 got %q", got)
	}
	if got := b.FollowUpMessage(3, &storage.LeadRecord{}); got != "FU120" {
		t.Fatalf("attempt 3 got %q", got)
	}
}
