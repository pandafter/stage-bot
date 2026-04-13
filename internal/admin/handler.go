package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/storage"
)

type Handler struct {
	cfg      *config.Config
	leads    *storage.LeadsRepo
	conv     *storage.ConversationRepo
	logger   *zap.Logger
	httpCli  *http.Client
}

func NewHandler(cfg *config.Config, leads *storage.LeadsRepo, conv *storage.ConversationRepo, logger *zap.Logger) *Handler {
	return &Handler{
		cfg:     cfg,
		leads:   leads,
		conv:    conv,
		logger:  logger,
		httpCli: &http.Client{Timeout: 10 * time.Second},
	}
}

// Auth enforces the ADMIN_TOKEN check via ?token= query param.
func (h *Handler) Auth(c *fiber.Ctx) error {
	if h.cfg.AdminToken == "" {
		return c.Status(fiber.StatusServiceUnavailable).SendString("ADMIN_TOKEN not configured")
	}
	token := c.Query("token")
	if token == "" {
		token = c.Get("X-Admin-Token")
	}
	if token != h.cfg.AdminToken {
		return c.Status(fiber.StatusForbidden).SendString("forbidden")
	}
	return c.Next()
}

// Dashboard renders the main diagnostic page.
func (h *Handler) Dashboard(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	token := h.cfg.AdminToken

	var b strings.Builder
	b.WriteString(pageHeader("Panel — Scuderia St4ge"))

	// Config health
	b.WriteString(`<h2>Estado de configuración</h2><table class="kv">`)
	rows := []struct {
		name string
		set  bool
		val  string
		warn string
	}{
		{"ENV", true, h.cfg.Env, ""},
		{"PAGE_ACCESS_TOKEN", h.cfg.PageAccessToken != "", maskToken(h.cfg.PageAccessToken), ""},
		{"APP_SECRET", h.cfg.AppSecret != "", maskToken(h.cfg.AppSecret), ""},
		{"WEBHOOK_VERIFY_TOKEN", h.cfg.WebhookVerifyToken != "", maskToken(h.cfg.WebhookVerifyToken), ""},
		{"INSTAGRAM_ACCOUNT_ID", h.cfg.InstagramAccountID != "", h.cfg.InstagramAccountID, ""},
		{"ANTHROPIC_API_KEY", h.cfg.AnthropicAPIKey != "", maskToken(h.cfg.AnthropicAPIKey), ""},
		{"ELEVENLABS_API_KEY", h.cfg.ElevenLabsAPIKey != "", maskToken(h.cfg.ElevenLabsAPIKey), ""},
		{"ELEVENLABS_VOICE_ID", h.cfg.ElevenLabsVoiceID != "", h.cfg.ElevenLabsVoiceID, ""},
		{"GOOGLE_SHEET_ID", h.cfg.GoogleSheetID != "", h.cfg.GoogleSheetID, ""},
		{"PUBLIC_URL", h.cfg.PublicURL != "", h.cfg.PublicURL, ""},
		{"DATABASE_URL", h.cfg.DatabaseURL != "", "set", ""},
		{"TEST_SENDER_ID", h.cfg.TestSenderID != "", h.cfg.TestSenderID, testSenderWarning(h.cfg.TestSenderID)},
	}
	for _, r := range rows {
		status := `<span class="ok">OK</span>`
		if !r.set {
			status = `<span class="bad">FALTA</span>`
		}
		warnHTML := ""
		if r.warn != "" {
			warnHTML = `<div class="warn">` + html.EscapeString(r.warn) + `</div>`
		}
		fmt.Fprintf(&b, `<tr><td>%s</td><td>%s</td><td class="mono">%s</td><td>%s</td></tr>`,
			r.name, status, html.EscapeString(r.val), warnHTML)
	}
	b.WriteString(`</table>`)

	// API ping buttons
	b.WriteString(`<h2>Probar APIs</h2>`)
	fmt.Fprintf(&b, `<p><a class="btn" href="/admin/ping/claude?token=%s">Probar Claude</a>
		<a class="btn" href="/admin/ping/elevenlabs?token=%s">Probar ElevenLabs</a>
		<a class="btn" href="/admin/ping/instagram?token=%s">Probar Instagram Graph</a></p>`,
		token, token, token)

	// Recent leads
	b.WriteString(`<h2>Últimos leads (24h)</h2>`)
	leads, lastMsg, err := h.recentLeads(ctx, 50, 24*time.Hour)
	if err != nil {
		fmt.Fprintf(&b, `<p class="bad">Error cargando leads: %s</p>`, html.EscapeString(err.Error()))
	} else if len(leads) == 0 {
		b.WriteString(`<p><em>No hay leads con actividad en las últimas 24h.</em></p>`)
	} else {
		b.WriteString(`<table class="leads"><tr><th>Lead ID</th><th>Score</th><th>Estado</th><th>Msgs</th><th>Último mensaje</th><th>Cuándo</th><th></th></tr>`)
		for _, l := range leads {
			last := lastMsg[l.ID]
			preview := "—"
			who := ""
			when := ""
			if last != nil {
				preview = truncate(last.Content, 80)
				who = string(last.Role)
				when = relTime(last.CreatedAt)
			}
			rowClass := ""
			if last != nil && last.Role == storage.RoleUser {
				rowClass = ` class="unanswered"`
			}
			fmt.Fprintf(&b, `<tr%s><td class="mono">%s</td><td>%d</td><td>%s</td><td>%d</td><td>[%s] %s</td><td>%s</td><td><a href="/admin/lead/%s?token=%s">ver</a></td></tr>`,
				rowClass,
				html.EscapeString(l.ID),
				l.LeadScore,
				html.EscapeString(string(l.State)),
				l.TotalMessages,
				html.EscapeString(who),
				html.EscapeString(preview),
				when,
				html.EscapeString(l.ID),
				token,
			)
		}
		b.WriteString(`</table>`)
		b.WriteString(`<p><small>Filas resaltadas: último mensaje es del usuario (posiblemente sin respuesta).</small></p>`)
	}

	b.WriteString(pageFooter())
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(b.String())
}

