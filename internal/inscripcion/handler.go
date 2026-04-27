package inscripcion

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"crypto/rand"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/kart-academy/instagram-bot/internal/storage"
)

//go:embed form.html
var formHTML string

//go:embed landing.html
var landingHTML string

//go:embed success.html
var successHTML string

//go:embed assets/logo.jpg
var logoJPG []byte

//go:embed assets/bancolombiaLogo.png
var bancolombiaLogoPNG []byte

//go:embed assets/nequiLogo.webp
var nequiLogoWEBP []byte

// Plan describes a registration plan.
type Plan struct {
	ID         string // form value
	Label      string
	PriceCOP   int    // total price (per person)
	BoldLink   string // optional, for "tarjeta"
	Note       string
	IsPreventa bool
}

// Modalidad determines whether the user pays only the cup reservation or the
// full preventa price.
type Modalidad struct {
	ID       string
	Label    string
	PriceCOP int
}

var Modalidades = []Modalidad{
	{ID: "reserva", Label: "Solo reserva del cupo", PriceCOP: 150000},
	{ID: "completo", Label: "Pago completo (preventa)", PriceCOP: 730000},
}

func findModalidad(id string) *Modalidad {
	for i := range Modalidades {
		if Modalidades[i].ID == id {
			return &Modalidades[i]
		}
	}
	return nil
}

// Legacy fallback link used only if Bold API is unavailable.
const ReservaBoldLink = "https://checkout.bold.co/payment/LNK_UQU9WHRQDT"
const ReservaCOP = 150000

// CardSurchargePct is the extra cost for paying with credit card.
const CardSurchargePct = 5

// Default plan label kept for the records table (we no longer surface multiple
// plans in the form; modalidad replaces that concept).
const DefaultPlanID = "preventa"

// Available course dates (label shown to user). Match the values rendered in
// form.html.
var CourseDates = []string{
	"MAYO 9 y 10",
	"MAYO 23 y 24",
}

// PaymentMethods exposes every method accepted by the form. A method is
// "digital" when it is processed through Bold (we generate a checkout URL on
// the fly). The "transferencia" fallback is manual and requires comprobante.
type PaymentMethod struct {
	ID         string
	Label      string
	BoldMethod BoldPaymentMethod // empty for the manual transfer fallback
}

var PaymentMethods = []PaymentMethod{
	{ID: "tarjeta", Label: "Tarjeta de crédito / débito", BoldMethod: BoldMethodCard},
	{ID: "nequi", Label: "Nequi", BoldMethod: BoldMethodNequi},
	{ID: "bancolombia", Label: "Botón Bancolombia", BoldMethod: BoldMethodBancolombia},
	{ID: "pse", Label: "PSE — otros bancos", BoldMethod: BoldMethodPSE},
	{ID: "transferencia", Label: "Transferencia manual + comprobante"},
}

func findMethod(id string) *PaymentMethod {
	for i := range PaymentMethods {
		if PaymentMethods[i].ID == id {
			return &PaymentMethods[i]
		}
	}
	return nil
}

type Config struct {
	UploadsDir        string // directory for receipt uploads, e.g. ./uploads
	PublicURL         string // for absolute redirect after success
	TelegramBotToken  string
	TelegramChatID    string
	AdminToken        string // required to call diagnostic endpoints
	BoldWebhookSecret string // shared secret to verify Bold webhook HMAC
	BoldAPIKey        string // identity key to call Bold's link-creation API
	SheetsWebhookURL  string // Apps Script web-app URL receiving inscripciones
	SheetsSharedToken string // shared secret echoed in payload for verification
}

type Handler struct {
	cfg    Config
	repo   *storage.InscripcionesRepo
	logger *zap.Logger
}

func NewHandler(cfg Config, repo *storage.InscripcionesRepo, logger *zap.Logger) *Handler {
	if cfg.UploadsDir == "" {
		cfg.UploadsDir = "./uploads"
	}
	_ = os.MkdirAll(cfg.UploadsDir, 0o755)
	return &Handler{cfg: cfg, repo: repo, logger: logger}
}

// Serve Landing
func (h *Handler) ServeLanding(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(landingHTML)
}

// ServeForm renders the public registration form.
func (h *Handler) ServeForm(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(formHTML)
}

// ServeLogo serves the brand logo bundled with the binary.
func (h *Handler) ServeLogo(c *fiber.Ctx) error {
	c.Set("Content-Type", "image/jpeg")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.Send(logoJPG)
}

