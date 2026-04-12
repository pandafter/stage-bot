package ai

import (
	"testing"

	"github.com/kart-academy/instagram-bot/internal/domain"
)

func TestLeadScorer_NewLead(t *testing.T) {
	scorer := NewLeadScorer()
	score := scorer.Get("user1")

	if score.State != domain.LeadStateNew {
		t.Errorf("new lead state = %s, want %s", score.State, domain.LeadStateNew)
	}
	if score.Total != 0 {
		t.Errorf("new lead score = %d, want 0", score.Total)
	}
}

func TestLeadScorer_SingleMessage(t *testing.T) {
	scorer := NewLeadScorer()
	score := scorer.Update("user1", domain.IntentGreeting)

	if score.MessageCount != 1 {
		t.Errorf("message count = %d, want 1", score.MessageCount)
	}
	// 5 (base) + 5 (greeting weight) = 10
	if score.Total != 10 {
		t.Errorf("score after greeting = %d, want 10", score.Total)
	}
}

func TestLeadScorer_PriceAndScheduleBonus(t *testing.T) {
	scorer := NewLeadScorer()

	scorer.Update("user1", domain.IntentGreeting)
	scorer.Update("user1", domain.IntentPriceInquiry)
	score := scorer.Update("user1", domain.IntentScheduleInquiry)

	if !score.PriceAsked {
		t.Error("PriceAsked should be true")
	}
	if !score.ScheduleAsked {
		t.Error("ScheduleAsked should be true")
	}
	// Should get the bonus for asking both
	// Greeting: 5+5=10, Price: 5+20=25, Schedule: 5+15+5(bonus)=25 -> total=60
	if score.Total < 55 {
		t.Errorf("score after price+schedule = %d, want >= 55", score.Total)
	}
}

func TestLeadScorer_FunnelProgression(t *testing.T) {
	scorer := NewLeadScorer()

	// New user greets
	s := scorer.Update("user1", domain.IntentGreeting)
	if s.State != domain.LeadStateNew {
		t.Errorf("after greeting: state = %s, want %s", s.State, domain.LeadStateNew)
	}

	// Asks about courses
	s = scorer.Update("user1", domain.IntentCourseInquiry)
	if s.State != domain.LeadStateEngaged {
		t.Errorf("after course inquiry: state = %s, want %s", s.State, domain.LeadStateEngaged)
	}

	// Asks price -> should jump to interested
	s = scorer.Update("user1", domain.IntentPriceInquiry)
	if s.State != domain.LeadStateInterested {
		t.Errorf("after price inquiry: state = %s, want %s", s.State, domain.LeadStateInterested)
	}

	// Asks schedule
	s = scorer.Update("user1", domain.IntentScheduleInquiry)

	// Buy signal -> should be closing or hot
	s = scorer.Update("user1", domain.IntentBuySignal)
	if s.State != domain.LeadStateHot && s.State != domain.LeadStateClosing {
		t.Errorf("after buy signal: state = %s, want hot or closing", s.State)
	}
}

func TestLeadScorer_ObjectionTracking(t *testing.T) {
	scorer := NewLeadScorer()
	scorer.Update("user1", domain.IntentGreeting)
	scorer.Update("user1", domain.IntentObjectionPrice)
	score := scorer.Update("user1", domain.IntentObjectionTime)

	if score.ObjectionsHit != 2 {
		t.Errorf("objections = %d, want 2", score.ObjectionsHit)
	}
}

func TestLeadScorer_IndependentUsers(t *testing.T) {
	scorer := NewLeadScorer()

	scorer.Update("user1", domain.IntentBuySignal)
	scorer.Update("user2", domain.IntentGreeting)

	s1 := scorer.Get("user1")
	s2 := scorer.Get("user2")

	if s1.Total == s2.Total {
		t.Error("different users should have different scores")
	}
	if s1.BuySignalSent != true {
		t.Error("user1 should have buy signal")
	}
	if s2.BuySignalSent != false {
		t.Error("user2 should not have buy signal")
	}
}

func TestSelectStrategy_Greeting(t *testing.T) {
	score := &domain.LeadScore{MessageCount: 1, State: domain.LeadStateNew}
	s := SelectStrategy(domain.IntentGreeting, score)
	if s != domain.StrategyWelcome {
		t.Errorf("first greeting strategy = %s, want %s", s, domain.StrategyWelcome)
	}

	// Returning user greeting
	score.MessageCount = 5
	s = SelectStrategy(domain.IntentGreeting, score)
	if s != domain.StrategyInform {
		t.Errorf("returning greeting strategy = %s, want %s", s, domain.StrategyInform)
	}
}

func TestSelectStrategy_Objections(t *testing.T) {
	score := &domain.LeadScore{Total: 50}
	for _, intent := range []domain.Intent{domain.IntentObjectionPrice, domain.IntentObjectionTime, domain.IntentObjectionDoubt} {
		s := SelectStrategy(intent, score)
		if s != domain.StrategyHandleObjection {
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
			score := &domain.LeadScore{Total: tt.score, MessageCount: 5}
			got := SelectStrategy(domain.IntentCourseInquiry, score)
			if got != tt.want {
				t.Errorf("strategy at score %d = %s, want %s", tt.score, got, tt.want)
			}
		})
	}
}

func TestSelectStrategy_BuySignalAlwaysCloses(t *testing.T) {
	// Even at low score, buy signal should close
	score := &domain.LeadScore{Total: 10}
	s := SelectStrategy(domain.IntentBuySignal, score)
	if s != domain.StrategyClose {
		t.Errorf("buy signal strategy = %s, want %s", s, domain.StrategyClose)
	}
}

func TestSelectStrategy_PaymentConfirms(t *testing.T) {
	score := &domain.LeadScore{Total: 50}
	s := SelectStrategy(domain.IntentPaymentConfirm, score)
	if s != domain.StrategyConfirmSale {
		t.Errorf("payment confirm strategy = %s, want %s", s, domain.StrategyConfirmSale)
	}
}
