package queue

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// PostgresQueue is the Postgres-backed implementation of Queue.
// Uses FOR UPDATE SKIP LOCKED for concurrent-safe claims.
type PostgresQueue struct {
	db *sql.DB
}

func NewPostgres(db *sql.DB) *PostgresQueue {
	return &PostgresQueue{db: db}
}

func (q *PostgresQueue) Enqueue(ctx context.Context, senderID string, payload []byte) (int64, error) {
	var id int64
	err := q.db.QueryRowContext(ctx,
		`INSERT INTO message_jobs (sender_id, payload) VALUES ($1, $2) RETURNING id`,
		senderID, payload).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("enqueue: %w", err)
	}
	return id, nil
}

// Claim picks the oldest pending job whose sender doesn't already have one processing.
// This guarantees per-sender ordering while letting different senders run in parallel.
func (q *PostgresQueue) Claim(ctx context.Context) (*Job, error) {
	query := `
		UPDATE message_jobs SET
			status = 'processing',
			attempts = attempts + 1,
			started_at = NOW()
		WHERE id = (
			SELECT id FROM message_jobs j
			WHERE status = 'pending'
			  AND next_retry_at <= NOW()
			  AND NOT EXISTS (
			    SELECT 1 FROM message_jobs p
			    WHERE p.sender_id = j.sender_id AND p.status = 'processing'
			  )
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, sender_id, payload, status, attempts, next_retry_at,
		          error_text, created_at, started_at, finished_at
	`
	row := q.db.QueryRowContext(ctx, query)
	job, err := scanJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("claim: %w", err)
	}
	return job, nil
}

func (q *PostgresQueue) Complete(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE message_jobs SET status = 'done', finished_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("complete: %w", err)
	}
	return nil
}

// Fail schedules exponential backoff or marks dead after maxAttempts.
// Backoff: 30s, 2m, 10m for attempts 1, 2, 3.
func (q *PostgresQueue) Fail(ctx context.Context, id int64, errText string, maxAttempts int) error {
	var attempts int
	err := q.db.QueryRowContext(ctx,
		`SELECT attempts FROM message_jobs WHERE id = $1`, id).Scan(&attempts)
	if err != nil {
		return fmt.Errorf("fail lookup: %w", err)
	}

	if attempts >= maxAttempts {
		_, err := q.db.ExecContext(ctx,
			`UPDATE message_jobs SET status = 'dead', error_text = $2, finished_at = NOW() WHERE id = $1`,
			id, truncateErr(errText))
		if err != nil {
			return fmt.Errorf("fail->dead: %w", err)
		}
		return nil
	}

	backoff := backoffFor(attempts)
	_, err = q.db.ExecContext(ctx,
		`UPDATE message_jobs SET
		    status = 'pending',
		    error_text = $2,
		    next_retry_at = NOW() + $3::interval
		 WHERE id = $1`,
		id, truncateErr(errText), fmt.Sprintf("%d seconds", int(backoff.Seconds())))
	if err != nil {
		return fmt.Errorf("fail->retry: %w", err)
	}
	return nil
}

func (q *PostgresQueue) Stats(ctx context.Context) (Stats, error) {
	var s Stats
	rows, err := q.db.QueryContext(ctx,
		`SELECT status, COUNT(*) FROM message_jobs
		 WHERE status IN ('pending','processing','failed','dead')
		 GROUP BY status`)
	if err != nil {
		return s, fmt.Errorf("stats counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return s, err
		}
		switch status {
		case StatusPending:
			s.Pending = n
		case StatusProcessing:
			s.Processing = n
		case StatusFailed:
			s.Failed = n
		case StatusDead:
			s.Dead = n
		}
	}

	_ = q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM message_jobs WHERE status='done' AND finished_at > NOW() - INTERVAL '24 hours'`,
	).Scan(&s.DoneLast24h)

	var oldestPending sql.NullFloat64
	_ = q.db.QueryRowContext(ctx,
		`SELECT EXTRACT(EPOCH FROM (NOW() - MIN(created_at))) FROM message_jobs WHERE status='pending'`,
	).Scan(&oldestPending)
	if oldestPending.Valid {
		s.OldestPendingSeconds = int(oldestPending.Float64)
	}

	return s, nil
}

func (q *PostgresQueue) RecoverStuck(ctx context.Context, olderThan time.Duration) (int, error) {
	res, err := q.db.ExecContext(ctx,
		`UPDATE message_jobs SET status='pending', next_retry_at=NOW()
		 WHERE status='processing' AND started_at < NOW() - $1::interval`,
		fmt.Sprintf("%d seconds", int(olderThan.Seconds())))
	if err != nil {
		return 0, fmt.Errorf("recover: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (q *PostgresQueue) DeadLetter(ctx context.Context, limit int) ([]*Job, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT id, sender_id, payload, status, attempts, next_retry_at,
		        error_text, created_at, started_at, finished_at
		 FROM message_jobs WHERE status='dead' ORDER BY finished_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("dead letter query: %w", err)
	}
	defer rows.Close()

	var out []*Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (q *PostgresQueue) RetryJob(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE message_jobs SET status='pending', attempts=0, next_retry_at=NOW(), error_text='' WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("retry: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJob(r rowScanner) (*Job, error) {
	var j Job
	var started, finished sql.NullTime
	err := r.Scan(&j.ID, &j.SenderID, &j.Payload, &j.Status, &j.Attempts,
		&j.NextRetryAt, &j.ErrorText, &j.CreatedAt, &started, &finished)
	if err != nil {
		return nil, err
	}
	if started.Valid {
		j.StartedAt = &started.Time
	}
	if finished.Valid {
		j.FinishedAt = &finished.Time
	}
	return &j, nil
}

func backoffFor(attempts int) time.Duration {
	switch attempts {
	case 1:
		return 30 * time.Second
	case 2:
		return 2 * time.Minute
	default:
		return 10 * time.Minute
	}
}

func truncateErr(s string) string {
	if len(s) > 500 {
		return s[:500]
	}
	return s
}
