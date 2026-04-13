package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

type DB struct {
	conn   *sql.DB
	logger *zap.Logger
}

func NewDB(dsn string, logger *zap.Logger) (*DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("empty DATABASE_URL")
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(2)
	conn.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	db := &DB{conn: conn, logger: logger}

	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	logger.Info("postgres initialized")
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Conn() *sql.DB {
	return db.conn
}

func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS leads (
			id             TEXT PRIMARY KEY,
			username       TEXT DEFAULT '',
			first_seen     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_seen      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			total_messages INTEGER NOT NULL DEFAULT 0,
			lead_score     INTEGER NOT NULL DEFAULT 0,
			state          TEXT NOT NULL DEFAULT 'new',
			outcome        TEXT NOT NULL DEFAULT 'open',
			price_asked    BOOLEAN NOT NULL DEFAULT FALSE,
			schedule_asked BOOLEAN NOT NULL DEFAULT FALSE,
			buy_signal     BOOLEAN NOT NULL DEFAULT FALSE,
			objections     INTEGER NOT NULL DEFAULT 0,
			notes          TEXT NOT NULL DEFAULT ''
		)`,

		`CREATE TABLE IF NOT EXISTS conversation_messages (
			id           BIGSERIAL PRIMARY KEY,
			lead_id      TEXT NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
			role         TEXT NOT NULL,
			content      TEXT NOT NULL,
			content_type TEXT NOT NULL DEFAULT 'text',
			intent       TEXT NOT NULL DEFAULT '',
			strategy     TEXT NOT NULL DEFAULT '',
			score_delta  INTEGER NOT NULL DEFAULT 0,
			score_after  INTEGER NOT NULL DEFAULT 0,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS sales_events (
			id         BIGSERIAL PRIMARY KEY,
			lead_id    TEXT NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
			event_type TEXT NOT NULL,
			data       JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE INDEX IF NOT EXISTS idx_conv_lead_time ON conversation_messages(lead_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_leads_outcome ON leads(outcome)`,
		`CREATE INDEX IF NOT EXISTS idx_leads_state ON leads(state)`,
		`CREATE INDEX IF NOT EXISTS idx_sales_lead ON sales_events(lead_id, created_at)`,

		`CREATE TABLE IF NOT EXISTS message_jobs (
			id            BIGSERIAL PRIMARY KEY,
			sender_id     TEXT NOT NULL,
			payload       JSONB NOT NULL,
			status        TEXT NOT NULL DEFAULT 'pending',
			attempts      INTEGER NOT NULL DEFAULT 0,
			next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			error_text    TEXT NOT NULL DEFAULT '',
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			started_at    TIMESTAMPTZ,
			finished_at   TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_pending ON message_jobs(next_retry_at) WHERE status = 'pending'`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_sender_status ON message_jobs(sender_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_status_created ON message_jobs(status, created_at)`,

		`ALTER TABLE conversation_messages ADD COLUMN IF NOT EXISTS tokens_in INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE conversation_messages ADD COLUMN IF NOT EXISTS tokens_out INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE conversation_messages ADD COLUMN IF NOT EXISTS model TEXT NOT NULL DEFAULT ''`,
	}

	for _, m := range migrations {
		if _, err := db.conn.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	db.logger.Info("postgres migrations complete")
	return nil
}
