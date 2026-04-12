package ai

import (
	"testing"

	"github.com/kart-academy/instagram-bot/internal/domain"
	"github.com/kart-academy/instagram-bot/internal/storage"
)

func newRecord() *storage.LeadRecord {
	return &storage.LeadRecord{State: domain.LeadStateNew}
}

func TestApplyIntent_SingleMessage(t *testing.T) {
	rec := newRecord()
	ApplyIntent(rec, domain.IntentGreeting)

	if rec.TotalMessages != 1 {
		t.Errorf("message count = %d, want 1", rec.TotalMessages)
	}
	if rec.LeadScore != 10 {
		t.Errorf("score after greeting = %d, want 10", rec.LeadScore)
	}
}

func TestApplyIntent_PriceAndScheduleBonus(t *testing.T) {
	rec := newRecord()
	ApplyIntent(rec, domain.IntentGreeting)
	ApplyIntent(rec, domain.IntentPriceInquiry)
	ApplyIntent(rec, domain.IntentScheduleInquiry)

	if !rec.PriceAsked {
		t.Error("PriceAsked should be true")
	}
	if !rec.ScheduleAsked {
		t.Error("ScheduleAsked should be true")
	}
	if rec.LeadScore < 55 {
		t.Errorf("score after price+schedule = %d, want >= 55", rec.LeadScore)
	}
}

func TestApplyIntent_FunnelProgression(t *testing.T) {
	rec := newRecord()

	ApplyIntent(rec, domain.IntentGreeting)
	if rec.State != domain.LeadStateNew {
		t.Errorf("after greeting: state = %s, want %s", rec.State, domain.LeadStateNew)
	}

	ApplyIntent(rec, domain.IntentCourseInquiry)
	if rec.State != domain.LeadStateEngaged {
		t.Errorf("after course inquiry: state = %s, want %s", rec.State, domain.LeadStateEngaged)
	}

	ApplyIntent(rec, domain.IntentPriceInquiry)
	if rec.State != domain.LeadStateInterested {
		t.Errorf("after price inquiry: state = %s, want %s", rec.State, domain.LeadStateInterested)
	}

	ApplyIntent(rec, domain.IntentScheduleInquiry)
	ApplyIntent(rec, domain.IntentBuySignal)
	if rec.State != domain.LeadStateHot && rec.State != domain.LeadStateClosing {
		t.Errorf("after buy signal: state = %s, want hot or closing", rec.State)
	}
}

func TestApplyIntent_ObjectionTracking(t *testing.T) {
	rec := newRecord()
	ApplyIntent(rec, domain.IntentGreeting)
	ApplyIntent(rec, domain.IntentObjectionPrice)
	ApplyIntent(rec, domain.IntentObjectionTime)

	if rec.Objections != 2 {
		t.Errorf("objections = %d, want 2", rec.Objections)
	}
}

func TestSelectStrategy_Greeting(t *testing.T) {
	rec := &storage.LeadRecord{TotalMessages: 1, State: domain.LeadStateNew}
	if s := SelectStrategy(domain.IntentGreeting, rec); s != domain.StrategyWelcome {
		t.Errorf("first greeting strategy = %s, want %s", s, domain.StrategyWelcome)
	}

	rec.TotalMessages = 5
	if s := SelectStrategy(domain.IntentGreeting, rec); s != domain.StrategyInform {
		t.Errorf("returning greeting strategy = %s, want %s", s, domain.StrategyInform)
	}
}

func TestSelectStrategy_Objections(t *testing.T) {
	rec := &storage.LeadRecord{LeadScore: 50}
	for _, intent := range []domain.Intent{domain.IntentObjectionPrice, domain.IntentObjectionTime, domain.IntentObjectionDoubt} {
		if s := SelectStrategy(intent, rec); s != domain.StrategyHandleObjection {
			t.Errorf("strategy for %s = %s, want %s", intent, s, domain.StrategyHandleObjection)
		}
	}
}

func TestSelectStrategy_ScoreDriven(t *testing.T) {
	tests := []struct {
		name  string
		score int
		want  domain.Strategy
	}{
		{"low score informs", 15, domain.StrategyInform},
		{"mid score persuades", 45, domain.StrategyPersuade},
		{"high score guides", 70, domain.StrategyGuide},
		{"very high closes", 90, domain.StrategyClose},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &storage.LeadRecord{LeadScore: tt.score, TotalMessages: 5}
			got := SelectStrategy(domain.IntentCourseInquiry, rec)
			if got != tt.want {
				t.Errorf("strategy at score %d = %s, want %s", tt.score, got, tt.want)
			}
		})
	}
}

func TestSelectStrategy_BuySignalAlwaysCloses(t *testing.T) {
	rec := &storage.LeadRecord{LeadScore: 10}
	if s := SelectStrategy(domain.IntentBuySignal, rec); s != domain.StrategyClose {
		t.Errorf("buy signal strategy = %s, want %s", s, domain.StrategyClose)
	}
}

func TestSelectStrategy_PaymentConfirms(t *testing.T) {
	rec := &storage.LeadRecord{LeadScore: 50}
	if s := SelectStrategy(domain.IntentPaymentConfirm, rec); s != domain.StrategyConfirmSale {
		t.Errorf("payment confirm strategy = %s, want %s", s, domain.StrategyConfirmSale)
	}
}
