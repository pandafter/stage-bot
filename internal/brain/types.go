package brain

// Intent represents the classified intention of a user message.
type Intent string

const (
	IntentGreeting        Intent = "GREETING"
	IntentCourseInquiry   Intent = "COURSE_INQUIRY"
	IntentPriceInquiry    Intent = "PRICE_INQUIRY"
	IntentScheduleInquiry Intent = "SCHEDULE_INQUIRY"
	IntentRequirements    Intent = "REQUIREMENTS"
	IntentLocationInquiry Intent = "LOCATION_INQUIRY"
	IntentBuySignal       Intent = "BUY_SIGNAL"
	IntentObjectionPrice  Intent = "OBJECTION_PRICE"
	IntentObjectionTime   Intent = "OBJECTION_TIME"
	IntentObjectionDoubt  Intent = "OBJECTION_DOUBT"
	IntentObjectionOther  Intent = "OBJECTION_OTHER"
	IntentPaymentConfirm  Intent = "PAYMENT_CONFIRM"
	IntentThanks          Intent = "THANKS"
	IntentOffTopic        Intent = "OFF_TOPIC"
	IntentVoiceMessage    Intent = "VOICE_MESSAGE"
	IntentUnknown         Intent = "UNKNOWN"
)

// Strategy represents the sales strategy to use for the response.
type Strategy string

const (
	StrategyWelcome         Strategy = "WELCOME"
	StrategyInform          Strategy = "INFORM"
	StrategyPersuade        Strategy = "PERSUADE"
	StrategyGuide           Strategy = "GUIDE"
	StrategyClose           Strategy = "CLOSE"
	StrategyHandleObjection Strategy = "HANDLE_OBJECTION"
	StrategyUpsell          Strategy = "UPSELL"
	StrategyRedirect        Strategy = "REDIRECT"
	StrategyConfirmSale     Strategy = "CONFIRM_SALE"
)

// LeadState represents the current state of a lead in the sales funnel.
type LeadState string

const (
	LeadStateNew        LeadState = "new"
	LeadStateEngaged    LeadState = "engaged"
	LeadStateInterested LeadState = "interested"
	LeadStateHot        LeadState = "hot"
	LeadStateClosing    LeadState = "closing"
	LeadStateCustomer   LeadState = "customer"
	LeadStateInactive   LeadState = "inactive"
)

// Response is the output of the brain pipeline.
type Response struct {
	Text     string
	Strategy Strategy
	Intent   Intent
	Score    int
}

// ConversationContext holds all the context needed for the brain to decide.
type ConversationContext struct {
	LeadID        string
	LeadState     LeadState
	LeadScore     int
	History       string
	TotalMessages int
}
