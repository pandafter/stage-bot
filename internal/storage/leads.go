package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kart-academy/instagram-bot/internal/domain"
)

// LeadOutcome tracks the final result of a lead for learning.
type LeadOutcome string

const (
	OutcomeOpen      LeadOutcome = "open"
	OutcomeWon       LeadOutcome = "won"
	OutcomeLost      LeadOutcome = "lost"
	OutcomeAbandoned LeadOutcome = "abandoned"
)

type LeadRecord struct {
	ID            string
	Username      string
	TotalMessages int
	LeadScore     int
	State         domain.LeadState
	Outcome       LeadOutcome
	PriceAsked    bool
	ScheduleAsked bool
	BuySignal     bool
	Objections    int
	FollowupCount int
	LastFollowup  *time.Time
}

type LeadsRepo struct {
	db *sql.DB
}

func NewLeadsRepo(db *DB) *LeadsRepo {
	return &LeadsRepo{db: db.Conn()}
}

// Ensure creates the lead row if missing. Returns the current record.
func (r *LeadsRepo) Ensure(ctx context.Context, leadID string) (*LeadRecord, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO leads (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, leadID)
	if err != nil {
		return nil, fmt.Errorf("ensure lead: %w", err)
	}
	return r.Get(ctx, leadID)
}

func (r *LeadsRepo) Get(ctx context.Context, leadID string) (*LeadRecord, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, username, total_messages, lead_score, state, outcome,
			price_asked, schedule_asked, buy_signal, objections,
			followup_count, last_followup
		 FROM leads WHERE id = $1`, leadID)

	var rec LeadRecord
	var state, outcome string
	err := row.Scan(&rec.ID, &rec.Username, &rec.TotalMessages, &rec.LeadScore,
		&state, &outcome, &rec.PriceAsked, &rec.ScheduleAsked, &rec.BuySignal, &rec.Objections,
		&rec.FollowupCount, &rec.LastFollowup)
	if err != nil {
		return nil, fmt.Errorf("get lead: %w", err)
	}
	rec.State = domain.LeadState(state)
	rec.Outcome = LeadOutcome(outcome)
	return &rec, nil
}

// UpdateScore persists score and funnel flags after processing a message.
func (r *LeadsRepo) UpdateScore(ctx context.Context, rec *LeadRecord) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE leads SET
			total_messages = $2,
			lead_score     = $3,
			state          = $4,
			price_asked    = $5,
			schedule_asked = $6,
			buy_signal     = $7,
			objections     = $8,
			last_seen      = NOW()
		 WHERE id = $1`,
		rec.ID, rec.TotalMessages, rec.LeadScore, string(rec.State),
		rec.PriceAsked, rec.ScheduleAsked, rec.BuySignal, rec.Objections,
	)
	if err != nil {
		return fmt.Errorf("update lead score: %w", err)
	}
	return nil
}

// LeadsStats summarizes the funnel for the dashboard.
type LeadsStats struct {
	Total      int
	Open       int
	Won        int
	Lost       int
	Abandoned  int
	Hot        int     // score >= 70
	AvgScore   float64 // across all leads
	WithBuySig int     // buy_signal = true
	PriceAsked int     // price_asked = true
}

// Stats aggregates lead-funnel metrics in a single round trip.
func (r *LeadsRepo) Stats(ctx context.Context) (LeadsStats, error) {
	var s LeadsStats
	row := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE outcome = 'open'),
			COUNT(*) FILTER (WHERE outcome = 'won'),
			COUNT(*) FILTER (WHERE outcome = 'lost'),
			COUNT(*) FILTER (WHERE outcome = 'abandoned'),
			COUNT(*) FILTER (WHERE lead_score >= 70),
			COALESCE(AVG(lead_score)::float, 0),
			COUNT(*) FILTER (WHERE buy_signal = TRUE),
			COUNT(*) FILTER (WHERE price_asked = TRUE)
		FROM leads
	`)
	if err := row.Scan(&s.Total, &s.Open, &s.Won, &s.Lost, &s.Abandoned,
		&s.Hot, &s.AvgScore, &s.WithBuySig, &s.PriceAsked); err != nil {
		return s, fmt.Errorf("leads stats: %w", err)
	}
	return s, nil
}

// SetOutcome marks a lead as won/lost/abandoned so learning can sample it.
func (r *LeadsRepo) SetOutcome(ctx context.Context, leadID string, outcome LeadOutcome, notes string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE leads SET outcome = $2, notes = $3 WHERE id = $1`,
		leadID, string(outcome), notes)
	if err != nil {
		return fmt.Errorf("set outcome: %w", err)
	}
	return nil
}

