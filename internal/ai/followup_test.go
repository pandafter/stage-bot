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

