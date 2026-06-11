package storage

import (
	"context"
	"time"
)

type BotLead struct {
	SenderID      string
	FirstMsgAt    time.Time
	LastSentAt    time.Time
	FollowupCount int
}

type BotLeadsRepo struct{ db *DB }

func NewBotLeadsRepo(db *DB) *BotLeadsRepo { return &BotLeadsRepo{db: db} }

// IsKnown returns true if we have already auto-replied to this sender.
func (r *BotLeadsRepo) IsKnown(ctx context.Context, senderID string) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM bot_leads WHERE sender_id=$1)`, senderID,
	).Scan(&exists)
	return exists, err
}

// Upsert inserts a new lead or updates last_sent_at if we just sent again.
func (r *BotLeadsRepo) Upsert(ctx context.Context, senderID string) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO bot_leads (sender_id) VALUES ($1)
		 ON CONFLICT (sender_id) DO NOTHING`,
		senderID,
	)
	return err
}

// RecordFollowup updates last_sent_at and increments followup_count for a lead.
func (r *BotLeadsRepo) RecordFollowup(ctx context.Context, senderID string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE bot_leads
		 SET last_sent_at = NOW(), followup_count = followup_count + 1
		 WHERE sender_id = $1`,
		senderID,
	)
	return err
}

// DueForFollowup returns leads whose last_sent_at is older than the given interval
// and whose followup_count is below maxFollowups.
func (r *BotLeadsRepo) DueForFollowup(ctx context.Context, interval time.Duration, maxFollowups int) ([]BotLead, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT sender_id, first_msg_at, last_sent_at, followup_count
		 FROM bot_leads
		 WHERE last_sent_at < NOW() - ($1 * INTERVAL '1 second')
		   AND followup_count < $2`,
		int(interval.Seconds()), maxFollowups,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BotLead
	for rows.Next() {
		var l BotLead
		if err := rows.Scan(&l.SenderID, &l.FirstMsgAt, &l.LastSentAt, &l.FollowupCount); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
