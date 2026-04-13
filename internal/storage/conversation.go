package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

type ConversationMessage struct {
	ID          int64
	LeadID      string
	Role        MessageRole
	Content     string
	ContentType string
	Intent      string
	Strategy    string
	ScoreDelta  int
	ScoreAfter  int
	TokensIn    int
	TokensOut   int
	Model       string
	CreatedAt   time.Time
}

type ConversationRepo struct {
	db *sql.DB
}

func NewConversationRepo(db *DB) *ConversationRepo {
	return &ConversationRepo{db: db.Conn()}
}

// Append stores a single message. The DB assigns id and created_at.
func (r *ConversationRepo) Append(ctx context.Context, m ConversationMessage) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO conversation_messages
		 (lead_id, role, content, content_type, intent, strategy, score_delta, score_after, tokens_in, tokens_out, model)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		m.LeadID, string(m.Role), m.Content, m.ContentType,
		m.Intent, m.Strategy, m.ScoreDelta, m.ScoreAfter,
		m.TokensIn, m.TokensOut, m.Model,
	)
	if err != nil {
		return fmt.Errorf("append message: %w", err)
	}
	return nil
}

// LastN returns the N most recent messages for a lead, in chronological order.
func (r *ConversationRepo) LastN(ctx context.Context, leadID string, n int) ([]ConversationMessage, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, lead_id, role, content, content_type, intent, strategy,
		        score_delta, score_after, created_at
		 FROM (
		   SELECT * FROM conversation_messages
		   WHERE lead_id = $1
		   ORDER BY created_at DESC, id DESC
		   LIMIT $2
		 ) sub
		 ORDER BY created_at ASC, id ASC`,
		leadID, n)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var out []ConversationMessage
	for rows.Next() {
		var m ConversationMessage
		var role string
		if err := rows.Scan(&m.ID, &m.LeadID, &role, &m.Content, &m.ContentType,
			&m.Intent, &m.Strategy, &m.ScoreDelta, &m.ScoreAfter, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		m.Role = MessageRole(role)
		out = append(out, m)
	}
	return out, rows.Err()
}

// UsageWindow holds aggregate token/message stats over a time window.
type UsageWindow struct {
	Messages         int
	AssistantReplies int
	UserMessages     int
	TokensIn         int64
	TokensOut        int64
}

// UsageStats aggregates message + token counts across several windows in one query.
type UsageStats struct {
	Last24h UsageWindow
	Last7d  UsageWindow
	Total   UsageWindow
}

// AggregateUsage returns message + token counts for 24h, 7d, and all-time.
func (r *ConversationRepo) AggregateUsage(ctx context.Context) (UsageStats, error) {
	var s UsageStats
	row := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours'),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours' AND role='assistant'),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours' AND role='user'),
			COALESCE(SUM(tokens_in)  FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours'), 0),
			COALESCE(SUM(tokens_out) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours'), 0),

			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '7 days'),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '7 days' AND role='assistant'),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '7 days' AND role='user'),
			COALESCE(SUM(tokens_in)  FILTER (WHERE created_at >= NOW() - INTERVAL '7 days'), 0),
			COALESCE(SUM(tokens_out) FILTER (WHERE created_at >= NOW() - INTERVAL '7 days'), 0),

			COUNT(*),
			COUNT(*) FILTER (WHERE role='assistant'),
			COUNT(*) FILTER (WHERE role='user'),
			COALESCE(SUM(tokens_in), 0),
			COALESCE(SUM(tokens_out), 0)
		FROM conversation_messages
	`)
	if err := row.Scan(
		&s.Last24h.Messages, &s.Last24h.AssistantReplies, &s.Last24h.UserMessages, &s.Last24h.TokensIn, &s.Last24h.TokensOut,
		&s.Last7d.Messages, &s.Last7d.AssistantReplies, &s.Last7d.UserMessages, &s.Last7d.TokensIn, &s.Last7d.TokensOut,
		&s.Total.Messages, &s.Total.AssistantReplies, &s.Total.UserMessages, &s.Total.TokensIn, &s.Total.TokensOut,
	); err != nil {
		return s, fmt.Errorf("usage stats: %w", err)
	}
	return s, nil
}

// LeadCost is the per-lead aggregated token spend used to rank by cost.
type LeadCost struct {
	LeadID    string
	Messages  int
	TokensIn  int64
	TokensOut int64
}

// TopLeadsByCost returns the most expensive leads by total tokens, newest activity first as tie-breaker.
func (r *ConversationRepo) TopLeadsByCost(ctx context.Context, limit int) ([]LeadCost, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT lead_id,
		       COUNT(*) FILTER (WHERE role='assistant'),
		       COALESCE(SUM(tokens_in), 0),
		       COALESCE(SUM(tokens_out), 0)
		FROM conversation_messages
		GROUP BY lead_id
		HAVING COALESCE(SUM(tokens_in), 0) + COALESCE(SUM(tokens_out), 0) > 0
		ORDER BY (COALESCE(SUM(tokens_in), 0) + COALESCE(SUM(tokens_out), 0)) DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("top leads by cost: %w", err)
	}
	defer rows.Close()

	var out []LeadCost
	for rows.Next() {
		var lc LeadCost
		if err := rows.Scan(&lc.LeadID, &lc.Messages, &lc.TokensIn, &lc.TokensOut); err != nil {
			return nil, fmt.Errorf("scan lead cost: %w", err)
		}
		out = append(out, lc)
	}
	return out, rows.Err()
}

// FullForLead returns every message for a lead in chronological order.
// Used by the learning layer when sampling won/lost deals.
func (r *ConversationRepo) FullForLead(ctx context.Context, leadID string) ([]ConversationMessage, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, lead_id, role, content, content_type, intent, strategy,
		        score_delta, score_after, created_at
		 FROM conversation_messages
		 WHERE lead_id = $1
		 ORDER BY created_at ASC, id ASC`, leadID)
	if err != nil {
		return nil, fmt.Errorf("query full conversation: %w", err)
	}
	defer rows.Close()

	var out []ConversationMessage
	for rows.Next() {
		var m ConversationMessage
		var role string
		if err := rows.Scan(&m.ID, &m.LeadID, &role, &m.Content, &m.ContentType,
			&m.Intent, &m.Strategy, &m.ScoreDelta, &m.ScoreAfter, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		m.Role = MessageRole(role)
		out = append(out, m)
	}
	return out, rows.Err()
}
