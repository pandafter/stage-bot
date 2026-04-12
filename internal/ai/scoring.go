package ai

import (
	"github.com/kart-academy/instagram-bot/internal/domain"
	"github.com/kart-academy/instagram-bot/internal/storage"
)

// ScoreThresholds define the sales funnel stages.
const (
	ThresholdInform   = 0  // 0-30: Inform and educate
	ThresholdPersuade = 31 // 31-60: Present benefits and social proof
	ThresholdGuide    = 61 // 61-80: Guide toward decision
	ThresholdClose    = 81 // 81+: Close the sale (send payment link)
)

// ApplyIntent updates the lead record based on a new message intent.
// Returns the score delta applied so callers can log it per-message.
func ApplyIntent(rec *storage.LeadRecord, intent domain.Intent) int {
	before := rec.LeadScore

	rec.TotalMessages++
	rec.LeadScore += 5
	rec.LeadScore += IntentSalesWeight(intent)

	switch intent {
	case domain.IntentPriceInquiry:
		rec.PriceAsked = true
	case domain.IntentScheduleInquiry:
		rec.ScheduleAsked = true
	case domain.IntentBuySignal, domain.IntentPaymentConfirm:
		rec.BuySignal = true
	case domain.IntentObjectionPrice, domain.IntentObjectionTime,
		domain.IntentObjectionDoubt, domain.IntentObjectionOther:
		rec.Objections++
	}

	if rec.PriceAsked && rec.ScheduleAsked {
		rec.LeadScore += 5
	}

	rec.State = scoreToState(rec.LeadScore)
	return rec.LeadScore - before
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

// SelectStrategy picks the response strategy based on intent and lead state.
func SelectStrategy(intent domain.Intent, rec *storage.LeadRecord) domain.Strategy {
	switch intent {
	case domain.IntentGreeting:
		if rec.TotalMessages <= 1 {
			return domain.StrategyWelcome
		}
		return domain.StrategyInform

	case domain.IntentObjectionPrice, domain.IntentObjectionTime,
		domain.IntentObjectionDoubt, domain.IntentObjectionOther:
		return domain.StrategyHandleObjection

	case domain.IntentPaymentConfirm:
		return domain.StrategyConfirmSale

	case domain.IntentBuySignal:
		return domain.StrategyClose

	case domain.IntentThanks:
		if rec.BuySignal {
			return domain.StrategyConfirmSale
		}
		return domain.StrategyInform

	case domain.IntentOffTopic:
		return domain.StrategyRedirect
	}

	switch {
	case rec.LeadScore >= ThresholdClose:
		return domain.StrategyClose
	case rec.LeadScore >= ThresholdGuide:
		return domain.StrategyGuide
	case rec.LeadScore >= ThresholdPersuade:
		return domain.StrategyPersuade
	default:
		return domain.StrategyInform
	}
}
