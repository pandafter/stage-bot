package queue

import (
	"context"
	"time"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusDone       = "done"
	StatusFailed     = "failed"
	StatusDead       = "dead"
)

// Job is a unit of work claimed from the queue.
type Job struct {
	ID          int64
	SenderID    string
	Payload     []byte
	Status      string
	Attempts    int
	NextRetryAt time.Time
	ErrorText   string
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

// Stats summarises queue depth across statuses.
type Stats struct {
	Pending      int
	Processing   int
	Failed       int
	Dead         int
	DoneLast24h  int
	OldestPendingSeconds int
}

// Queue is the abstraction over the underlying store (Postgres today, Redis later).
type Queue interface {
	Enqueue(ctx context.Context, senderID string, payload []byte) (int64, error)

	// Claim atomically grabs the next pending job, excluding senders that already
	// have a processing job (per-sender serialisation). Returns (nil, nil) if empty.
	Claim(ctx context.Context) (*Job, error)

	Complete(ctx context.Context, id int64) error

	// Fail marks a job failed, scheduling retry if attempts < maxAttempts, else dead.
	Fail(ctx context.Context, id int64, errText string, maxAttempts int) error

	Stats(ctx context.Context) (Stats, error)

	// RecoverStuck resets processing jobs older than threshold back to pending.
	// Called on startup to heal after crashes.
	RecoverStuck(ctx context.Context, olderThan time.Duration) (int, error)

	// DeadLetter returns recent dead jobs (for inspection in admin panel).
	DeadLetter(ctx context.Context, limit int) ([]*Job, error)

	// RetryJob moves a dead/failed job back to pending.
	RetryJob(ctx context.Context, id int64) error
}
