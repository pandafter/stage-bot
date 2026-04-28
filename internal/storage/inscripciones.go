package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type InscripcionRecord struct {
	ID               string
	Email            string
	MetodoPago       string
	FechaCurso       string
	Plan             string
	MontoCOP         int
	NombrePiloto     string
	Edad             int
	TipoDocumento    string
	NumeroDocumento  string
	Telefono         string
	Ciudad           string
	EPS              string
	GrupoSanguineo   string
	FamiliarNombre   string
	FamiliarTelefono string
	InstagramUser    string
	ComprobantePath  string
	Status           string
	CreatedAt        time.Time
}

type InscripcionesRepo struct {
	db *sql.DB
}

func NewInscripcionesRepo(db *DB) *InscripcionesRepo {
	return &InscripcionesRepo{db: db.Conn()}
}

func (r *InscripcionesRepo) Insert(ctx context.Context, rec *InscripcionRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO inscripciones (
			id, email, metodo_pago, fecha_curso, plan, monto_cop,
			nombre_piloto, edad, tipo_documento, numero_documento, telefono,
			ciudad, eps, grupo_sanguineo, familiar_nombre, familiar_telefono,
			instagram_user, comprobante_path, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		rec.ID, rec.Email, rec.MetodoPago, rec.FechaCurso, rec.Plan, rec.MontoCOP,
		rec.NombrePiloto, rec.Edad, rec.TipoDocumento, rec.NumeroDocumento, rec.Telefono,
		rec.Ciudad, rec.EPS, rec.GrupoSanguineo, rec.FamiliarNombre, rec.FamiliarTelefono,
		rec.InstagramUser, rec.ComprobantePath, rec.Status,
	)
	if err != nil {
		return fmt.Errorf("insert inscripcion: %w", err)
	}
	return nil
}

func (r *InscripcionesRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE inscripciones SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (r *InscripcionesRepo) GetByID(ctx context.Context, id string) (*InscripcionRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, email, metodo_pago, fecha_curso, plan, monto_cop,
			nombre_piloto, edad, tipo_documento, numero_documento, telefono,
			ciudad, eps, grupo_sanguineo, familiar_nombre, familiar_telefono,
			instagram_user, comprobante_path, status, created_at
		FROM inscripciones WHERE id = $1`, id)
	rec := &InscripcionRecord{}
	if err := row.Scan(
		&rec.ID, &rec.Email, &rec.MetodoPago, &rec.FechaCurso, &rec.Plan, &rec.MontoCOP,
		&rec.NombrePiloto, &rec.Edad, &rec.TipoDocumento, &rec.NumeroDocumento, &rec.Telefono,
		&rec.Ciudad, &rec.EPS, &rec.GrupoSanguineo, &rec.FamiliarNombre, &rec.FamiliarTelefono,
		&rec.InstagramUser, &rec.ComprobantePath, &rec.Status, &rec.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get inscripcion %s: %w", id, err)
	}
	return rec, nil
}
