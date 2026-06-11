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
		`CREATE TABLE IF NOT EXISTS tenants (
			id           TEXT PRIMARY KEY,
			name         TEXT NOT NULL,
			domain       TEXT UNIQUE,
			theme        JSONB NOT NULL DEFAULT '{}',
			features     JSONB NOT NULL DEFAULT '{}',
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`INSERT INTO tenants (id, name, domain)
		 VALUES ('st4ge', 'Scudería St4ge', 'scuderiast4ge.com')
		 ON CONFLICT (id) DO NOTHING`,
		`CREATE TABLE IF NOT EXISTS admin_users (
			id            BIGSERIAL PRIMARY KEY,
			tenant_id     TEXT NOT NULL REFERENCES tenants(id),
			username      TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			role          TEXT NOT NULL DEFAULT 'owner',
			last_login_at TIMESTAMPTZ,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (tenant_id, username)
		)`,
		`CREATE TABLE IF NOT EXISTS cms_sections (
			tenant_id     TEXT NOT NULL REFERENCES tenants(id),
			section_key   TEXT NOT NULL,
			data          JSONB NOT NULL,
			is_published  BOOLEAN NOT NULL DEFAULT TRUE,
			version       INTEGER NOT NULL DEFAULT 1,
			updated_by    BIGINT REFERENCES admin_users(id),
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (tenant_id, section_key)
		)`,
		`CREATE TABLE IF NOT EXISTS cms_media (
			id           BIGSERIAL PRIMARY KEY,
			tenant_id    TEXT NOT NULL REFERENCES tenants(id),
			filename     TEXT NOT NULL,
			url          TEXT NOT NULL,
			storage_key  TEXT NOT NULL,
			mime_type    TEXT NOT NULL,
			size_bytes   BIGINT NOT NULL,
			width        INTEGER,
			height       INTEGER,
			alt_text     TEXT NOT NULL DEFAULT '',
			tags         TEXT[] NOT NULL DEFAULT '{}',
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cms_media_tenant ON cms_media(tenant_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS form_dates (
			id           BIGSERIAL PRIMARY KEY,
			tenant_id    TEXT NOT NULL REFERENCES tenants(id),
			label        TEXT NOT NULL,
			starts_on    DATE NOT NULL,
			is_enabled   BOOLEAN NOT NULL DEFAULT TRUE,
			capacity     INTEGER,
			sort_order   INTEGER NOT NULL DEFAULT 0,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_form_dates_tenant ON form_dates(tenant_id, sort_order)`,
		`CREATE TABLE IF NOT EXISTS pricing_plans (
			id             BIGSERIAL PRIMARY KEY,
			tenant_id      TEXT NOT NULL REFERENCES tenants(id),
			key            TEXT NOT NULL,
			name           TEXT NOT NULL,
			description    TEXT NOT NULL DEFAULT '',
			price_cop      INTEGER NOT NULL,
			features       JSONB NOT NULL DEFAULT '[]',
			image_media_id BIGINT REFERENCES cms_media(id),
			is_enabled     BOOLEAN NOT NULL DEFAULT TRUE,
			sort_order     INTEGER NOT NULL DEFAULT 0,
			UNIQUE (tenant_id, key)
		)`,
		`CREATE TABLE IF NOT EXISTS payment_methods (
			id            BIGSERIAL PRIMARY KEY,
			tenant_id     TEXT NOT NULL REFERENCES tenants(id),
			key           TEXT NOT NULL,
			label         TEXT NOT NULL,
			is_enabled    BOOLEAN NOT NULL DEFAULT TRUE,
			surcharge_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
			sort_order    INTEGER NOT NULL DEFAULT 0,
			config        JSONB NOT NULL DEFAULT '{}',
			UNIQUE (tenant_id, key)
		)`,
		`ALTER TABLE inscripciones ADD COLUMN IF NOT EXISTS tenant_id TEXT`,
		`UPDATE inscripciones SET tenant_id = 'st4ge' WHERE tenant_id IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_inscripciones_tenant_status ON inscripciones(tenant_id, status, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS admin_audit_log (
			id          BIGSERIAL PRIMARY KEY,
			tenant_id   TEXT NOT NULL,
			admin_id    BIGINT NOT NULL,
			action      TEXT NOT NULL,
			entity      TEXT NOT NULL,
			entity_id   TEXT,
			diff        JSONB,
			at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_tenant_at ON admin_audit_log(tenant_id, at DESC)`,
		`ALTER TABLE pricing_plans ADD COLUMN IF NOT EXISTS bold_link TEXT NOT NULL DEFAULT ''`,
	}
	for _, m := range migrations {
		if _, err := db.Pool.Exec(ctx, m); err != nil {
			return fmt.Errorf("migration: %w\nSQL: %s", err, m)
		}
	}
	db.logger.Info("postgres migrations complete")
	return nil
}
