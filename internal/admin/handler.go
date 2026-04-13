package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/config"
	"github.com/kart-academy/instagram-bot/internal/domain"
	"github.com/kart-academy/instagram-bot/internal/queue"
	"github.com/kart-academy/instagram-bot/internal/storage"
)

type Handler struct {
	cfg       *config.Config
	leads     *storage.LeadsRepo
	conv      *storage.ConversationRepo
	messenger domain.Messenger
	ai        domain.AIEngine
	queue     queue.Queue
	logger    *zap.Logger
	httpCli   *http.Client
}

func NewHandler(
	cfg *config.Config,
	leads *storage.LeadsRepo,
	conv *storage.ConversationRepo,
	messenger domain.Messenger,
	ai domain.AIEngine,
	q queue.Queue,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		cfg:       cfg,
		leads:     leads,
		conv:      conv,
		messenger: messenger,
		ai:        ai,
		queue:     q,
		logger:    logger,
		httpCli:   &http.Client{Timeout: 15 * time.Second},
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
	}
	// TEST_SENDER_ID: empty is the GOOD state in prod (bot responde a todos).
	testOK := h.cfg.TestSenderID == ""
	testVal := h.cfg.TestSenderID
	if testOK {
		testVal = "(vacío — bot responde a todos ✓)"
	}
	rows = append(rows, struct {
		name string
		set  bool
		val  string
		warn string
	}{"TEST_SENDER_ID", testOK, testVal, testSenderWarning(h.cfg.TestSenderID)})

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

	// Real Instagram conversations (via Meta API)
	b.WriteString(`<h2>Conversaciones reales de Instagram</h2>`)
	fmt.Fprintf(&b, `<p><a class="btn" href="/admin/instagram/conversations?token=%s">Ver DMs reales de Meta (últimas 24h)</a></p>`, token)
	b.WriteString(`<p><small>Lista directa desde Meta Graph API, independiente del bot. Requiere permiso <code>instagram_manage_messages</code>.</small></p>`)

	// Queue stats
	b.WriteString(`<h2>Cola de mensajes</h2>`)
	if h.queue != nil {
		if stats, err := h.queue.Stats(ctx); err == nil {
			queueBadge := func(label string, n int, cls string) string {
				return fmt.Sprintf(`<span class="badge %s">%s: %d</span>`, cls, label, n)
			}
			warn := ""
			if stats.OldestPendingSeconds > 60 {
				warn = fmt.Sprintf(` <span class="bad">(job más viejo: %ds esperando)</span>`, stats.OldestPendingSeconds)
			}
			fmt.Fprintf(&b, `<p>%s %s %s %s %s%s</p>`,
				queueBadge("pending", stats.Pending, "ok"),
				queueBadge("processing", stats.Processing, "ok"),
				queueBadge("failed", stats.Failed, "warn-badge"),
				queueBadge("dead", stats.Dead, "bad-badge"),
				queueBadge("done 24h", stats.DoneLast24h, "ok"),
				warn,
			)
			fmt.Fprintf(&b, `<p><a class="btn" href="/admin/queue?token=%s">Ver dead-letter</a></p>`, token)
		} else {
			fmt.Fprintf(&b, `<p class="bad">Error cargando stats: %s</p>`, html.EscapeString(err.Error()))
		}
	} else {
		b.WriteString(`<p><em>Queue no inicializada.</em></p>`)
	}

	// Manual send to specific sender
	b.WriteString(`<h2>Atender a un usuario específico</h2>`)
	fmt.Fprintf(&b, `
	<form method="POST" action="/admin/send?token=%s" class="manual-send">
		<label>Sender ID (Instagram-scoped):
			<input type="text" name="sender_id" placeholder="ej: 7002416586476447" required style="width:280px">
		</label>
		<label>Modo:
			<select name="mode">
				<option value="ai">AI (Claude genera respuesta basada en historial + instrucción)</option>
				<option value="literal">Literal (enviar texto exacto sin AI)</option>
			</select>
		</label>
		<label>Instrucción / mensaje (opcional en modo AI):
			<textarea name="instruction" rows="3" placeholder="Ej AI: 'mándale el link de pago y confirma el horario'. Ej Literal: 'Hola Juan, ya está listo tu cupo, te mando el link?'"></textarea>
		</label>
		<button class="btn" type="submit">Enviar mensaje al usuario</button>
	</form>
	<p><small>Modo <strong>AI</strong>: tu instrucción guía al bot, que escribe el mensaje siguiendo tono y contexto.<br>
	Modo <strong>Literal</strong>: el texto va tal cual al usuario, sin pasar por Claude.<br>
	Sin instrucción + AI: el bot retoma automáticamente la conversación donde quedó.</small></p>`,
		token)

	// Recent leads
	b.WriteString(`<h2>Últimos leads (24h)</h2>`)
	leads, info, err := h.recentLeads(ctx, 50, 24*time.Hour)
	if err != nil {
		fmt.Fprintf(&b, `<p class="bad">Error cargando leads: %s</p>`, html.EscapeString(err.Error()))
	} else if len(leads) == 0 {
		b.WriteString(`<p><em>No hay leads con actividad en las últimas 24h.</em></p>`)
	} else {
		b.WriteString(`<table class="leads"><tr><th>Lead ID</th><th>Score</th><th>Estado</th><th>Msgs</th><th>Último mensaje del usuario</th><th>Cuándo</th><th>Estado resp.</th><th></th></tr>`)
		for _, l := range leads {
			nfo := info[l.ID]
			userPreview := "—"
			when := ""
			respStatus := `<span class="ok">respondido</span>`
			rowClass := ""
			if nfo != nil {
				if nfo.LastUserMsg != nil {
					userPreview = truncate(nfo.LastUserMsg.Content, 90)
					when = relTime(nfo.LastUserMsg.CreatedAt)
				}
				if nfo.Unanswered {
					respStatus = `<span class="bad">SIN RESPUESTA</span>`
					rowClass = ` class="unanswered"`
				}
			}
			fmt.Fprintf(&b, `<tr%s><td class="mono">%s</td><td>%d</td><td>%s</td><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td><a href="/admin/lead/%s?token=%s">ver</a></td></tr>`,
				rowClass,
				html.EscapeString(l.ID),
				l.LeadScore,
				html.EscapeString(string(l.State)),
				l.TotalMessages,
				html.EscapeString(userPreview),
				when,
				respStatus,
				html.EscapeString(l.ID),
				token,
			)
		}
		b.WriteString(`</table>`)
		b.WriteString(`<p><small>Filas amarillas: el último mensaje fue del usuario y el bot no ha respondido.</small></p>`)
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

	fmt.Fprintf(&b, `<h2>Acciones</h2>
	<form method="POST" action="/admin/lead/%s/retake?token=%s" style="display:inline">
		<button class="btn" type="submit">Retomar conversación ahora (bot responde)</button>
	</form>
	<p><small>Hace que el bot lea el historial y mande un mensaje para empujar al cierre. Si el AI considera que ya no aplica, devuelve SKIP.</small></p>`,
		html.EscapeString(leadID), h.cfg.AdminToken)

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

// InstagramConversations pulls the real list of DMs from Meta Graph API and renders them.
func (h *Handler) InstagramConversations(c *fiber.Ctx) error {
	if h.cfg.PageAccessToken == "" {
		return c.Status(fiber.StatusBadRequest).SendString("PAGE_ACCESS_TOKEN no configurada")
	}

	q := url.Values{}
	q.Set("platform", "instagram")
	q.Set("fields", "participants,updated_time,messages.limit(3){from,message,created_time}")
	q.Set("access_token", h.cfg.PageAccessToken)
	q.Set("limit", "50")
	apiURL := "https://graph.instagram.com/v21.0/me/conversations?" + q.Encode()

	resp, err := h.httpCli.Get(apiURL)
	if err != nil {
		return c.Type("txt").SendString("Error red: " + err.Error())
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)

	var parsed struct {
		Data []struct {
			ID           string `json:"id"`
			UpdatedTime  string `json:"updated_time"`
			Participants struct {
				Data []struct {
					ID       string `json:"id"`
					Username string `json:"username"`
				} `json:"data"`
			} `json:"participants"`
			Messages struct {
				Data []struct {
					ID          string `json:"id"`
					Message     string `json:"message"`
					CreatedTime string `json:"created_time"`
					From        struct {
						ID       string `json:"id"`
						Username string `json:"username"`
					} `json:"from"`
				} `json:"data"`
			} `json:"messages"`
		} `json:"data"`
		Error map[string]any `json:"error"`
	}
	_ = json.Unmarshal(buf.Bytes(), &parsed)

	var b strings.Builder
	b.WriteString(pageHeader("Conversaciones Instagram (últimas 24h)"))
	fmt.Fprintf(&b, `<p><a href="/admin?token=%s">&larr; volver al panel</a></p>`, h.cfg.AdminToken)

	if resp.StatusCode != http.StatusOK || parsed.Error != nil {
		fmt.Fprintf(&b, `<div class="warn"><strong>Meta devolvió error (status %d)</strong></div>`, resp.StatusCode)
		fmt.Fprintf(&b, `<pre class="mono" style="background:#fff;padding:1em;border:1px solid #ddd;white-space:pre-wrap">%s</pre>`, html.EscapeString(truncate(buf.String(), 2000)))
		fmt.Fprintf(&b, `<p><strong>Causas comunes:</strong><br>
			• Falta el permiso <code>instagram_manage_messages</code> en la app de Meta.<br>
			• La app está en modo desarrollo y tu cuenta no es Instagram Tester.<br>
			• El <code>PAGE_ACCESS_TOKEN</code> expiró o no tiene scope suficiente.</p>`)
		b.WriteString(pageFooter())
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(b.String())
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	shown := 0
	b.WriteString(`<table class="leads"><tr><th>Usuario</th><th>Scoped ID</th><th>Último mensaje</th><th>De</th><th>Cuándo</th><th></th></tr>`)
	for _, conv := range parsed.Data {
		updated, _ := time.Parse(time.RFC3339, conv.UpdatedTime)
		if updated.Before(cutoff) {
			continue
		}
		shown++
		var theirUser, theirID string
		for _, p := range conv.Participants.Data {
			if p.ID != h.cfg.InstagramAccountID {
				theirUser, theirID = p.Username, p.ID
				break
			}
		}
		if theirUser == "" {
			theirUser = "(desconocido)"
		}

		lastMsgText, lastFrom, lastWhen := "—", "", ""
		if len(conv.Messages.Data) > 0 {
			m := conv.Messages.Data[0]
			lastMsgText = truncate(m.Message, 120)
			if m.From.ID == h.cfg.InstagramAccountID {
				lastFrom = "bot/staff"
			} else {
				lastFrom = "usuario"
			}
			t, _ := time.Parse(time.RFC3339, m.CreatedTime)
			lastWhen = relTime(t)
		}

		retakeBtn := ""
		if theirID != "" {
			retakeBtn = fmt.Sprintf(`<form method="POST" action="/admin/lead/%s/retake?token=%s" style="display:inline"><button class="btn" type="submit">Retomar</button></form>`,
				html.EscapeString(theirID), h.cfg.AdminToken)
		}
		fmt.Fprintf(&b, `<tr><td>@%s</td><td class="mono">%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			html.EscapeString(theirUser),
			html.EscapeString(theirID),
			html.EscapeString(lastMsgText),
			html.EscapeString(lastFrom),
			html.EscapeString(lastWhen),
			retakeBtn,
		)
	}
	b.WriteString(`</table>`)
	if shown == 0 {
		b.WriteString(`<p><em>No hay conversaciones con actividad en las últimas 24h.</em></p>`)
	}

	b.WriteString(pageFooter())
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(b.String())
}

// RetakeLead runs the Brain pipeline with a synthetic follow-up input and sends the reply.
func (h *Handler) RetakeLead(c *fiber.Ctx) error {
	leadID := c.Params("id")
	if leadID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("lead id requerido")
	}
	return h.dispatchSend(c, leadID, "", "ai")
}

// SendToUser handles the manual send form: sender_id + instruction + mode.
// Modes:
//   - "ai" (default): instruction (or synthetic followup if empty) is fed to Brain.Process
//   - "literal": instruction is sent verbatim as a text message, bypassing Claude
func (h *Handler) SendToUser(c *fiber.Ctx) error {
	senderID := strings.TrimSpace(c.FormValue("sender_id"))
	instruction := strings.TrimSpace(c.FormValue("instruction"))
	mode := c.FormValue("mode")
	if mode == "" {
		mode = "ai"
	}
	if senderID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("sender_id requerido")
	}
	return h.dispatchSend(c, senderID, instruction, mode)
}

// dispatchSend is the shared logic for both retake-by-lead and manual-send-by-id.
func (h *Handler) dispatchSend(c *fiber.Ctx, senderID, instruction, mode string) error {
	if h.ai == nil || h.messenger == nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString("AI o Messenger no configurados")
	}

	var sent string
	switch mode {
	case "literal":
		if instruction == "" {
			return c.Status(fiber.StatusBadRequest).SendString("modo literal requiere texto en 'instruction'")
		}
		if err := h.messenger.SendText(senderID, instruction); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error al enviar: " + err.Error())
		}
		sent = instruction
		h.logger.Info("manual literal sent",
			zap.String("sender_id", senderID),
			zap.String("text", truncate(sent, 120)))

	default: // "ai"
		input := instruction
		allowSkip := false
		if input == "" {
			// Auto follow-up: AI decides if it makes sense to send anything.
			input = "[SISTEMA INTERNO — NO ES MENSAJE DEL CLIENTE] El cliente quedó sin respuesta. Lee el historial completo de la conversación, detecta en qué punto del funnel estaba, y retoma con un mensaje corto, cálido y orientado al siguiente paso. Si la conversación ya estaba cerrada (venta hecha o cliente dijo no definitivo), responde solo con la palabra SKIP."
			allowSkip = true
		} else {
			// Explicit operator instruction: always produce a message.
			input = "[INSTRUCCIÓN INTERNA DEL EQUIPO — NO ES MENSAJE DEL CLIENTE, NO LA REPITAS TAL CUAL] " +
				instruction +
				". Redacta el mensaje al cliente siguiendo el tono natural del bot y el historial si existe. Si el cliente es nuevo y no hay historial, saluda y arranca la conversación de forma cálida y corta. NO respondas SKIP: el equipo ya decidió enviar algo, tu trabajo es redactar el mejor mensaje posible."
		}
		reply, err := h.ai.Process(senderID, input)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("AI error: " + err.Error())
		}
		if allowSkip && strings.TrimSpace(strings.ToUpper(reply)) == "SKIP" {
			return utf8Text(c, "El AI decidió saltarse: "+senderID+" (no aplica enviar mensaje)")
		}
		if err := h.messenger.SendText(senderID, reply); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error al enviar: " + err.Error())
		}
		sent = reply
		h.logger.Info("manual ai-send sent",
			zap.String("sender_id", senderID),
			zap.String("instruction", truncate(instruction, 120)),
			zap.String("reply", truncate(reply, 120)))
	}

	return utf8Text(c, "OK — enviado a "+senderID+":\n\n"+sent)
}

func utf8Text(c *fiber.Ctx, body string) error {
	c.Set("Content-Type", "text/plain; charset=utf-8")
	return c.SendString(body)
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

// leadInfo bundles per-lead data needed for the dashboard row.
type leadInfo struct {
	LastUserMsg *storage.ConversationMessage
	LastAnyMsg  *storage.ConversationMessage
	Unanswered  bool // latest message is from user
}

// recentLeads returns leads with activity in the window plus their last user message
// and a flag indicating whether the bot has responded to it.
func (h *Handler) recentLeads(ctx context.Context, limit int, window time.Duration) ([]*storage.LeadRecord, map[string]*leadInfo, error) {
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

	info := make(map[string]*leadInfo)
	cutoff := time.Now().Add(-window)

	var filtered []*storage.LeadRecord
	for _, l := range all {
		msgs, err := h.conv.LastN(ctx, l.ID, 10)
		if err != nil || len(msgs) == 0 {
			continue
		}
		nfo := &leadInfo{}
		// msgs is chronological ASC; last is newest.
		for i := len(msgs) - 1; i >= 0; i-- {
			if nfo.LastAnyMsg == nil {
				m := msgs[i]
				nfo.LastAnyMsg = &m
			}
			if msgs[i].Role == storage.RoleUser && nfo.LastUserMsg == nil {
				m := msgs[i]
				nfo.LastUserMsg = &m
				break
			}
		}
		if nfo.LastAnyMsg == nil || nfo.LastAnyMsg.CreatedAt.Before(cutoff) {
			continue
		}
		nfo.Unanswered = nfo.LastAnyMsg.Role == storage.RoleUser
		info[l.ID] = nfo
		filtered = append(filtered, l)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return info[filtered[i].ID].LastAnyMsg.CreatedAt.After(info[filtered[j].ID].LastAnyMsg.CreatedAt)
	})

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, info, nil
}

// QueueView renders the dead-letter list with retry buttons.
func (h *Handler) QueueView(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	var b strings.Builder
	b.WriteString(pageHeader("Cola de mensajes — dead-letter"))
	fmt.Fprintf(&b, `<p><a href="/admin?token=%s">&larr; volver</a></p>`, h.cfg.AdminToken)

	if h.queue == nil {
		b.WriteString(`<p class="bad">Queue no inicializada.</p>`)
		b.WriteString(pageFooter())
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(b.String())
	}

	stats, _ := h.queue.Stats(ctx)
	fmt.Fprintf(&b, `<p><strong>Stats:</strong> pending=%d, processing=%d, failed=%d, dead=%d, done 24h=%d</p>`,
		stats.Pending, stats.Processing, stats.Failed, stats.Dead, stats.DoneLast24h)

	dead, err := h.queue.DeadLetter(ctx, 50)
	if err != nil {
		fmt.Fprintf(&b, `<p class="bad">Error: %s</p>`, html.EscapeString(err.Error()))
		b.WriteString(pageFooter())
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(b.String())
	}

	if len(dead) == 0 {
		b.WriteString(`<p><em>Sin mensajes en dead-letter.</em></p>`)
	} else {
		b.WriteString(`<table class="leads"><tr><th>ID</th><th>Sender</th><th>Attempts</th><th>Error</th><th>Created</th><th></th></tr>`)
		for _, j := range dead {
			fmt.Fprintf(&b, `<tr><td>%d</td><td class="mono">%s</td><td>%d</td><td>%s</td><td>%s</td><td>
				<form method="POST" action="/admin/queue/retry/%d?token=%s" style="display:inline">
					<button class="btn" type="submit">Reintentar</button>
				</form></td></tr>`,
				j.ID, html.EscapeString(j.SenderID), j.Attempts,
				html.EscapeString(truncate(j.ErrorText, 200)),
				j.CreatedAt.Format("2006-01-02 15:04"),
				j.ID, h.cfg.AdminToken)
		}
		b.WriteString(`</table>`)
	}

	b.WriteString(pageFooter())
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(b.String())
}

// QueueRetry moves a dead/failed job back to pending.
func (h *Handler) QueueRetry(c *fiber.Ctx) error {
	if h.queue == nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString("Queue no inicializada")
	}
	idStr := c.Params("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("id inválido")
	}
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()
	if err := h.queue.RetryJob(ctx, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	return c.Redirect("/admin/queue?token="+h.cfg.AdminToken, fiber.StatusSeeOther)
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
.badge{display:inline-block;padding:.25em .6em;border-radius:12px;font-size:.85em;margin-right:.4em}
.badge.ok{background:#dcfce7;color:#065f46}
.badge.warn-badge{background:#fef3c7;color:#92400e}
.badge.bad-badge{background:#fee2e2;color:#991b1b}
form.manual-send{background:#fff;padding:1em;border:1px solid #ddd;border-radius:6px;max-width:700px}
form.manual-send label{display:block;margin:.6em 0;font-size:.9em;color:#444;font-weight:600}
form.manual-send input,form.manual-send select,form.manual-send textarea{display:block;width:100%;padding:.5em;border:1px solid #ccc;border-radius:4px;font-size:.95em;margin-top:.25em;font-family:inherit;box-sizing:border-box}
form.manual-send textarea{resize:vertical}
form.manual-send button{margin-top:.5em}
</style></head><body><h1>` + html.EscapeString(title) + `</h1>`
}

func pageFooter() string {
	return `<hr><p><small>Panel de diagnóstico · Scuderia St4ge bot</small></p></body></html>`
}
