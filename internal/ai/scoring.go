package ai

import (
	"sync"

	"github.com/kart-academy/instagram-bot/internal/domain"
)

// ScoreThresholds define the sales funnel stages.
const (
	ThresholdInform   = 0  // 0-30: Inform and educate
	ThresholdPersuade = 31 // 31-60: Present benefits and social proof
	ThresholdGuide    = 61 // 61-80: Guide toward decision
	ThresholdClose    = 81 // 81+: Close the sale (send payment link)
)

// LeadScorer tracks and calculates lead scores per sender.
type LeadScorer struct {
	scores map[string]*domain.LeadScore
	mu     sync.RWMutex
}

func NewLeadScorer() *LeadScorer {
	return &LeadScorer{
		scores: make(map[string]*domain.LeadScore),
	}
}

// Update processes a new message intent and updates the lead's score.
func (ls *LeadScorer) Update(senderID string, intent domain.Intent) *domain.LeadScore {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	score, ok := ls.scores[senderID]
	if !ok {
		score = &domain.LeadScore{State: domain.LeadStateNew}
		ls.scores[senderID] = score
	}

	score.MessageCount++
	score.IntentHistory = append(score.IntentHistory, intent)

	// Base points per message
	score.Total += 5

	// Intent-specific scoring
	score.Total += IntentSalesWeight(intent)

	// Track key signals
	switch intent {
	case domain.IntentPriceInquiry:
		score.PriceAsked = true
	case domain.IntentScheduleInquiry:
		score.ScheduleAsked = true
	case domain.IntentBuySignal, domain.IntentPaymentConfirm:
		score.BuySignalSent = true
	case domain.IntentObjectionPrice, domain.IntentObjectionTime, domain.IntentObjectionDoubt, domain.IntentObjectionOther:
		score.ObjectionsHit++
	}

	// Bonus: asked both price AND schedule = very interested
	if score.PriceAsked && score.ScheduleAsked {
		score.Total += 5
	}

	// Update lead state based on score
	score.State = scoreToState(score.Total)

	return score
}

// Get returns the current score for a sender (read-only).
func (ls *LeadScorer) Get(senderID string) *domain.LeadScore {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	if s, ok := ls.scores[senderID]; ok {
		return s
	}
	return &domain.LeadScore{State: domain.LeadStateNew}
}

// scoreToState maps a numeric score to a funnel state.
func scoreToState(total int) domain.LeadState {
	switch {
	case total >= ThresholdClose:
		return domain.LeadStateClosing
	case total >= ThresholdGuide:
		return domain.LeadStateHot
	case total >= ThresholdPersuade:
		return domain.LeadStateInterested
	case total > 10:
		return domain.LeadStateEngaged
	default:
		return domain.LeadStateNew
	}
}

// SelectStrategy picks the response strategy based on intent and lead score.
func SelectStrategy(intent domain.Intent, score *domain.LeadScore) domain.Strategy {
	// Intent-driven overrides
	switch intent {
	case domain.IntentGreeting:
		if score.MessageCount <= 1 {
			return domain.StrategyWelcome
		}
		return domain.StrategyInform

	case domain.IntentObjectionPrice, domain.IntentObjectionTime, domain.IntentObjectionDoubt, domain.IntentObjectionOther:
		return domain.StrategyHandleObjection

	case domain.IntentPaymentConfirm:
		return domain.StrategyConfirmSale

	case domain.IntentBuySignal:
		return domain.StrategyClose

	case domain.IntentThanks:
		if score.BuySignalSent {
			return domain.StrategyConfirmSale
		}
		return domain.StrategyInform

	case domain.IntentOffTopic:
		return domain.StrategyRedirect
	}

	// Score-driven strategy
	switch {
	case score.Total >= ThresholdClose:
		return domain.StrategyClose
	case score.Total >= ThresholdGuide:
		return domain.StrategyGuide
	case score.Total >= ThresholdPersuade:
		return domain.StrategyPersuade
	default:
		return domain.StrategyInform
	}
}
