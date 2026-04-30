package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type DB struct {
	Pool   *pgxpool.Pool
	logger *zap.Logger
}

func New(ctx context.Context, dsn string, logger *zap.Logger) (*DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("empty DATABASE_URL")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect pgx pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	db := &DB{Pool: pool, logger: logger}
	if err := db.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}
	logger.Info("postgres initialized")
	return db, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}

func (db *DB) migrate(ctx context.Context) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS inscripciones (
			id               TEXT PRIMARY KEY,
			email            TEXT NOT NULL,
			metodo_pago      TEXT NOT NULL,
			fecha_curso      TEXT NOT NULL,
			plan             TEXT NOT NULL,
			monto_cop        INTEGER NOT NULL DEFAULT 0,
			nombre_piloto    TEXT NOT NULL,
			edad             INTEGER NOT NULL DEFAULT 0,
			tipo_documento   TEXT NOT NULL,
			numero_documento TEXT NOT NULL,
			telefono         TEXT NOT NULL,
			ciudad           TEXT NOT NULL DEFAULT '',
			eps              TEXT NOT NULL,
			grupo_sanguineo  TEXT NOT NULL,
			familiar_nombre  TEXT NOT NULL,
			familiar_telefono TEXT NOT NULL,
			instagram_user   TEXT NOT NULL DEFAULT '',
			comprobante_path TEXT NOT NULL DEFAULT '',
			status           TEXT NOT NULL DEFAULT 'pendiente',
			created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_inscripciones_status ON inscripciones(status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_inscripciones_email ON inscripciones(email)`,
	}
	for _, m := range migrations {
		if _, err := db.Pool.Exec(ctx, m); err != nil {
			return fmt.Errorf("migration: %w\nSQL: %s", err, m)
		}
	}
	db.logger.Info("postgres migrations complete")
	return nil
}
