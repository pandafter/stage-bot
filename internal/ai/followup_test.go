package ai

import (
	"testing"
	"time"

	"github.com/kart-academy/instagram-bot/internal/storage"
)

func TestShouldDeferForNightPromise(t *testing.T) {
	loc := followupLocalTZ
	msgTime := time.Date(2026, 4, 21, 11, 0, 0, 0, loc)
	now := time.Date(2026, 4, 21, 15, 0, 0, 0, loc)

	msgs := []storage.ConversationMessage{
		{
			Role:      storage.RoleUser,
			Content:   "Listo, te aviso en la noche",
			CreatedAt: msgTime,
		},
	}

	if !shouldDeferForNightPromise(msgs, now) {
		t.Fatal("expected followup to be deferred before night for night promise")
	}
}

func TestShouldNotDeferForNightPromiseAtNight(t *testing.T) {
	loc := followupLocalTZ
	msgTime := time.Date(2026, 4, 21, 11, 0, 0, 0, loc)
	now := time.Date(2026, 4, 21, 19, 0, 0, 0, loc)

	msgs := []storage.ConversationMessage{
		{
			Role:      storage.RoleUser,
			Content:   "En la noche te confirmo",
			CreatedAt: msgTime,
		},
	}

	if shouldDeferForNightPromise(msgs, now) {
		t.Fatal("expected followup not to be deferred once night starts")
	}
}

func TestShouldNotDeferWithoutNightPromise(t *testing.T) {
	loc := followupLocalTZ
	msgTime := time.Date(2026, 4, 21, 11, 0, 0, 0, loc)
	now := time.Date(2026, 4, 21, 15, 0, 0, 0, loc)

	msgs := []storage.ConversationMessage{
		{
			Role:      storage.RoleUser,
			Content:   "Te aviso mañana",
			CreatedAt: msgTime,
		},
	}

	if shouldDeferForNightPromise(msgs, now) {
		t.Fatal("expected no deferral when there is no explicit night promise")
	}
}

func TestFridayFollowupPlan_DefersBeforeFriday(t *testing.T) {
	loc := followupLocalTZ
	msgTime := time.Date(2026, 4, 22, 11, 0, 0, 0, loc) // Wednesday
	now := time.Date(2026, 4, 23, 15, 0, 0, 0, loc)     // Thursday

	msgs := []storage.ConversationMessage{
		{
			Role:      storage.RoleUser,
			Content:   "Listo, escríbeme el viernes",
			CreatedAt: msgTime,
		},
	}

	plan := fridayFollowupPlan(msgs, now)
	if !plan.deferSend {
		t.Fatal("expected friday followup to defer before friday")
	}
	if plan.forceText != "" {
		t.Fatalf("expected no forced text before friday, got %q", plan.forceText)
	}
}

func TestFridayFollowupPlan_UsesFridayMessageOnFriday(t *testing.T) {
	loc := followupLocalTZ
	msgTime := time.Date(2026, 4, 22, 11, 0, 0, 0, loc) // Wednesday
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, loc)     // Friday

	msgs := []storage.ConversationMessage{
		{
			Role:      storage.RoleUser,
			Content:   "Háblame el viernes porfa",
			CreatedAt: msgTime,
		},
	}

	plan := fridayFollowupPlan(msgs, now)
	if plan.deferSend {
		t.Fatal("expected no friday deferral on friday")
	}
	if plan.forceText != fridayFollowupMessage {
		t.Fatalf("expected forced friday followup %q, got %q", fridayFollowupMessage, plan.forceText)
	}
}

func TestFridayFollowupPlan_DoesNotForceAfterFriday(t *testing.T) {
	loc := followupLocalTZ
	msgTime := time.Date(2026, 4, 22, 11, 0, 0, 0, loc) // Wednesday
	now := time.Date(2026, 4, 25, 10, 0, 0, 0, loc)     // Saturday

	msgs := []storage.ConversationMessage{
		{
			Role:      storage.RoleUser,
			Content:   "Me escribes el viernes",
			CreatedAt: msgTime,
		},
	}

	plan := fridayFollowupPlan(msgs, now)
	if plan.deferSend {
		t.Fatal("expected no friday deferral after friday")
	}
	if plan.forceText != "" {
		t.Fatalf("expected no forced friday text after friday, got %q", plan.forceText)
	}
}

func TestFridayFollowupPlan_IgnoresFridayWithoutFollowupVerb(t *testing.T) {
	loc := followupLocalTZ
	msgTime := time.Date(2026, 4, 22, 11, 0, 0, 0, loc)
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, loc)

	msgs := []storage.ConversationMessage{
		{
			Role:      storage.RoleUser,
			Content:   "Tengo libre el viernes",
			CreatedAt: msgTime,
		},
	}

	plan := fridayFollowupPlan(msgs, now)
	if plan.deferSend || plan.forceText != "" {
		t.Fatal("expected no friday followup plan when friday is mentioned without followup intent")
	}
}

func TestDropPastFollowupResponses(t *testing.T) {
	msgs := []storage.ConversationMessage{
		{Role: storage.RoleUser, Content: "hola", Intent: "GREETING"},
		{Role: storage.RoleAssistant, Content: "te escribo", Intent: "FOLLOWUP"},
		{Role: storage.RoleAssistant, Content: "te paso precios", Intent: "PRICE_INQUIRY"},
	}

	filtered := dropPastFollowupResponses(msgs)
	if len(filtered) != 2 {
		t.Fatalf("got %d messages, want 2", len(filtered))
	}
	if filtered[0].Role != storage.RoleUser {
		t.Fatal("expected first message to remain user message")
	}
	if filtered[1].Intent != "PRICE_INQUIRY" {
		t.Fatalf("expected non-followup assistant message to remain, got %q", filtered[1].Intent)
	}
}

func TestSimpleFollowupMessage_Default(t *testing.T) {
	lead := &storage.LeadRecord{}
	got := simpleFollowupMessage(lead)
	want := "Hola, ¿cómo estás? ¿Quieres que sigamos con precios o horarios?"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSimpleFollowupMessage_HotLead(t *testing.T) {
	lead := &storage.LeadRecord{LeadScore: 70}
	got := simpleFollowupMessage(lead)
	want := "Hola, ¿cómo estás? ¿Quieres que sigamos con la inscripción?"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSimpleFollowupMessage_BuySignal(t *testing.T) {
	lead := &storage.LeadRecord{BuySignal: true, LeadScore: 30}
	got := simpleFollowupMessage(lead)
	want := "Hola, ¿cómo estás? ¿Quieres que sigamos con la inscripción?"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSimpleFollowupMessage_PriceAndScheduleAsked(t *testing.T) {
	lead := &storage.LeadRecord{PriceAsked: true, ScheduleAsked: true}
	got := simpleFollowupMessage(lead)
	want := "Hola, ¿cómo estás? ¿Quieres que sigamos con la inscripción?"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