// ListByOutcome returns leads with a specific outcome, newest first.
func (r *LeadsRepo) ListByOutcome(ctx context.Context, outcome LeadOutcome, limit int) ([]*LeadRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, username, total_messages, lead_score, state, outcome,
			price_asked, schedule_asked, buy_signal, objections,
			followup_count, last_followup
		 FROM leads WHERE outcome = $1 ORDER BY last_seen DESC LIMIT $2`,
		string(outcome), limit)
	if err != nil {
		return nil, fmt.Errorf("list leads: %w", err)
	}
	defer rows.Close()

	var leads []*LeadRecord
	for rows.Next() {
		var rec LeadRecord
		var state, outcomeStr string
		if err := rows.Scan(&rec.ID, &rec.Username, &rec.TotalMessages, &rec.LeadScore,
			&state, &outcomeStr, &rec.PriceAsked, &rec.ScheduleAsked, &rec.BuySignal, &rec.Objections,
			&rec.FollowupCount, &rec.LastFollowup); err != nil {
			return nil, fmt.Errorf("scan lead: %w", err)
		}
		rec.State = domain.LeadState(state)
		rec.Outcome = LeadOutcome(outcomeStr)
		leads = append(leads, &rec)
	}
	return leads, rows.Err()
}

// StaleLeads returns leads that were interested (score >= 20) but stopped responding.
// Rules:
//   - outcome = open (not won/lost/abandoned)
//   - last_seen between staleAfter and maxAge ago (e.g. 24h-7d)
//   - followup_count < maxFollowups
//   - last_followup is null OR older than cooldown (e.g. 48h between follow-ups)
func (r *LeadsRepo) StaleLeads(ctx context.Context, staleAfter, maxAge, cooldown time.Duration, maxFollowups, limit int) ([]*LeadRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, username, total_messages, lead_score, state, outcome,
			price_asked, schedule_asked, buy_signal, objections,
			followup_count, last_followup
		FROM leads
		WHERE outcome = 'open'
		  AND lead_score >= 20
		  AND total_messages >= 2
		  AND last_seen < NOW() - $1::interval
		  AND last_seen > NOW() - $2::interval
		  AND followup_count < $3
		  AND (last_followup IS NULL OR last_followup < NOW() - $4::interval)
		ORDER BY lead_score DESC, last_seen ASC
		LIMIT $5`,
		fmt.Sprintf("%d seconds", int(staleAfter.Seconds())),
		fmt.Sprintf("%d seconds", int(maxAge.Seconds())),
		maxFollowups,
		fmt.Sprintf("%d seconds", int(cooldown.Seconds())),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("stale leads: %w", err)
	}
	defer rows.Close()

	var leads []*LeadRecord
	for rows.Next() {
		var rec LeadRecord
		var state, outcomeStr string
		if err := rows.Scan(&rec.ID, &rec.Username, &rec.TotalMessages, &rec.LeadScore,
			&state, &outcomeStr, &rec.PriceAsked, &rec.ScheduleAsked, &rec.BuySignal, &rec.Objections,
			&rec.FollowupCount, &rec.LastFollowup); err != nil {
			return nil, fmt.Errorf("scan stale lead: %w", err)
		}
		rec.State = domain.LeadState(state)
		rec.Outcome = LeadOutcome(outcomeStr)
		leads = append(leads, &rec)
	}
	return leads, rows.Err()
}

// MarkFollowup increments the follow-up counter and sets last_followup to now.
func (r *LeadsRepo) MarkFollowup(ctx context.Context, leadID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE leads SET followup_count = followup_count + 1, last_followup = NOW() WHERE id = $1`,
		leadID)
	if err != nil {
		return fmt.Errorf("mark followup: %w", err)
	}
	return nil
}

// ResetFollowup clears the follow-up counter when a lead responds.
// Called from the brain pipeline when a user message arrives.
func (r *LeadsRepo) ResetFollowup(ctx context.Context, leadID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE leads SET followup_count = 0, last_followup = NULL WHERE id = $1 AND followup_count > 0`,
		leadID)
	if err != nil {
		return fmt.Errorf("reset followup: %w", err)
	}
	return nil
}

// AutoAbandon marks leads as abandoned if they've been open and idle for too long.
// Returns the number of leads marked.
func (r *LeadsRepo) AutoAbandon(ctx context.Context, idleThreshold time.Duration) (int, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE leads SET outcome = 'abandoned'
		WHERE outcome = 'open'
		  AND total_messages >= 2
		  AND last_seen < NOW() - $1::interval
		  AND followup_count >= 3`,
		fmt.Sprintf("%d seconds", int(idleThreshold.Seconds())),
	)
	if err != nil {
		return 0, fmt.Errorf("auto abandon: %w", err)
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// FollowUpStats returns metrics about follow-up effectiveness.
type FollowUpStats struct {
	TotalSent      int     // total follow-up messages sent (across all leads)
	LeadsContacted int     // leads that received at least one follow-up
	LeadsResponded int     // leads that responded after a follow-up
	ResponseRate   float64 // LeadsResponded / LeadsContacted
}

// GetFollowUpStats aggregates follow-up performance.
func (r *LeadsRepo) GetFollowUpStats(ctx context.Context) (FollowUpStats, error) {
	var s FollowUpStats
	row := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(followup_count), 0),
			COUNT(*) FILTER (WHERE followup_count > 0),
			COUNT(*) FILTER (WHERE followup_count > 0 AND last_seen > last_followup)
		FROM leads
		WHERE outcome = 'open' OR outcome = 'won' OR outcome = 'lost' OR outcome = 'abandoned'
	`)
	if err := row.Scan(&s.TotalSent, &s.LeadsContacted, &s.LeadsResponded); err != nil {
		return s, fmt.Errorf("followup stats: %w", err)
	}
	if s.LeadsContacted > 0 {
		s.ResponseRate = float64(s.LeadsResponded) / float64(s.LeadsContacted) * 100
	}
	return s, nil
}