// ServeBancolombiaLogo serves bancolombia logo.
func (h *Handler) ServeBancolombiaLogo(c *fiber.Ctx) error {
	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.Send(bancolombiaLogoPNG)
}

// ServeNequiLogo serves nequi logo.
func (h *Handler) ServeNequiLogo(c *fiber.Ctx) error {
	c.Set("Content-Type", "image/webp")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.Send(nequiLogoWEBP)
}

// TelegramDebug returns the bot identity (getMe) and the latest chats it has
// seen messages from (getUpdates). Use this to find the correct chat_id.
// Protected by the ADMIN_TOKEN query param.
func (h *Handler) TelegramDebug(c *fiber.Ctx) error {
	if h.cfg.AdminToken == "" || c.Query("token") != h.cfg.AdminToken {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"ok": false, "error": "unauthorized"})
	}
	if h.cfg.TelegramBotToken == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"ok": false, "error": "TELEGRAM_BOT_TOKEN missing"})
	}
	out := fiber.Map{
		"configured_chat_id": h.cfg.TelegramChatID,
	}
	if me, err := tgGetJSON(h.cfg.TelegramBotToken, "getMe"); err == nil {
		out["getMe"] = me
	} else {
		out["getMe_error"] = err.Error()
	}
	if upd, err := tgGetJSON(h.cfg.TelegramBotToken, "getUpdates"); err == nil {
		// Extract just the chats the bot has seen, to make the response readable.
		seen := map[string]any{}
		if result, ok := upd["result"].([]any); ok {
			for _, u := range result {
				if m, ok := u.(map[string]any); ok {
					var msg map[string]any
					if x, ok := m["message"].(map[string]any); ok {
						msg = x
					} else if x, ok := m["channel_post"].(map[string]any); ok {
						msg = x
					} else if x, ok := m["my_chat_member"].(map[string]any); ok {
						msg = x
					}
					if msg == nil {
						continue
					}
					if chat, ok := msg["chat"].(map[string]any); ok {
						id := fmt.Sprint(chat["id"])
						seen[id] = chat
					}
				}
			}
		}
		out["chats_seen"] = seen
		out["raw_updates_count"] = len(upd["result"].([]any))
		out["hint"] = "Si el grupo no aparece en chats_seen, manda un mensaje en el grupo (con el bot dentro) y vuelve a refrescar."
	} else {
		out["getUpdates_error"] = err.Error()
	}
	return c.JSON(out)
}

func tgGetJSON(token, method string) (map[string]any, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", token, method)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode: %w (body=%s)", err, string(body))
	}
	return out, nil
}

// TelegramTest sends a test notification to the configured chat. Useful to
// verify TELEGRAM_BOT_TOKEN / TELEGRAM_CHAT_ID without doing a full
// registration. Protected by the ADMIN_TOKEN query param.
func (h *Handler) TelegramTest(c *fiber.Ctx) error {
	if h.cfg.AdminToken == "" || c.Query("token") != h.cfg.AdminToken {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"ok":    false,
			"error": "unauthorized: pass ?token=ADMIN_TOKEN",
		})
	}
	if h.cfg.TelegramBotToken == "" || h.cfg.TelegramChatID == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"ok":     false,
			"error":  "Telegram no configurado: faltan TELEGRAM_BOT_TOKEN y/o TELEGRAM_CHAT_ID",
			"hasToken": h.cfg.TelegramBotToken != "",
			"hasChat":  h.cfg.TelegramChatID != "",
		})
	}
	msg := fmt.Sprintf("✅ *Test de Scudería St4ge* — %s\n\nSi ves este mensaje, la integración con el grupo está activa. Las nuevas inscripciones llegarán aquí con el comprobante adjunto.",
		time.Now().Format("2006-01-02 15:04:05"))
	if err := h.tgSendMessage(msg); err != nil {
		h.logger.Error("telegram test failed", zap.Error(err))
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"ok":    false,
			"error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"ok": true, "message": "Mensaje enviado al grupo de Telegram"})
}

