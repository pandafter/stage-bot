package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
	"go.uber.org/zap"
)

type DB struct {
	conn   *sql.DB
	logger *zap.Logger
}

func NewDB(dbPath string, logger *zap.Logger) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON", dbPath)

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	conn.SetMaxOpenConns(1) // SQLite: single writer
	conn.SetMaxIdleConns(5)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	db := &DB{conn: conn, logger: logger}

	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	logger.Info("database initialized", zap.String("path", dbPath))
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
			username       TEXT,
			first_seen     DATETIME NOT NULL DEFAULT (datetime('now')),
			last_seen      DATETIME NOT NULL DEFAULT (datetime('now')),
			total_messages INTEGER DEFAULT 0,
			lead_score     INTEGER DEFAULT 0,
			state          TEXT DEFAULT 'new',
			purchased      TEXT DEFAULT '[]',
			notes          TEXT DEFAULT ''
		)`,

		`CREATE TABLE IF NOT EXISTS messages (
			id            TEXT PRIMARY KEY,
			lead_id       TEXT NOT NULL,
			direction     TEXT NOT NULL,
			content_type  TEXT NOT NULL,
			content       TEXT NOT NULL,
			intent        TEXT DEFAULT '',
			score_delta   INTEGER DEFAULT 0,
			created_at    DATETIME NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (lead_id) REFERENCES leads(id)
		)`,

		`CREATE TABLE IF NOT EXISTS analytics (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type  TEXT NOT NULL,
			lead_id     TEXT DEFAULT '',
			data        TEXT DEFAULT '{}',
			created_at  DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE INDEX IF NOT EXISTS idx_messages_lead ON messages(lead_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_leads_state ON leads(state)`,
		`CREATE INDEX IF NOT EXISTS idx_analytics_type ON analytics(event_type, created_at)`,
	}

	for _, m := range migrations {
		if _, err := db.conn.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	db.logger.Info("database migrations complete")
	return nil
}
