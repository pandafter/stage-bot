package inscripcion

import (
	"bytes"
	"context"
	_ "embed"
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

// Plan describes a registration plan.
type Plan struct {
	ID         string // form value
	Label      string
	PriceCOP   int    // total price (per person)
	BoldLink   string // optional, for "tarjeta"
	Note       string
	IsPreventa bool
}

var Plans = []Plan{
	{ID: "preventa", Label: "Preventa individual / parejas", PriceCOP: 730000, BoldLink: "https://checkout.bold.co/payment/LNK_VFE303JBR9", IsPreventa: true},
	{ID: "normal", Label: "Tarifa estándar (sin descuento)", PriceCOP: 890000},
	{ID: "reserva", Label: "Solo reserva (asistir luego)", PriceCOP: 150000, BoldLink: "https://checkout.bold.co/payment/LNK_UQU9WHRQDT"},
}

// ReservaBoldLink is always the same — $150k via Bold for cup reservation.
const ReservaBoldLink = "https://checkout.bold.co/payment/LNK_UQU9WHRQDT"
const ReservaCOP = 150000

// CardSurchargePct is the extra cost for paying with credit card.
const CardSurchargePct = 5

// Available course dates (label shown to user). Match the values rendered in
// form.html.
var CourseDates = []string{
	"MAYO 9 y 10",
	"MAYO 23 y 24",
}

// PaymentMethods.
var PaymentMethods = []struct {
	ID    string
	Label string
}{
	{"bancolombia", "Bancolombia (transferencia)"},
	{"nequi", "Nequi (transferencia)"},
	{"bold", "Tarjeta de crédito (Bold) — recargo +5%"},
}

type Config struct {
	UploadsDir       string // directory for receipt uploads, e.g. ./uploads
	PublicURL        string // for absolute redirect after success
	TelegramBotToken string
	TelegramChatID   string
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

// Submit handles POSTed registration data.
func (h *Handler) Submit(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return errPage(c, fiber.StatusBadRequest, "Formulario inválido", err.Error())
	}

	rec, plan, validationErr := parseForm(form)
	if validationErr != nil {
		return errPage(c, fiber.StatusBadRequest, "Faltan datos obligatorios", validationErr.Error())
	}

	// Receipt upload required for transfer methods.
	requiresReceipt := rec.MetodoPago == "bancolombia" || rec.MetodoPago == "nequi"
	files := form.File["comprobante"]
	if requiresReceipt && len(files) == 0 {
		return errPage(c, fiber.StatusBadRequest, "Comprobante requerido",
			"Para Bancolombia/Nequi debes adjuntar el comprobante de la reserva ($150.000).")
	}

	rec.ID = newID()
	rec.Status = "pendiente"
	if requiresReceipt && len(files) > 0 {
		path, err := h.saveReceipt(rec.ID, files[0])
		if err != nil {
			h.logger.Error("save receipt", zap.Error(err))
			return errPage(c, fiber.StatusInternalServerError, "Error subiendo comprobante", err.Error())
		}
		rec.ComprobantePath = path
		rec.Status = "reserva confirmada, saldo pendiente"
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 10*time.Second)
	defer cancel()
	if err := h.repo.Insert(ctx, rec); err != nil {
		h.logger.Error("insert inscripcion", zap.Error(err))
		return errPage(c, fiber.StatusInternalServerError, "Error guardando inscripción", err.Error())
	}

	go h.notifyDirector(*rec, *plan)

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(renderSuccess(rec, plan))
}

// parseForm extracts and validates form fields.
func parseForm(form *multipart.Form) (*storage.InscripcionRecord, *Plan, error) {
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
		Plan:             get("plan"),
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
		return nil, nil, errors.New("email inválido")
	}
	if rec.NombrePiloto == "" {
		return nil, nil, errors.New("nombre del piloto es obligatorio")
	}
	if rec.NumeroDocumento == "" {
		return nil, nil, errors.New("número de documento es obligatorio")
	}
	if rec.Telefono == "" {
		return nil, nil, errors.New("teléfono es obligatorio")
	}
	if rec.EPS == "" || rec.GrupoSanguineo == "" {
		return nil, nil, errors.New("EPS y grupo sanguíneo son obligatorios")
	}
	if rec.FamiliarNombre == "" || rec.FamiliarTelefono == "" {
		return nil, nil, errors.New("contacto familiar de emergencia es obligatorio")
	}

	edadStr := get("edad")
	if edadStr == "" {
		return nil, nil, errors.New("edad es obligatoria")
	}
	edad, err := strconv.Atoi(edadStr)
	if err != nil || edad < 8 || edad > 90 {
		return nil, nil, errors.New("edad fuera de rango (8-90)")
	}
	rec.Edad = edad

	plan := findPlan(rec.Plan)
	if plan == nil {
		return nil, nil, errors.New("plan inválido")
	}

	monto := plan.PriceCOP
	if rec.MetodoPago == "bold" {
		monto = monto + (monto*CardSurchargePct)/100
	}
	rec.MontoCOP = monto

	if !validDate(rec.FechaCurso) {
		return nil, nil, errors.New("fecha de curso inválida")
	}
	if !validMethod(rec.MetodoPago) {
		return nil, nil, errors.New("método de pago inválido")
	}

	return rec, plan, nil
}

