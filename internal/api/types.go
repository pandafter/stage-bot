package api

// Modalidad describes a registration plan (price tier).
type Modalidad struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	PriceCOP int    `json:"price_cop"`
}

// PaymentMethod is one of the accepted payment methods.
type PaymentMethod struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ConfigResponse is returned by GET /api/config.
type ConfigResponse struct {
	Modalidades      []Modalidad     `json:"modalidades"`
	Metodos          []PaymentMethod `json:"metodos"`
	Fechas           []string        `json:"fechas"`
	CardSurchargePct int             `json:"card_surcharge_pct"`
	ReservaCOP       int             `json:"reserva_cop"`
	PrecioCompleto   int             `json:"precio_completo"`
	PrecioDescuento  int             `json:"precio_descuento"`
}

// CreateInscripcionRequest is the JSON body posted to POST /api/inscripciones.
type CreateInscripcionRequest struct {
	Email            string `json:"email"`
	NombrePiloto     string `json:"nombre_piloto"`
	Edad             int    `json:"edad"`
	TipoDocumento    string `json:"tipo_documento"`
	NumeroDocumento  string `json:"numero_documento"`
	Telefono         string `json:"telefono"`
	Ciudad           string `json:"ciudad"`
	EPS              string `json:"eps"`
	GrupoSanguineo   string `json:"grupo_sanguineo"`
	FamiliarNombre   string `json:"familiar_nombre"`
	FamiliarTelefono string `json:"familiar_telefono"`
	InstagramUser    string `json:"instagram_user"`
	Modalidad        string `json:"modalidad"`
	MetodoPago       string `json:"metodo_pago"`
	FechaCurso       string `json:"fecha_curso"`
}

// CreateInscripcionResponse is returned after a successful POST.
type CreateInscripcionResponse struct {
	ID                  string `json:"id"`
	Status              string `json:"status"`
	MontoCOP            int    `json:"monto_cop"`
	CheckoutURL         string `json:"checkout_url,omitempty"`
	RequiresComprobante bool   `json:"requires_comprobante"`
}

// InscripcionStatusResponse is returned by GET /api/inscripciones/:id.
type InscripcionStatusResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	MontoCOP     int    `json:"monto_cop"`
	FechaCurso   string `json:"fecha_curso"`
	MetodoPago   string `json:"metodo_pago"`
	Modalidad    string `json:"modalidad"`
	NombrePiloto string `json:"nombre_piloto"`
}

// Static catalog data (the source of truth — both backend and frontend
// derive their UI from this via /api/config).
const (
	CardSurchargePct = 5
	ReservaCOP       = 150000
	DefaultPlanID    = "preventa"
	PrecioCompleto   = 890000 // Full price (no discount)
	PrecioDescuento  = 730000 // Preventa discount price
)

var Modalidades = []Modalidad{
	{ID: "reserva", Label: "Solo reserva del cupo", PriceCOP: 150000},
	{ID: "completo", Label: "Pago completo (preventa)", PriceCOP: 730000},
}

var Metodos = []PaymentMethod{
	{ID: "tarjeta", Label: "Tarjeta de crédito / débito"},
	{ID: "nequi", Label: "Nequi"},
	{ID: "bancolombia", Label: "Botón Bancolombia"},
	{ID: "pse", Label: "PSE — otros bancos"},
	{ID: "transferencia", Label: "Transferencia manual + comprobante"},
}

var Fechas = []string{
	"MAYO 9 y 10",
	"MAYO 23 y 24",
}

func findModalidad(id string) *Modalidad {
	return findModalidadFrom(id, Modalidades)
}

func findModalidadFrom(id string, list []Modalidad) *Modalidad {
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}

func findMetodo(id string) *PaymentMethod {
	for i := range Metodos {
		if Metodos[i].ID == id {
			return &Metodos[i]
		}
	}
	return nil
}

// methodIsDigital reports whether the method is processed via Bold.
func methodIsDigital(id string) bool {
	switch id {
	case "tarjeta", "nequi", "bancolombia", "pse":
		return true
	default:
		return false
	}
}