// Submit handles POSTed registration data.
func (h *Handler) Submit(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return errPage(c, fiber.StatusBadRequest, "Formulario inválido", err.Error())
	}

	rec, modalidad, method, validationErr := parseForm(form)
	if validationErr != nil {
		return errPage(c, fiber.StatusBadRequest, "Faltan datos obligatorios", validationErr.Error())
	}

	// Manual transfer is the only path that requires a receipt upload.
	files := form.File["comprobante"]
	if method.ID == "transferencia" && len(files) == 0 {
		return errPage(c, fiber.StatusBadRequest, "Comprobante requerido",
			"Para transferencia manual debes adjuntar el comprobante de la reserva ($150.000).")
	}

	rec.ID = newID()
	rec.Status = "pendiente"
	if method.ID == "transferencia" && len(files) > 0 {
		path, err := h.saveReceipt(rec.ID, files[0])
		if err != nil {
			h.logger.Error("save receipt", zap.Error(err))
			return errPage(c, fiber.StatusInternalServerError, "Error subiendo comprobante", err.Error())
		}
		rec.ComprobantePath = path
		rec.Status = "comprobante recibido, en validación"
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 15*time.Second)
	defer cancel()
	if err := h.repo.Insert(ctx, rec); err != nil {
		h.logger.Error("insert inscripcion", zap.Error(err))
		return errPage(c, fiber.StatusInternalServerError, "Error guardando inscripción", err.Error())
	}

	// For digital methods, mint a Bold link tailored to the chosen method.
	var checkoutURL string
	if method.BoldMethod != "" {
		url, err := h.CreateBoldLink(ctx, method.BoldMethod, rec.MontoCOP, rec.ID,
			boldDescription(modalidad.ID, rec.Edad), rec.Email)
		if err != nil {
			h.logger.Error("bold link create failed", zap.Error(err))
			// Fall back to the static reservation link so the user can still pay.
			url = ReservaBoldLink + "?reference=" + rec.ID
		}
		checkoutURL = url
	}

	go h.notifyDirector(*rec, *modalidad, *method, checkoutURL)
	go h.pushToSheets(context.Background(), *rec, modalidad.ID, checkoutURL)

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(renderSuccess(rec, modalidad, method, checkoutURL))
}

// parseForm extracts and validates form fields. Returns the partial record,
// the chosen Modalidad and PaymentMethod.
func parseForm(form *multipart.Form) (*storage.InscripcionRecord, *Modalidad, *PaymentMethod, error) {
	get := func(k string) string {
		v := form.Value[k]
		if len(v) == 0 {
			return ""
		}
		return strings.TrimSpace(v[0])
	}

	rec := &storage.InscripcionRecord{
		Email:            strings.ToLower(get("email")),
		MetodoPago:       get("metodo_pago"),
		FechaCurso:       get("fecha_curso"),
		Plan:             DefaultPlanID,
		NombrePiloto:     get("nombre_piloto"),
		TipoDocumento:    get("tipo_documento"),
		NumeroDocumento:  get("numero_documento"),
		Telefono:         get("telefono"),
		Ciudad:           get("ciudad"),
		EPS:              get("eps"),
		GrupoSanguineo:   get("grupo_sanguineo"),
		FamiliarNombre:   get("familiar_nombre"),
		FamiliarTelefono: get("familiar_telefono"),
		InstagramUser:    get("instagram_user"),
	}

	if rec.Email == "" || !validEmail(rec.Email) {
		return nil, nil, nil, errors.New("email inválido")
	}
	if rec.NombrePiloto == "" {
		return nil, nil, nil, errors.New("nombre del piloto es obligatorio")
	}
	if rec.NumeroDocumento == "" {
		return nil, nil, nil, errors.New("número de documento es obligatorio")
	}
	if rec.Telefono == "" {
		return nil, nil, nil, errors.New("teléfono es obligatorio")
	}
	if rec.EPS == "" || rec.GrupoSanguineo == "" {
		return nil, nil, nil, errors.New("EPS y grupo sanguíneo son obligatorios")
	}
	if rec.FamiliarNombre == "" || rec.FamiliarTelefono == "" {
		return nil, nil, nil, errors.New("contacto familiar de emergencia es obligatorio")
	}

	edadStr := get("edad")
	if edadStr == "" {
		return nil, nil, nil, errors.New("edad es obligatoria")
	}
	edad, err := strconv.Atoi(edadStr)
	if err != nil || edad < 8 || edad > 90 {
		return nil, nil, nil, errors.New("edad fuera de rango (8-90)")
	}
	rec.Edad = edad

	modalidad := findModalidad(get("modalidad"))
	if modalidad == nil {
		return nil, nil, nil, errors.New("modalidad inválida (elige reserva o pago completo)")
	}
	rec.Plan = modalidad.ID

	method := findMethod(rec.MetodoPago)
	if method == nil {
		return nil, nil, nil, errors.New("método de pago inválido")
	}
	if !validDate(rec.FechaCurso) {
		return nil, nil, nil, errors.New("fecha de curso inválida")
	}

	monto := modalidad.PriceCOP
	if method.ID == "tarjeta" {
		monto = monto + (monto*CardSurchargePct)/100
	}
	rec.MontoCOP = monto

	return rec, modalidad, method, nil
}

