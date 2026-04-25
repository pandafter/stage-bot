package knowledge

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"go.uber.org/zap"
)

// Store fetches and caches knowledge from a Google Sheet.
type Store struct {
	sheetID    string
	httpClient *http.Client
	logger     *zap.Logger

	mu   sync.RWMutex
	data *SheetData
	last time.Time
}

const cacheTTL = 5 * time.Minute

type ScriptedMessages struct {
	Message1     string
	Message2     string
	Director     string
	FollowUp6H   string
	FollowUp24H  string
	FollowUp120H string
}

// SheetData holds all parsed knowledge from the spreadsheet.
type SheetData struct {
	Empresa             string   // general company info text
	EmpresaWhatsApp     string   // sales WhatsApp number (digits only, no +)
	EmpresaCharlaCubre  string   // what the in-person "charla" course session covers
	Cursos              string   // courses details
	Precios             string   // pricing info
	FAQ                 string   // frequently asked questions
	Objeciones          string   // objection handling
	EjemplosVentas      string   // real sales response examples
	GuideMessages       []string // lettered guide messages from ejemplos_ventas
	LinksVenta          string   // payment/sale links
	Fechas              string   // upcoming course dates and availability
	Scripted            ScriptedMessages
}

func NewStore(sheetID string, logger *zap.Logger) *Store {
	return &Store{
		sheetID: sheetID,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		logger: logger,
	}
}

// Enabled returns true if a sheet ID is configured.
func (s *Store) Enabled() bool {
	return s.sheetID != ""
}

