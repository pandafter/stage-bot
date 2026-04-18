package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
)

// SalesArticle represents a single article from the sales methodology dataset.
type SalesArticle struct {
	ID        string   `json:"id"`
	Title     string   `json:"titulo"`
	Category  string   `json:"categoria"`
	Tags      []string `json:"tags"`
	Content   string   `json:"contenido"`
}

// SalesDataset loads and serves sales methodology articles, selecting
// the most relevant ones based on the current sales strategy and intent.
type SalesDataset struct {
	articles []SalesArticle
	logger   *zap.Logger
}

// LoadSalesDataset reads the JSON file and returns a ready-to-use dataset.
func LoadSalesDataset(path string, logger *zap.Logger) (*SalesDataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sales dataset: %w", err)
	}

	var articles []SalesArticle
	if err := json.Unmarshal(data, &articles); err != nil {
		return nil, fmt.Errorf("parse sales dataset: %w", err)
	}

	logger.Info("sales dataset loaded", zap.Int("articles", len(articles)))
	return &SalesDataset{articles: articles, logger: logger}, nil
}

// strategyTags maps each sales strategy to the tags/keywords most relevant for article selection.
var strategyTags = map[string][]string{
	"WELCOME":          {"primer mensaje", "apertura", "DM", "primer contacto"},
	"INFORM":           {"consultiva", "proceso", "contenido", "perfil"},
	"PERSUADE":         {"prueba social", "testimonios", "autoridad", "embudo", "Reels"},
	"GUIDE":            {"script", "consultiva", "cierre", "proceso", "decisión"},
	"CLOSE":            {"cierre", "conversión", "decisión", "propuesta"},
	"HANDLE_OBJECTION": {"objeciones", "precio", "negociación", "clientes difíciles"},
	"CONFIRM_SALE":     {"cierre", "propuesta", "conversión"},
	"REDIRECT":         {"contenido", "embudo", "Reels"},
	"UPSELL":           {"retención", "referidos", "voz a voz", "embajadores"},
}

// intentTags adds intent-specific relevance on top of strategy.
var intentTags = map[string][]string{
	"PRICE_INQUIRY":    {"precio", "psicología", "ancla de precio", "estrategia"},
	"OBJECTION_PRICE":  {"objeciones", "precio", "negociación", "valor"},
	"OBJECTION_TIME":   {"objeciones", "follow-up", "seguimiento"},
	"OBJECTION_DOUBT":  {"objeciones", "clientes difíciles", "credibilidad"},
	"OBJECTION_OTHER":  {"objeciones", "clientes difíciles"},
	"BUY_SIGNAL":       {"cierre", "conversión", "decisión"},
	"PAYMENT_CONFIRM":  {"cierre", "propuesta", "conversión"},
	"SCHEDULE_INQUIRY": {"consultiva", "proceso"},
	"COURSE_INQUIRY":   {"consultiva", "contenido", "proceso"},
}

// ForStrategy returns the most relevant articles formatted as a prompt fragment.
// It scores articles by tag overlap and returns the top matches (max 3).
func (sd *SalesDataset) ForStrategy(strategy, intent string) string {
	if sd == nil || len(sd.articles) == 0 {
		return ""
	}

	// Collect relevant tags from strategy + intent
	var relevantTags []string
	if tags, ok := strategyTags[strategy]; ok {
		relevantTags = append(relevantTags, tags...)
	}
	if tags, ok := intentTags[intent]; ok {
		relevantTags = append(relevantTags, tags...)
	}

	if len(relevantTags) == 0 {
		// Fallback: general sales methodology
		relevantTags = []string{"DM", "consultiva", "Colombia", "ventas"}
	}

	// Score each article by tag overlap
	type scored struct {
		article SalesArticle
		score   int
	}
	var results []scored
	for _, art := range sd.articles {
		s := 0
		artLower := strings.ToLower(strings.Join(art.Tags, " ") + " " + art.Category + " " + art.Title)
		for _, tag := range relevantTags {
			if strings.Contains(artLower, strings.ToLower(tag)) {
				s++
			}
		}
		if s > 0 {
			results = append(results, scored{article: art, score: s})
		}
	}

	// Sort by score descending (simple selection for top 3)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	maxArticles := 3
	if len(results) < maxArticles {
		maxArticles = len(results)
	}
	if maxArticles == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("METODOLOGÍA DE VENTAS (aplica estos principios en tu respuesta):\n")
	for i := 0; i < maxArticles; i++ {
		art := results[i].article
		// Truncate content to keep prompt reasonable — first ~600 chars capture the key ideas
		content := art.Content
		if len(content) > 600 {
			// Cut at last space before limit
			cut := 600
			if idx := strings.LastIndex(content[:cut], " "); idx > 400 {
				cut = idx
			}
			content = content[:cut] + "…"
		}
		fmt.Fprintf(&b, "\n--- %s ---\n%s\n", art.Title, content)
	}
	return b.String()
}
