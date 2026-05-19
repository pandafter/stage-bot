package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type InscripcionStats struct {
	Total          int          `json:"total"`
	Pendientes     int          `json:"pendientes"`
	Pagadas        int          `json:"pagadas"`
	Rechazadas     int          `json:"rechazadas"`
	MontoRecaudado int64        `json:"monto_recaudado"`
	PorFecha       []FechaCount `json:"por_fecha"`
}

type FechaCount struct {
	Fecha string `json:"fecha"`
	Count int    `json:"count"`
}

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
	db *DB
}

func NewInscripcionesRepo(db *DB) *InscripcionesRepo {
	return &InscripcionesRepo{db: db}
}

func (r *InscripcionesRepo) Insert(ctx context.Context, rec *InscripcionRecord) error {
	_, err := r.db.Pool.Exec(ctx, `
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
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE inscripciones SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (r *InscripcionesRepo) UpdateComprobante(ctx context.Context, id, path, status string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE inscripciones SET comprobante_path = $1, status = $2 WHERE id = $3`,
		path, status, id)
	return err
}

func (r *InscripcionesRepo) GetByID(ctx context.Context, id string) (*InscripcionRecord, error) {
	rec := &InscripcionRecord{}
	row := r.db.Pool.QueryRow(ctx, `
		SELECT id, email, metodo_pago, fecha_curso, plan, monto_cop,
			nombre_piloto, edad, tipo_documento, numero_documento, telefono,
			ciudad, eps, grupo_sanguineo, familiar_nombre, familiar_telefono,
			instagram_user, comprobante_path, status, created_at
		FROM inscripciones WHERE id = $1`, id)
	err := row.Scan(
		&rec.ID, &rec.Email, &rec.MetodoPago, &rec.FechaCurso, &rec.Plan, &rec.MontoCOP,
		&rec.NombrePiloto, &rec.Edad, &rec.TipoDocumento, &rec.NumeroDocumento, &rec.Telefono,
		&rec.Ciudad, &rec.EPS, &rec.GrupoSanguineo, &rec.FamiliarNombre, &rec.FamiliarTelefono,
		&rec.InstagramUser, &rec.ComprobantePath, &rec.Status, &rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get inscripcion %s: %w", id, err)
	}
	return rec, nil
}

func (r *InscripcionesRepo) Stats(ctx context.Context) (InscripcionStats, error) {
	var s InscripcionStats
	err := r.db.Pool.QueryRow(ctx, `
		SELECT
		  COUNT(*) AS total,
		  SUM(CASE WHEN status IN ('pendiente','comprobante recibido, en validación','comprobante_pendiente') THEN 1 ELSE 0 END) AS pendientes,
		  SUM(CASE WHEN status = 'pagado' THEN 1 ELSE 0 END) AS pagadas,
		  SUM(CASE WHEN status IN ('pago rechazado','rechazado','pago anulado','cancelado') THEN 1 ELSE 0 END) AS rechazadas,
		  COALESCE(SUM(CASE WHEN status = 'pagado' THEN monto_cop ELSE 0 END), 0) AS monto_recaudado
		FROM inscripciones`,
	).Scan(&s.Total, &s.Pendientes, &s.Pagadas, &s.Rechazadas, &s.MontoRecaudado)
	if err != nil {
		return s, fmt.Errorf("inscripciones.Stats: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT fecha_curso, COUNT(*) AS cnt
		FROM inscripciones
		WHERE fecha_curso != ''
		GROUP BY fecha_curso
		ORDER BY cnt DESC`)
	if err != nil {
		return s, fmt.Errorf("inscripciones.Stats por_fecha: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var fc FechaCount
		if err := rows.Scan(&fc.Fecha, &fc.Count); err != nil {
			return s, err
		}
		s.PorFecha = append(s.PorFecha, fc)
	}
	if s.PorFecha == nil {
		s.PorFecha = []FechaCount{}
	}
	return s, rows.Err()
}

func (r *InscripcionesRepo) List(ctx context.Context, tenantID, status, dateFrom, dateTo, plan, search string, page, limit int) ([]InscripcionRecord, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	where := []string{"1=1"}
	args := []any{}
	n := 1

	if status != "" {
		where = append(where, fmt.Sprintf("status=$%d", n))
		args = append(args, status)
		n++
	}
	if plan != "" {
		where = append(where, fmt.Sprintf("plan=$%d", n))
		args = append(args, plan)
		n++
	}
	if dateFrom != "" {
		where = append(where, fmt.Sprintf("created_at >= $%d::timestamptz", n))
		args = append(args, dateFrom)
		n++
	}
	if dateTo != "" {
		where = append(where, fmt.Sprintf("created_at <= $%d::timestamptz", n))
		args = append(args, dateTo)
		n++
	}
	if search != "" {
		where = append(where, fmt.Sprintf("(email ILIKE $%d OR nombre_piloto ILIKE $%d OR numero_documento ILIKE $%d)", n, n, n))
		args = append(args, "%"+search+"%")
		n++
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQ := "SELECT COUNT(*) FROM inscripciones WHERE " + whereClause
	if err := r.db.Pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("inscripciones.List count: %w", err)
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(
		`SELECT id, email, metodo_pago, fecha_curso, plan, monto_cop,
		        nombre_piloto, edad, tipo_documento, numero_documento, telefono,
		        ciudad, eps, grupo_sanguineo, familiar_nombre, familiar_telefono,
		        instagram_user, comprobante_path, status, created_at
		 FROM inscripciones WHERE %s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`,
		whereClause, n, n+1,
	)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("inscripciones.List query: %w", err)
	}
	defer rows.Close()

	var out []InscripcionRecord
	for rows.Next() {
		var ins InscripcionRecord
		if err := rows.Scan(
			&ins.ID, &ins.Email, &ins.MetodoPago, &ins.FechaCurso, &ins.Plan, &ins.MontoCOP,
			&ins.NombrePiloto, &ins.Edad, &ins.TipoDocumento, &ins.NumeroDocumento, &ins.Telefono,
			&ins.Ciudad, &ins.EPS, &ins.GrupoSanguineo, &ins.FamiliarNombre, &ins.FamiliarTelefono,
			&ins.InstagramUser, &ins.ComprobantePath, &ins.Status, &ins.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, ins)
	}
	return out, total, rows.Err()
}
