package knowledge

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

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

// SheetData holds all parsed knowledge from the spreadsheet.
type SheetData struct {
	Empresa          string // general company info text
	Cursos           string // courses details
	Precios          string // pricing info
	FAQ              string // frequently asked questions
	Objeciones       string // objection handling
	EjemplosVentas   string // real sales response examples
	LinksVenta       string // payment/sale links
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
	}

	for tab, dest := range tabs {
		content, err := s.fetchTab(tab)
		if err != nil {
			s.logger.Warn("failed to fetch sheet tab", zap.String("tab", tab), zap.Error(err))
			continue
		}
		*dest = content
	}

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
		{"INFORMACIÓN DE LA EMPRESA", data.Empresa},
		{"CURSOS DISPONIBLES", data.Cursos},
		{"PRECIOS Y PLANES DE PAGO", data.Precios},
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

// fetchTab downloads a single sheet tab as CSV and converts it to readable text.
func (s *Store) fetchTab(tabName string) (string, error) {
	url := fmt.Sprintf(
		"https://docs.google.com/spreadsheets/d/%s/gviz/tq?tqx=out:csv&sheet=%s",
		s.sheetID, tabName,
	)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch tab %s: %w", tabName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch tab %s: status %d", tabName, resp.StatusCode)
	}

	reader := csv.NewReader(resp.Body)
	reader.LazyQuotes = true

	var lines []string
	headers, err := reader.Read()
	if err != nil {
		return "", fmt.Errorf("read headers for %s: %w", tabName, err)
	}

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		var parts []string
		for i, val := range row {
			val = strings.TrimSpace(val)
			if val == "" {
				continue
			}
			header := ""
			if i < len(headers) {
				header = strings.TrimSpace(headers[i])
			}
			if header != "" {
				parts = append(parts, fmt.Sprintf("%s: %s", header, val))
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