func validDate(d string) bool {
	for _, c := range CourseDates {
		if c == d {
			return true
		}
	}
	return false
}

var emailRe = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)

func validEmail(e string) bool { return emailRe.MatchString(e) }

func (h *Handler) saveReceipt(id string, fh *multipart.FileHeader) (string, error) {
	if fh.Size > 10*1024*1024 {
		return "", errors.New("archivo supera 10 MB")
	}
	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext == "" {
		ext = ".bin"
	}
	out := filepath.Join(h.cfg.UploadsDir, id+ext)
	dst, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return out, nil
}

func (h *Handler) notifyDirector(rec storage.InscripcionRecord, modalidad Modalidad, method PaymentMethod, checkoutURL string) {
	if h.cfg.TelegramBotToken == "" || h.cfg.TelegramChatID == "" {
		h.logger.Info("inscripcion received (no Telegram configured)",
			zap.String("id", rec.ID),
			zap.String("piloto", rec.NombrePiloto),
			zap.String("email", rec.Email),
			zap.String("modalidad", modalidad.ID),
			zap.Int("monto", rec.MontoCOP))
		return
	}

	caption := buildTelegramMessage(rec, modalidad, method, checkoutURL)

	if rec.ComprobantePath != "" {
		if err := h.tgSendPhoto(rec.ComprobantePath, caption); err != nil {
			h.logger.Error("telegram sendPhoto failed, falling back to text", zap.Error(err))
			if err := h.tgSendMessage(caption); err != nil {
				h.logger.Error("telegram sendMessage fallback failed", zap.Error(err))
			}
		}
		return
	}

	if err := h.tgSendMessage(caption); err != nil {
		h.logger.Error("telegram sendMessage failed", zap.Error(err))
	}
}

func buildTelegramMessage(rec storage.InscripcionRecord, modalidad Modalidad, method PaymentMethod, checkoutURL string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🏁 *Nueva inscripción* — %s\n\n", tgEscape(modalidad.Label))
	fmt.Fprintf(&b, "*ID:* `%s`\n", tgEscape(rec.ID))
	fmt.Fprintf(&b, "*Estado:* %s\n", tgEscape(rec.Status))
	fmt.Fprintf(&b, "*Monto:* $%s COP\n", tgEscape(formatCOP(rec.MontoCOP)))
	fmt.Fprintf(&b, "*Pago:* %s\n", tgEscape(method.Label))
	fmt.Fprintf(&b, "*Fecha curso:* %s\n", tgEscape(rec.FechaCurso))
	if checkoutURL != "" {
		fmt.Fprintf(&b, "*Link Bold:* %s\n", checkoutURL)
	}
	b.WriteString("\n👤 *Piloto*\n")
	fmt.Fprintf(&b, "%s, %d años\n", tgEscape(rec.NombrePiloto), rec.Edad)
	fmt.Fprintf(&b, "%s %s\n", tgEscape(rec.TipoDocumento), tgEscape(rec.NumeroDocumento))
	fmt.Fprintf(&b, "📱 %s\n", tgEscape(rec.Telefono))
	fmt.Fprintf(&b, "✉️ %s\n", tgEscape(rec.Email))
	if rec.Ciudad != "" {
		fmt.Fprintf(&b, "🏙 %s\n", tgEscape(rec.Ciudad))
	}
	if rec.InstagramUser != "" {
		fmt.Fprintf(&b, "📸 IG: %s\n", tgEscape(rec.InstagramUser))
	}
	fmt.Fprintf(&b, "\n🩺 EPS: %s · Sangre: %s\n", tgEscape(rec.EPS), tgEscape(rec.GrupoSanguineo))
	fmt.Fprintf(&b, "🆘 Emergencia: %s — %s", tgEscape(rec.FamiliarNombre), tgEscape(rec.FamiliarTelefono))
	return b.String()
}

