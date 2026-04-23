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
	Empresa        string // general company info text
	Cursos         string // courses details
	Precios        string // pricing info
	FAQ            string // frequently asked questions
	Objeciones     string // objection handling
	EjemplosVentas string // real sales response examples
	LinksVenta     string // payment/sale links
	Fechas         string // upcoming course dates and availability
	Scripted       ScriptedMessages
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

	return b.String()
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
	director := strings.TrimSpace(data.Scripted.Director)
	if director == "" {
		return ""
	}
	if strings.Contains(strings.ToLower(director), "whatsapp") {
		return director
	}
	return fmt.Sprintf("Perfecto. Continúa la venta directamente por WhatsApp del director: %s", director)
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
		Message2:     "¿Quieres que te pase el número del director para continuar? Responde solo Sí o No.",
		FollowUp6H:   "Hola, te escribimos para hacer seguimiento. Si quieres continuar responde Sí y te pasamos el número del director.",
		FollowUp24H:  "Seguimos atentos. Si quieres continuar con la venta, responde Sí y te pasamos el número del director.",
		FollowUp120H: "Último mensaje de seguimiento: si quieres continuar, responde Sí y te pasamos el número del director.",
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