func findPlan(id string) *Plan {
	for i := range Plans {
		if Plans[i].ID == id {
			return &Plans[i]
		}
	}
	return nil
}

func validDate(d string) bool {
	for _, c := range CourseDates {
		if c == d {
			return true
		}
	}
	return false
}

func validMethod(m string) bool {
	for _, p := range PaymentMethods {
		if p.ID == m {
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

func (h *Handler) notifyDirector(rec storage.InscripcionRecord, plan Plan) {
	if h.cfg.TelegramBotToken == "" || h.cfg.TelegramChatID == "" {
		h.logger.Info("inscripcion received (no Telegram configured)",
			zap.String("id", rec.ID),
			zap.String("piloto", rec.NombrePiloto),
			zap.String("email", rec.Email),
			zap.String("plan", rec.Plan),
			zap.Int("monto", rec.MontoCOP))
		return
	}

	caption := buildTelegramMessage(rec, plan)

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

func buildTelegramMessage(rec storage.InscripcionRecord, plan Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🏁 *Nueva inscripción* — %s\n\n", tgEscape(plan.Label))
	fmt.Fprintf(&b, "*ID:* `%s`\n", tgEscape(rec.ID))
	fmt.Fprintf(&b, "*Estado:* %s\n", tgEscape(rec.Status))
	fmt.Fprintf(&b, "*Monto:* $%s COP\n", tgEscape(formatCOP(rec.MontoCOP)))
	fmt.Fprintf(&b, "*Pago:* %s\n", tgEscape(rec.MetodoPago))
	fmt.Fprintf(&b, "*Fecha curso:* %s\n\n", tgEscape(rec.FechaCurso))
	fmt.Fprintf(&b, "👤 *Piloto*\n")
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

func renderSuccess(rec *storage.InscripcionRecord, plan *Plan) string {
	nextStep := "Tu reserva quedó registrada. Pronto te contactaremos por WhatsApp para confirmar."
	if rec.MetodoPago == "bold" {
		nextStep = "Completa el pago de la reserva ($150.000) con Bold para confirmar tu cupo."
	}

	r := strings.NewReplacer(
		"{{ID}}", htmlEscape(rec.ID),
		"{{NOMBRE}}", htmlEscape(rec.NombrePiloto),
		"{{PLAN}}", htmlEscape(plan.Label),
		"{{MONTO}}", formatCOP(rec.MontoCOP),
		"{{FECHA}}", htmlEscape(rec.FechaCurso),
		"{{STATUS}}", htmlEscape(rec.Status),
		"{{EMAIL}}", htmlEscape(rec.Email),
		"{{NEXT_STEP}}", htmlEscape(nextStep),
	)
	out := r.Replace(successHTML)

	// If user picked Bold, auto-open the checkout in a new tab once the page loads.
	if rec.MetodoPago == "bold" {
		boldURL := ReservaBoldLink + "?ref=" + rec.ID
		autoOpen := fmt.Sprintf(`<script>setTimeout(function(){window.open(%q,'_blank');},800);</script></body>`, boldURL)
		out = strings.Replace(out, "</body>", autoOpen, 1)
	}
	return out
}