// LeadDetail renders the full conversation for a single lead.
func (h *Handler) LeadDetail(c *fiber.Ctx) error {
	leadID := c.Params("id")
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	lead, err := h.leads.Get(ctx, leadID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("lead no encontrado: " + err.Error())
	}

	msgs, err := h.conv.FullForLead(ctx, leadID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("error cargando conversación: " + err.Error())
	}

	var b strings.Builder
	b.WriteString(pageHeader("Lead " + leadID))
	fmt.Fprintf(&b, `<p><a href="/admin?token=%s">&larr; volver</a></p>`, h.cfg.AdminToken)
	fmt.Fprintf(&b, `<h2>Lead <span class="mono">%s</span></h2>`, html.EscapeString(leadID))
	fmt.Fprintf(&b, `<table class="kv">
		<tr><td>Username</td><td>%s</td></tr>
		<tr><td>Score</td><td>%d</td></tr>
		<tr><td>Estado</td><td>%s</td></tr>
		<tr><td>Outcome</td><td>%s</td></tr>
		<tr><td>Mensajes totales</td><td>%d</td></tr>
		<tr><td>Precio preguntado</td><td>%v</td></tr>
		<tr><td>Horario preguntado</td><td>%v</td></tr>
		<tr><td>Señal de compra</td><td>%v</td></tr>
		<tr><td>Objeciones</td><td>%d</td></tr>
	</table>`,
		html.EscapeString(lead.Username), lead.LeadScore, string(lead.State), string(lead.Outcome),
		lead.TotalMessages, lead.PriceAsked, lead.ScheduleAsked, lead.BuySignal, lead.Objections)

	b.WriteString(`<h2>Conversación completa</h2>`)
	if len(msgs) == 0 {
		b.WriteString(`<p><em>Sin mensajes.</em></p>`)
	} else {
		b.WriteString(`<div class="chat">`)
		for _, m := range msgs {
			side := "user"
			if m.Role == storage.RoleAssistant {
				side = "bot"
			}
			meta := m.CreatedAt.Format("2006-01-02 15:04:05")
			if m.Intent != "" {
				meta += " · intent=" + m.Intent
			}
			if m.Strategy != "" {
				meta += " · " + m.Strategy
			}
			fmt.Fprintf(&b, `<div class="bubble %s"><div class="body">%s</div><div class="meta">%s</div></div>`,
				side, html.EscapeString(m.Content), html.EscapeString(meta))
		}
		b.WriteString(`</div>`)
	}

	b.WriteString(pageFooter())
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(b.String())
}

// PingClaude tests a minimal call to the Claude API.
func (h *Handler) PingClaude(c *fiber.Ctx) error {
	if h.cfg.AnthropicAPIKey == "" {
		return c.Status(fiber.StatusBadRequest).SendString("ANTHROPIC_API_KEY no configurada")
	}

	body := []byte(`{"model":"claude-sonnet-4-6","max_tokens":10,"messages":[{"role":"user","content":"ping"}]}`)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", h.cfg.AnthropicAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	start := time.Now()
	resp, err := h.httpCli.Do(req)
	took := time.Since(start)
	if err != nil {
		return c.Type("txt").SendString(fmt.Sprintf("FAIL: %s (took %s)", err, took))
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)

	result := map[string]any{
		"status_code": resp.StatusCode,
		"latency_ms":  took.Milliseconds(),
		"body":        truncate(buf.String(), 500),
	}
	return c.JSON(result)
}

// PingElevenLabs tests reachability of the ElevenLabs API (user endpoint).
func (h *Handler) PingElevenLabs(c *fiber.Ctx) error {
	if h.cfg.ElevenLabsAPIKey == "" {
		return c.Status(fiber.StatusBadRequest).SendString("ELEVENLABS_API_KEY no configurada")
	}
	req, _ := http.NewRequest("GET", "https://api.elevenlabs.io/v1/user", nil)
	req.Header.Set("xi-api-key", h.cfg.ElevenLabsAPIKey)

	start := time.Now()
	resp, err := h.httpCli.Do(req)
	took := time.Since(start)
	if err != nil {
		return c.Type("txt").SendString(fmt.Sprintf("FAIL: %s (took %s)", err, took))
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)

	result := map[string]any{
		"status_code": resp.StatusCode,
		"latency_ms":  took.Milliseconds(),
		"body":        truncate(buf.String(), 500),
	}
	return c.JSON(result)
}

