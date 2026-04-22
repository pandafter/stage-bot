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

func TestPromisedDayFollowupPlan_DefersBeforeTargetDay(t *testing.T) {
	loc := followupLocalTZ
	msgTime := time.Date(2026, 4, 21, 11, 0, 0, 0, loc) // Tuesday
	now := time.Date(2026, 4, 22, 15, 0, 0, 0, loc)     // Wednesday

	msgs := []storage.ConversationMessage{
		{
			Role:      storage.RoleUser,
			Content:   "Listo, escríbeme el jueves",
			CreatedAt: msgTime,
		},
	}

	plan := promisedDayFollowupPlan(msgs, now)
	if !plan.deferSend {
		t.Fatal("expected promised-day followup to defer before target day")
	}
	if plan.forceText != "" {
		t.Fatalf("expected no forced text before target day, got %q", plan.forceText)
	}
}

func TestPromisedDayFollowupPlan_UsesMessageOnTargetDay(t *testing.T) {
	loc := followupLocalTZ
	msgTime := time.Date(2026, 4, 21, 11, 0, 0, 0, loc) // Tuesday
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, loc)     // Thursday

	msgs := []storage.ConversationMessage{
		{
			Role:      storage.RoleUser,
			Content:   "Háblame el jueves porfa",
			CreatedAt: msgTime,
		},
	}

	plan := promisedDayFollowupPlan(msgs, now)
	if plan.deferSend {
		t.Fatal("expected no promised-day deferral on target day")
	}
	if plan.forceText != promisedDayFollowupReply {
		t.Fatalf("expected forced promised-day followup %q, got %q", promisedDayFollowupReply, plan.forceText)
	}
}

func TestPromisedDayFollowupPlan_DoesNotForceAfterTargetDay(t *testing.T) {
	loc := followupLocalTZ
	msgTime := time.Date(2026, 4, 21, 11, 0, 0, 0, loc) // Tuesday
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, loc)     // Friday

	msgs := []storage.ConversationMessage{
		{
			Role:      storage.RoleUser,
			Content:   "Me escribes el jueves",
			CreatedAt: msgTime,
		},
	}

	plan := promisedDayFollowupPlan(msgs, now)
	if plan.deferSend {
		t.Fatal("expected no promised-day deferral after target day")
	}
	if plan.forceText != "" {
		t.Fatalf("expected no forced promised-day text after target day, got %q", plan.forceText)
	}
}

func TestPromisedDayFollowupPlan_IgnoresDayWithoutFollowupVerb(t *testing.T) {
	loc := followupLocalTZ
	msgTime := time.Date(2026, 4, 22, 11, 0, 0, 0, loc)
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, loc)

	msgs := []storage.ConversationMessage{
		{
			Role:      storage.RoleUser,
			Content:   "Tengo libre el jueves",
			CreatedAt: msgTime,
		},
	}

	plan := promisedDayFollowupPlan(msgs, now)
	if plan.deferSend || plan.forceText != "" {
		t.Fatal("expected no promised-day followup plan when day is mentioned without followup intent")
	}
}

func TestPromisedDayFollowupPlan_HandlesAccentlessAndAccentedDays(t *testing.T) {
	loc := followupLocalTZ
	msgTime := time.Date(2026, 4, 21, 11, 0, 0, 0, loc) // Tuesday
	now := time.Date(2026, 4, 22, 10, 0, 0, 0, loc)     // Wednesday

	msgs := []storage.ConversationMessage{
		{
			Role:      storage.RoleUser,
			Content:   "Te contacto el miércoles y te confirmo",
			CreatedAt: msgTime,
		},
	}

	plan := promisedDayFollowupPlan(msgs, now)
	if plan.deferSend {
		t.Fatal("expected no deferral on promised accented weekday")
	}
	if plan.forceText != promisedDayFollowupReply {
		t.Fatalf("expected forced promised-day followup %q, got %q", promisedDayFollowupReply, plan.forceText)
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