// tgEscape escapes characters reserved by Telegram's Markdown (legacy) parser.
func tgEscape(s string) string {
	r := strings.NewReplacer("_", `\_`, "*", `\*`, "[", `\[`, "`", `\` + "`")
	return r.Replace(s)
}

func (h *Handler) tgSendMessage(text string) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", h.cfg.TelegramBotToken)
	body := url.Values{}
	body.Set("chat_id", h.cfg.TelegramChatID)
	body.Set("text", text)
	body.Set("parse_mode", "Markdown")
	resp, err := http.PostForm(endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram %d: %s", resp.StatusCode, string(out))
	}
	return nil
}

func (h *Handler) tgSendPhoto(path, caption string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("chat_id", h.cfg.TelegramChatID)
	_ = mw.WriteField("caption", caption)
	_ = mw.WriteField("parse_mode", "Markdown")
	fw, err := mw.CreateFormFile("photo", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return err
	}
	mw.Close()

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", h.cfg.TelegramBotToken)
	req, err := http.NewRequest("POST", endpoint, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram %d: %s", resp.StatusCode, string(out))
	}
	return nil
}

func formatCOP(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, r := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, r)
	}
	return string(out)
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "INS-" + hex.EncodeToString(b)
}

func errPage(c *fiber.Ctx, status int, title, detail string) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	html := strings.ReplaceAll(strings.ReplaceAll(errorHTML, "{{TITLE}}", htmlEscape(title)), "{{DETAIL}}", htmlEscape(detail))
	return c.Status(status).SendString(html)
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return r.Replace(s)
}

const errorHTML = `<!doctype html><html lang="es"><head><meta charset="utf-8"><title>{{TITLE}}</title>
<style>body{font-family:-apple-system,Segoe UI,Roboto,sans-serif;background:#0f0f1e;color:#fff;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}.card{background:#1a1a2e;padding:32px;border-radius:12px;max-width:480px;text-align:center}h1{color:#ff6b6b;margin:0 0 12px}p{color:#bcbccc;line-height:1.5}a{color:#ffb800;text-decoration:none}</style>
</head><body><div class="card"><h1>{{TITLE}}</h1><p>{{DETAIL}}</p><p><a href="/inscripcion">← Volver al formulario</a></p></div></body></html>`

func renderSuccess(rec *storage.InscripcionRecord, modalidad *Modalidad, method *PaymentMethod, checkoutURL string) string {
	nextStep := "Tu inscripción quedó registrada. Pronto te contactaremos por WhatsApp para confirmar."
	if checkoutURL != "" {
		nextStep = fmt.Sprintf("Continúa al pago seguro con %s. Tu cupo se confirma automáticamente al recibir el pago.", method.Label)
	} else if method.ID == "transferencia" {
		nextStep = "Recibimos tu comprobante. Lo validaremos y te contactaremos por WhatsApp para confirmar."
	}

	bigBoldButton := ""
	if checkoutURL != "" {
		bigBoldButton = fmt.Sprintf(`<a href="%s" target="_blank" class="bold-btn">💳 Continuar al pago — $%s COP →</a>`,
			htmlEscape(checkoutURL), formatCOP(rec.MontoCOP))
	}

	r := strings.NewReplacer(
		"{{ID}}", htmlEscape(rec.ID),
		"{{NOMBRE}}", htmlEscape(rec.NombrePiloto),
		"{{PLAN}}", htmlEscape(modalidad.Label),
		"{{MONTO}}", formatCOP(rec.MontoCOP),
		"{{FECHA}}", htmlEscape(rec.FechaCurso),
		"{{STATUS}}", htmlEscape(rec.Status),
		"{{EMAIL}}", htmlEscape(rec.Email),
		"{{NEXT_STEP}}", htmlEscape(nextStep),
		"{{METHOD}}", htmlEscape(method.Label),
		"{{CHECKOUT_BUTTON}}", bigBoldButton,
	)
	out := r.Replace(successHTML)

	// Auto-open the Bold checkout once the page loads, only when we have a URL.
	if checkoutURL != "" {
		autoOpen := fmt.Sprintf(`<script>setTimeout(function(){window.open(%q,'_blank');},800);</script></body>`, checkoutURL)
		out = strings.Replace(out, "</body>", autoOpen, 1)
	}
	return out
}
