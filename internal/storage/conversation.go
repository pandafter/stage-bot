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
		 (lead_id, role, content, content_type, intent, strategy, score_delta, score_after)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		m.LeadID, string(m.Role), m.Content, m.ContentType,
		m.Intent, m.Strategy, m.ScoreDelta, m.ScoreAfter,
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