// PingInstagram tests reachability of Instagram Graph API using the page token.
func (h *Handler) PingInstagram(c *fiber.Ctx) error {
	if h.cfg.PageAccessToken == "" {
		return c.Status(fiber.StatusBadRequest).SendString("PAGE_ACCESS_TOKEN no configurada")
	}
	url := fmt.Sprintf("https://graph.instagram.com/v21.0/me?access_token=%s", h.cfg.PageAccessToken)
	start := time.Now()
	resp, err := h.httpCli.Get(url)
	took := time.Since(start)
	if err != nil {
		return c.Type("txt").SendString(fmt.Sprintf("FAIL: %s (took %s)", err, took))
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)

	out := map[string]any{
		"status_code": resp.StatusCode,
		"latency_ms":  took.Milliseconds(),
	}
	var parsed map[string]any
	if json.Unmarshal(buf.Bytes(), &parsed) == nil {
		out["body"] = parsed
	} else {
		out["body"] = truncate(buf.String(), 500)
	}
	return c.JSON(out)
}

// recentLeads returns leads with activity in the window, plus their last message.
func (h *Handler) recentLeads(ctx context.Context, limit int, window time.Duration) ([]*storage.LeadRecord, map[string]*storage.ConversationMessage, error) {
	// There's no direct "list by last_seen" — use outcome=open ordered by last_seen (already in LeadsRepo).
	// We fetch open + won + lost to cover all activity, then filter by recent conversation.
	outcomes := []storage.LeadOutcome{storage.OutcomeOpen, storage.OutcomeWon, storage.OutcomeLost, storage.OutcomeAbandoned}
	var all []*storage.LeadRecord
	seen := make(map[string]bool)
	for _, o := range outcomes {
		list, err := h.leads.ListByOutcome(ctx, o, 200)
		if err != nil {
			return nil, nil, err
		}
		for _, l := range list {
			if !seen[l.ID] {
				all = append(all, l)
				seen[l.ID] = true
			}
		}
	}

	lastMsg := make(map[string]*storage.ConversationMessage)
	cutoff := time.Now().Add(-window)

	var filtered []*storage.LeadRecord
	for _, l := range all {
		msgs, err := h.conv.LastN(ctx, l.ID, 1)
		if err != nil || len(msgs) == 0 {
			continue
		}
		last := msgs[len(msgs)-1]
		if last.CreatedAt.Before(cutoff) {
			continue
		}
		lastMsg[l.ID] = &last
		filtered = append(filtered, l)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return lastMsg[filtered[i].ID].CreatedAt.After(lastMsg[filtered[j].ID].CreatedAt)
	})

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, lastMsg, nil
}

func maskToken(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "…" + s[len(s)-4:]
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func relTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "ahora"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func testSenderWarning(val string) string {
	if val == "" {
		return ""
	}
	return "ATENCIÓN: con TEST_SENDER_ID seteado, el bot SOLO responde a ese usuario. Borra esta variable en Railway y redeploy para responder a todos."
}

func pageHeader(title string) string {
	return `<!doctype html><html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + html.EscapeString(title) + `</title>
<style>
body{font-family:-apple-system,system-ui,sans-serif;max-width:1100px;margin:2em auto;padding:0 1em;color:#222;background:#fafafa}
h1,h2{color:#111}
.mono{font-family:SFMono-Regular,Menlo,Consolas,monospace;font-size:.92em}
table{border-collapse:collapse;margin:.5em 0 1.5em;width:100%}
table.kv td{padding:.4em .8em;border-bottom:1px solid #eee;vertical-align:top}
table.kv td:first-child{font-weight:600;width:200px}
table.leads th,table.leads td{padding:.4em .6em;border-bottom:1px solid #eee;text-align:left;font-size:.92em}
table.leads th{background:#f0f0f0}
tr.unanswered{background:#fff6d6}
.ok{color:#0a7c2f;font-weight:600}
.bad{color:#b00020;font-weight:600}
.warn{color:#b00020;font-size:.88em;margin-top:.3em}
.btn{display:inline-block;padding:.5em 1em;margin:.25em;background:#222;color:#fff;text-decoration:none;border-radius:4px;font-size:.9em}
.btn:hover{background:#000}
.chat{display:flex;flex-direction:column;gap:.4em;max-width:750px}
.bubble{padding:.6em .9em;border-radius:10px;max-width:85%;word-wrap:break-word}
.bubble.user{background:#fff;border:1px solid #ddd;align-self:flex-start}
.bubble.bot{background:#dbeafe;align-self:flex-end}
.bubble .meta{font-size:.72em;color:#666;margin-top:.3em}
</style></head><body><h1>` + html.EscapeString(title) + `</h1>`
}

func pageFooter() string {
	return `<hr><p><small>Panel de diagnóstico · Scuderia St4ge bot</small></p></body></html>`
}