// Get returns the cached knowledge, refreshing if stale.
func (s *Store) Get() *SheetData {
	s.mu.RLock()
	if s.data != nil && time.Since(s.last) < cacheTTL {
		defer s.mu.RUnlock()
		return s.data
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if s.data != nil && time.Since(s.last) < cacheTTL {
		return s.data
	}

	data := &SheetData{}
	tabs := map[string]*string{
		"empresa":         &data.Empresa,
		"cursos":          &data.Cursos,
		"precios":         &data.Precios,
		"faq":             &data.FAQ,
		"objeciones":      &data.Objeciones,
		"ejemplos_ventas": &data.EjemplosVentas,
		"links_venta":     &data.LinksVenta,
		"fechas":          &data.Fechas,
	}

	for tab, dest := range tabs {
		content, err := s.fetchTab(tab)
		if err != nil {
			s.logger.Warn("failed to fetch sheet tab", zap.String("tab", tab), zap.Error(err))
			continue
		}
		*dest = content
	}

	if rows, err := s.fetchTabRows("empresa"); err == nil {
		data.EmpresaWhatsApp = extractEmpresaWhatsApp(rows)
		data.EmpresaCharlaCubre = extractFirstByKeys(rows,
			"charlaincluida", "charla", "charlacubre", "incluyecharla")
	}

	if rows, err := s.fetchTabRows("ejemplos_ventas"); err == nil {
		data.GuideMessages = extractColumnValues(rows,
			"mensajespredeterminados", "mensajepredeterminado", "guia", "opciones")
	}

	data.Scripted = s.fetchScriptedMessages()

	s.data = data
	s.last = time.Now()
	s.logger.Info("knowledge refreshed from Google Sheet")

	return s.data
}

// FormatContext returns all knowledge as a single string for the system prompt.
func (s *Store) FormatContext() string {
	data := s.Get()
	if data == nil {
		return ""
	}

	var b strings.Builder

	sections := []struct {
		title   string
		content string
	}{
		{"INFORMACIÓN DE LA EMPRESA Y NOTAS IA", data.Empresa},
		{"CURSOS DISPONIBLES", data.Cursos},
		{"PRECIOS Y PLANES DE PAGO", data.Precios},
		{"FECHAS Y DISPONIBILIDAD DE CUPOS (fuente de verdad actualizada)", data.Fechas},
		{"PREGUNTAS FRECUENTES", data.FAQ},
		{"MANEJO DE OBJECIONES", data.Objeciones},
		{"EJEMPLOS DE RESPUESTAS REALES DEL EQUIPO DE VENTAS", data.EjemplosVentas},
		{"LINKS DE VENTA/PAGO", data.LinksVenta},
	}

	for _, sec := range sections {
		if sec.content == "" {
			continue
		}
		fmt.Fprintf(&b, "\n=== %s ===\n%s\n", sec.title, sec.content)
	}

	if data.EmpresaCharlaCubre != "" {
		fmt.Fprintf(&b, "\n=== TEMAS QUE CUBRE LA CHARLA DEL CURSO ===\n%s\n", data.EmpresaCharlaCubre)
	}
	if data.EmpresaWhatsApp != "" {
		fmt.Fprintf(&b, "\n=== WHATSAPP DE VENTAS ===\n%s\n", data.EmpresaWhatsApp)
	}
	if len(data.GuideMessages) > 0 {
		b.WriteString("\n=== OPCIONES PREDETERMINADAS PARA GUIAR LA VENTA ===\n")
		for _, g := range data.GuideMessages {
			fmt.Fprintf(&b, "- %s\n", g)
		}
	}

	return b.String()
}

// EmpresaWhatsApp returns the cached sales WhatsApp number (digits only).
func (s *Store) EmpresaWhatsApp() string {
	if d := s.Get(); d != nil {
		return d.EmpresaWhatsApp
	}
	return ""
}

// CharlaIncluida returns the cached description of what the course charla covers.
func (s *Store) CharlaIncluida() string {
	if d := s.Get(); d != nil {
		return d.EmpresaCharlaCubre
	}
	return ""
}

// GuideMessages returns the cached lettered guide options.
func (s *Store) GuideMessages() []string {
	if d := s.Get(); d != nil {
		return d.GuideMessages
	}
	return nil
}

func (s *Store) PrimaryMessages() []string {
	data := s.Get()
	if data == nil {
		return nil
	}
	return compactTextList(data.Scripted.Message1, data.Scripted.Message2)
}

func (s *Store) DirectorContactMessage() string {
	data := s.Get()
	if data == nil {
		return ""
	}

	linksVenta := strings.TrimSpace(data.LinksVenta)
	director := strings.TrimSpace(data.Scripted.Director)

	if linksVenta == "" && director == "" {
		return ""
	}

	var parts []string
	if linksVenta != "" {
		parts = append(parts, "Formulario de inscripción y medios de pago:\n"+linksVenta)
	}
	if director != "" {
		if strings.Contains(strings.ToLower(director), "whatsapp") {
			parts = append(parts, director)
		} else {
			parts = append(parts, fmt.Sprintf("Número del director para acompañarte en la inscripción: %s", director))
		}
	}
	return strings.Join(parts, "\n\n")
}

func (s *Store) FollowUpMessage(attempt int) string {
	data := s.Get()
	if data == nil {
		return ""
	}
	switch attempt {
	case 1:
		return strings.TrimSpace(data.Scripted.FollowUp6H)
	case 2:
		return strings.TrimSpace(data.Scripted.FollowUp24H)
	default:
		return strings.TrimSpace(data.Scripted.FollowUp120H)
	}
}

func (s *Store) fetchScriptedMessages() ScriptedMessages {
	out := ScriptedMessages{
		Message1:     "Hola, te comparto la información principal de Scuderia St4ge.",
		Message2:     "¿Quieres el link de inscripción con las fechas y los medios de pago para reservar tu cupo con el descuento que tienes aprobado por tiempo limitado?",
		FollowUp6H:   "Hola, te escribimos para hacer seguimiento. ¿Quieres el link de inscripción con las fechas y los medios de pago para reservar tu cupo con el descuento que tienes aprobado por tiempo limitado?",
		FollowUp24H:  "Seguimos atentos. Si te interesa, te enviamos el link de inscripción con fechas y medios de pago para reservar tu cupo con el descuento aprobado.",
		FollowUp120H: "Último mensaje de seguimiento: si quieres continuar, te compartimos el link de inscripción con fechas y medios de pago para reservar tu cupo con descuento.",
	}

	tabCandidates := []string{"mensajes_predeterminado", "mensajes_predeterminados", "mensajes"}
	var rows []map[string]string
	for _, tab := range tabCandidates {
		foundRows, err := s.fetchTabRows(tab)
		if err != nil {
			s.logger.Debug("scripted tab not available", zap.String("tab", tab), zap.Error(err))
			continue
		}
		if len(foundRows) == 0 {
			continue
		}
		rows = foundRows
		break
	}
	if len(rows) == 0 {
		return out
	}

	row := rows[0]
	if val := firstNonEmptyByKeys(row,
		"mensaje1", "mensajeprincipal1", "mensajeprincipal", "mensaje", "message1", "primarymessage1",
	); val != "" {
		out.Message1 = val
	}
	if val := firstNonEmptyByKeys(row,
		"mensaje2", "mensajeprincipal2", "pregunta", "preguntasino", "message2", "primarymessage2",
	); val != "" {
		out.Message2 = val
	}
	if val := firstNonEmptyByKeys(row,
		"directormensaje", "numerodirector", "whatsappdirector", "telefonodirector", "contactodirector", "director",
	); val != "" {
		out.Director = val
	}

	genericFollowup := firstNonEmptyByKeys(row, "followup", "followups", "seguimiento")
	if val := firstNonEmptyByKeys(row, "followup6h", "followup1", "seguimiento6h"); val != "" {
		out.FollowUp6H = val
	} else if genericFollowup != "" {
		out.FollowUp6H = genericFollowup
	}
	if val := firstNonEmptyByKeys(row, "followup24h", "followup2", "seguimiento24h"); val != "" {
		out.FollowUp24H = val
	} else if genericFollowup != "" {
		out.FollowUp24H = genericFollowup
	}
	if val := firstNonEmptyByKeys(row, "followup120h", "followup3", "seguimiento120h"); val != "" {
		out.FollowUp120H = val
	} else if genericFollowup != "" {
		out.FollowUp120H = genericFollowup
	}

	return out
}

// fetchTab downloads a single sheet tab as CSV and converts it to readable text.
func (s *Store) fetchTab(tabName string) (string, error) {
	rows, err := s.fetchTabRows(tabName)
	if err != nil {
		return "", err
	}

	var lines []string
	for _, row := range rows {
		var parts []string
		for key, val := range row {
			val = strings.TrimSpace(val)
			if val == "" {
				continue
			}
			if key != "" {
				parts = append(parts, fmt.Sprintf("%s: %s", key, val))
			} else {
				parts = append(parts, val)
			}
		}
		if len(parts) > 0 {
			lines = append(lines, strings.Join(parts, " | "))
		}
	}

	return strings.Join(lines, "\n"), nil
}

func (s *Store) fetchTabRows(tabName string) ([]map[string]string, error) {
	url := fmt.Sprintf(
		"https://docs.google.com/spreadsheets/d/%s/gviz/tq?tqx=out:csv&sheet=%s",
		s.sheetID, tabName,
	)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch tab %s: %w", tabName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch tab %s: status %d", tabName, resp.StatusCode)
	}

	reader := csv.NewReader(resp.Body)
	reader.LazyQuotes = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read headers for %s: %w", tabName, err)
	}
	for i := range headers {
		headers[i] = normalizeHeaderKey(headers[i])
	}

	var rows []map[string]string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		entry := make(map[string]string, len(row))
		nonEmpty := false
		for i, val := range row {
			val = strings.TrimSpace(val)
			if val != "" {
				nonEmpty = true
			}
			key := ""
			if i < len(headers) {
				key = headers[i]
			}
			if key == "" {
				key = fmt.Sprintf("col%d", i)
			}
			entry[key] = val
		}
		if nonEmpty {
			rows = append(rows, entry)
		}
	}
	return rows, nil
}

func normalizeHeaderKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == 'á' || r == 'à' || r == 'ä':
			b.WriteRune('a')
		case r == 'é' || r == 'è' || r == 'ë':
			b.WriteRune('e')
		case r == 'í' || r == 'ì' || r == 'ï':
			b.WriteRune('i')
		case r == 'ó' || r == 'ò' || r == 'ö':
			b.WriteRune('o')
		case r == 'ú' || r == 'ù' || r == 'ü':
			b.WriteRune('u')
		case r == 'ñ':
			b.WriteRune('n')
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == ' ' || r == '_' || r == '-' || r == '.':
			// skip separator
		}
	}
	return b.String()
}

func firstNonEmptyByKeys(row map[string]string, keys ...string) string {
	for _, key := range keys {
		normalized := normalizeHeaderKey(key)
		if val, ok := row[normalized]; ok {
			val = strings.TrimSpace(val)
			if val != "" {
				return val
			}
		}
	}
	return ""
}

// extractEmpresaWhatsApp finds a sales WhatsApp number in the empresa rows.
// It tries explicit columns first, then scans every value for a Colombian-format
// phone number as fallback.
func extractEmpresaWhatsApp(rows []map[string]string) string {
	if val := extractFirstByKeys(rows,
		"whatsappventas", "whatsapp", "whatsappdirector", "telefono",
		"telefonoventas", "celular", "numero", "numerowhatsapp",
	); val != "" {
		return digitsOnly(val)
	}
	for _, row := range rows {
		for _, v := range row {
			if d := digitsOnly(v); len(d) >= 10 && len(d) <= 13 {
				return d
			}
		}
	}
	return ""
}

func extractFirstByKeys(rows []map[string]string, keys ...string) string {
	for _, row := range rows {
		if val := firstNonEmptyByKeys(row, keys...); val != "" {
			return val
		}
	}
	return ""
}

func extractColumnValues(rows []map[string]string, keys ...string) []string {
	var out []string
	for _, row := range rows {
		if val := firstNonEmptyByKeys(row, keys...); val != "" {
			out = append(out, val)
		}
	}
	return out
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func compactTextList(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}
