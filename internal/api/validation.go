package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"

	"github.com/kart-academy/instagram-bot/internal/storage"
)

var emailRe = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)

func validEmail(e string) bool { return emailRe.MatchString(strings.ToLower(e)) }

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "INS-" + hex.EncodeToString(b)
}

// computeMonto returns the final amount (with surcharge for card).
func computeMonto(modalidad *Modalidad, metodoID string) int {
	if modalidad == nil {
		return 0
	}
	monto := modalidad.PriceCOP
	if metodoID == "tarjeta" {
		monto = monto + (monto*CardSurchargePct)/100
	}
	return monto
}

// validateCreate checks the incoming request and returns the matched Modalidad
// + PaymentMethod, or an error explaining what is wrong.
func validateCreate(req *CreateInscripcionRequest, validFechas []string, modalidades []Modalidad) (*Modalidad, *PaymentMethod, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.NombrePiloto = strings.TrimSpace(req.NombrePiloto)
	req.NumeroDocumento = strings.TrimSpace(req.NumeroDocumento)
	req.Telefono = strings.TrimSpace(req.Telefono)
	req.EPS = strings.TrimSpace(req.EPS)
	req.GrupoSanguineo = strings.TrimSpace(req.GrupoSanguineo)
	req.FamiliarNombre = strings.TrimSpace(req.FamiliarNombre)
	req.FamiliarTelefono = strings.TrimSpace(req.FamiliarTelefono)
	req.TipoDocumento = strings.TrimSpace(req.TipoDocumento)
	req.FechaCurso = strings.TrimSpace(req.FechaCurso)
	req.Modalidad = strings.TrimSpace(req.Modalidad)
	req.MetodoPago = strings.TrimSpace(req.MetodoPago)

	if req.Email == "" || !validEmail(req.Email) {
		return nil, nil, errors.New("email inválido")
	}
	if req.NombrePiloto == "" {
		return nil, nil, errors.New("nombre del piloto es obligatorio")
	}
	if req.NumeroDocumento == "" {
		return nil, nil, errors.New("número de documento es obligatorio")
	}
	if req.Telefono == "" {
		return nil, nil, errors.New("teléfono es obligatorio")
	}
	if req.EPS == "" || req.GrupoSanguineo == "" {
		return nil, nil, errors.New("EPS y grupo sanguíneo son obligatorios")
	}
	if req.FamiliarNombre == "" || req.FamiliarTelefono == "" {
		return nil, nil, errors.New("contacto familiar de emergencia es obligatorio")
	}
	if req.Edad < 8 || req.Edad > 90 {
		return nil, nil, errors.New("edad fuera de rango (8-90)")
	}
	mod := findModalidadFrom(req.Modalidad, modalidades)
	if mod == nil {
		return nil, nil, errors.New("modalidad inválida")
	}
	met := findMetodo(req.MetodoPago)
	if met == nil {
		return nil, nil, errors.New("método de pago inválido")
	}
	fechaOK := false
	for _, f := range validFechas {
		if strings.EqualFold(f, req.FechaCurso) {
			req.FechaCurso = f // normalize to stored label
			fechaOK = true
			break
		}
	}
	if !fechaOK {
		return nil, nil, errors.New("fecha de curso inválida")
	}
	return mod, met, nil
}

// toRecord converts a validated request into the storage layer struct.
func (r *CreateInscripcionRequest) toRecord() *storage.InscripcionRecord {
	return &storage.InscripcionRecord{
		Email:            r.Email,
		MetodoPago:       r.MetodoPago,
		FechaCurso:       r.FechaCurso,
		Plan:             r.Modalidad,
		NombrePiloto:     r.NombrePiloto,
		Edad:             r.Edad,
		TipoDocumento:    r.TipoDocumento,
		NumeroDocumento:  r.NumeroDocumento,
		Telefono:         r.Telefono,
		Ciudad:           strings.TrimSpace(r.Ciudad),
		EPS:              r.EPS,
		GrupoSanguineo:   r.GrupoSanguineo,
		FamiliarNombre:   r.FamiliarNombre,
		FamiliarTelefono: r.FamiliarTelefono,
		InstagramUser:    strings.TrimSpace(r.InstagramUser),
	}
}
