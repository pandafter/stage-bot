# PROMETHEUS_PLAN.md — Scudería St4ge Web Platform

> "El kart no se construye en el taller. Se construye en el plano. Aquí está el plano."

**Proyecto:** Migración del bot de Instagram (Scudería St4ge) a una plataforma web profesional.
**Fecha:** 2026-04-28
**Arquitecto:** Prometheus
**Ejecutor objetivo:** Atlas
**Repositorio:** `D:/instagram-bot/` (renombre lógico: `scuderia-st4ge-web`, módulo Go conservado por compatibilidad).

---

## 1. VISIÓN GLOBAL

Convertir la base de código actual — un bot de Instagram con un sistema de inscripciones HTML server-rendered embebido — en una **SPA profesional Vue 3 + TypeScript** servida por un **backend Go (Fiber) limpio y mínimo**, conservando 100% de la lógica de negocio crítica:

- Captura de inscripciones para los cursos de kartismo de Scudería St4ge (Tocancipá, Colombia).
- Integración con **Bold** (pasarela de pago colombiana): creación de links de pago + webhook firmado con HMAC.
- Notificaciones a **Telegram** del director con foto del comprobante.
- Persistencia en **PostgreSQL** (pgx/v5).
- Despliegue en **Railway** vía Dockerfile multi-stage.

El sitio resultante es una **landing comercial + funnel de inscripción** que vive en un único dominio. El bot de Instagram, MCP, voice (ElevenLabs), AI (Claude), Google Sheets y queue worker quedan **completamente eliminados**.

### Lo que el usuario final percibe
1. Visita la URL pública → ve la landing oscura con identidad gold/red.
2. Hace clic en "Inscribirme" → SPA navega al funnel multi-step (sin recarga).
3. Llena datos del piloto, elige modalidad, elige método de pago.
4. Si paga digital → es redirigido a Bold checkout.
5. Bold confirma server-to-server por webhook → Telegram suena al director.
6. Usuario regresa al callback → ve estado en tiempo real (pagado / rechazado / verificando con polling).

---

## 2. PRINCIPIOS DE DISEÑO

1. **Identidad visual intacta.** Mismo `--bg:#090909`, `--gold:#f5c100`, `--red:#e63200`, `--text:#f0ede5`. Mismas fuentes (Barlow Condensed + DM Sans). Mismo `clip-path` angular. La migración es estructural, **no estética**.
2. **API-first.** El backend solo expone JSON + webhook + assets + SPA fallback. Ninguna ruta server-renderiza HTML excepto `index.html` del bundle Vue.
3. **Separación dura.** `cmd/server/` no importa nada del bot. `internal/api/` no conoce templates. El frontend no sabe nada de Bold.
4. **Embed o filesystem, nunca ambos.** Los assets pesados (videos, imágenes) viven en `frontend/public/` durante desarrollo y se sirven como archivos estáticos en producción (no `//go:embed`, son demasiado grandes para el binario).
5. **Concurrencia segura.** `pgxpool` con timeouts en cada handler. Notificaciones Telegram en goroutines no bloqueantes con su propio `context.Background()` y timeout interno.
6. **Type safety end-to-end.** Tipos compartidos: el backend define el contrato JSON, el frontend lo consume con interfaces TypeScript espejadas a mano (no codegen — es un proyecto pequeño, no vale la complejidad).
7. **Mobile-first.** El 80% del tráfico esperado entra desde Instagram in-app browser. El `IgBanner.vue` detecta IG y ofrece "abrir en navegador externo".
8. **Idempotencia en webhooks.** El webhook de Bold puede llegar dos veces; `UpdateStatus` debe ser tolerante.
9. **Cero dependencias innecesarias.** Vue 3, Vite, Vue Router, Pinia, axios. Sin Tailwind, sin UI kit, sin animation libs. CSS scoped o global propio.

---

## 3. ARQUITECTURA

```
                                  ┌──────────────────────────────┐
                                  │   Cliente (Móvil — IG WebView│
                                  │   o navegador desktop)        │
                                  └───────────────┬──────────────┘
                                                  │ HTTPS
                                                  ▼
              ┌──────────────────────────────────────────────────────────┐
              │              Railway Container (Go binario)               │
              │                                                           │
              │   Fiber v2  ─────────────────────────────────────────┐    │
              │      │                                                │    │
              │      ├─  GET  /health         ──► healthz (Railway)   │    │
              │      │                                                │    │
              │      ├─  GET  /api/config     ──► api.Config          │    │
              │      ├─  POST /api/inscripciones ──► api.Inscripciones│    │
              │      ├─  GET  /api/inscripciones/:id                  │    │
              │      ├─  POST /api/inscripciones/:id/comprobante      │    │
              │      │                                                │    │
              │      ├─  POST /webhook/bold   ──► api.BoldWebhook     │    │
              │      │         (HMAC verify, UpdateStatus, Telegram)  │    │
              │      │                                                │    │
              │      ├─  GET  /assets/*       ──► static FS (videos,  │    │
              │      │                              imágenes, logo)    │    │
              │      │                                                │    │
              │      └─  GET  /*              ──► serveSPA (Vue dist) │    │
              │                                       │                │    │
              │                                       ▼                │    │
              │                            embed.FS o filesystem       │    │
              │                            (frontend/dist/index.html)  │    │
              │                                                        │    │
              │   ┌──────────────┐    ┌────────────┐    ┌───────────┐ │    │
              │   │ pgxpool      │───►│ PostgreSQL │    │  Telegram │ │    │
              │   │ (DATABASE_URL│    │  (Railway) │    │   Bot API │ │    │
              │   └──────────────┘    └────────────┘    └─────▲─────┘ │    │
              │           ▲                                   │       │    │
              │           │                                   │       │    │
              │   ┌───────┴───────┐                  ┌────────┴─────┐ │    │
              │   │ Bold API      │                  │ goroutine    │ │    │
              │   │ (link create) │                  │ pool notif.  │ │    │
              │   └───────────────┘                  └──────────────┘ │    │
              └──────────────────────────────────────────────────────┘
```

### Flujo de datos: inscripción exitosa con tarjeta

```
Cliente (Vue)                Backend (Go)              Bold              Telegram
     │                            │                     │                    │
     │ POST /api/inscripciones    │                     │                    │
     ├───────────────────────────►│                     │                    │
     │                            │ INSERT inscripcion  │                    │
     │                            │ (status=pendiente)  │                    │
     │                            │                     │                    │
     │                            │ POST /online/link/v1│                    │
     │                            ├────────────────────►│                    │
     │                            │◄────────────────────┤ checkout_url       │
     │ 200 {id, checkout_url}     │                     │                    │
     │◄───────────────────────────┤                     │                    │
     │                            │ go notifyDirector ──┼───────────────────►│
     │                            │                     │                    │
     │ window.location=checkout_url                     │                    │
     │ ─────────────────────────────────────────────────►                    │
     │                            │                     │                    │
     │ ◄─────────  Bold UI ────────────────────────────►│                    │
     │                            │                     │                    │
     │                            │ POST /webhook/bold  │                    │
     │                            │◄────────────────────┤ (HMAC SHA-256)     │
     │                            │ verify + Update     │                    │
     │                            │ status=pagado       │                    │
     │                            │ go notifyConfirmed ─┼───────────────────►│
     │                            │                     │                    │
     │ GET /callback?ref=ID       │                     │                    │
     ├───────────────────────────►│                     │                    │
     │ (SPA renderiza CallbackView)                     │                    │
     │ poll GET /api/inscripciones/:id ──► status=pagado│                    │
     │ ✅ "Pago confirmado"        │                     │                    │
```

---

## 4. STACK TECNOLÓGICO

### Backend
| Componente | Tecnología | Justificación |
|---|---|---|
| Lenguaje | **Go 1.25** | Ya está. Compilación rápida, binario único, ideal para Railway. |
| HTTP framework | **Fiber v2** | Ya está. Rendimiento, compatible con `embed.FS`. |
| DB driver | **pgx/v5 (pgxpool)** | Migrar de `database/sql` a `pgxpool` por concurrencia y prepared statements nativos. |
| Logger | **zap** | Ya está. Production-grade. |
| Config | **godotenv** | Ya está. Solo en dev. En Railway las env vars son nativas. |

### Frontend
| Componente | Tecnología | Justificación |
|---|---|---|
| Framework | **Vue 3.5** | Composition API, `<script setup>`, reactivity transform. |
| Build tool | **Vite 6** | DX excelente, HMR instantáneo, build optimizado. |
| Lenguaje | **TypeScript 5.6** | Type safety en el contrato del API. |
| Router | **Vue Router 4** | Standard. SPA fallback en backend. |
| State | **Pinia 2** | Solo para el form multi-step (estado compartido entre 4 pasos). |
| HTTP | **axios 1.7** | Interceptores fáciles para errores globales. |
| Validación | **Manual + reactive refs** | El form tiene ~20 campos; no justifica VeeValidate o Zod. |
| Estilos | **CSS variables + scoped CSS** | Mantener identidad sin un framework. |

### Infraestructura
| Componente | Tecnología | Justificación |
|---|---|---|
| Hosting | **Railway** | Ya configurado. PostgreSQL plugin nativo. |
| Build | **Dockerfile multi-stage** | Stage 1 Node, Stage 2 Go, Stage 3 alpine. |
| DB | **PostgreSQL 16 (Railway plugin)** | Ya está. |
| TLS | **Railway-provided** | Auto-renovación. |

---

## 5. ESTRUCTURA DEL PROYECTO

```
D:/instagram-bot/                       (raíz del repo, mantenemos el path actual)
├── CLAUDE.md                           ← actualizar (ya no es bot)
├── PROMETHEUS_PLAN.md                  ← este archivo
├── README.md                           ← reescribir
├── Dockerfile                          ← reescribir multi-stage
├── railway.toml                        ← actualizar healthcheck
├── docker-compose.yml                  ← simplificar (postgres local + app)
├── go.mod                              ← módulo se mantiene (compatibilidad)
├── go.sum                              ← regenerar con `go mod tidy`
├── .env.example                        ← reescribir (solo vars necesarias)
├── .gitignore                          ← agregar `frontend/node_modules`, `frontend/dist`
├── Makefile                            ← targets nuevos: dev, build, build-frontend
│
├── cmd/
│   └── server/
│       └── main.go                     ← NUEVO entry point limpio
│
├── internal/
│   ├── api/
│   │   ├── inscripciones.go            ← NUEVO REST handlers
│   │   ├── bold_webhook.go             ← MOVIDO desde inscripcion/, conservar lógica HMAC
│   │   ├── bold_api.go                 ← MOVIDO desde inscripcion/
│   │   ├── config.go                   ← NUEVO GET /api/config
│   │   ├── health.go                   ← NUEVO GET /health
│   │   ├── telegram.go                 ← MOVIDO de inscripcion/handler.go (notifyDirector + helpers)
│   │   └── types.go                    ← NUEVO request/response DTOs
│   │
│   ├── storage/
│   │   ├── postgres.go                 ← MIGRAR a pgxpool
│   │   └── inscripciones.go            ← actualizar a pgxpool, conservar struct
│   │
│   ├── config/
│   │   └── config.go                   ← REESCRIBIR (solo vars necesarias)
│   │
│   ├── server/
│   │   └── server.go                   ← REESCRIBIR (Fiber app limpia, CORS, SPA fallback)
│   │
│   └── spa/
│       └── spa.go                      ← NUEVO: //go:embed frontend/dist/* + serveSPA
│
├── migrations/
│   └── 001_inscripciones.sql           ← conservar / verificar
│
├── frontend/                           ← NUEVO directorio Vue 3
│   ├── package.json
│   ├── package-lock.json
│   ├── tsconfig.json
│   ├── tsconfig.node.json
│   ├── vite.config.ts
│   ├── index.html                      ← shell HTML mínimo
│   ├── public/
│   │   └── assets/                     ← logo, imágenes, videos (movidos desde internal/inscripcion/assets/)
│   │       ├── logo.jpg
│   │       ├── AndresMelo.png
│   │       ├── EquipoMecanico.png
│   │       ├── MapaKartodromo.png
│   │       ├── SantiGutierrez.png
│   │       ├── bancolombiaLogo.png
│   │       ├── nequiLogo.webp
│   │       ├── imagen1.png
│   │       ├── video1.mp4
│   │       ├── video2.mp4
│   │       ├── video3.mp4
│   │       └── video4.mp4
│   └── src/
│       ├── main.ts                     ← bootstrap: app, router, pinia, axios
│       ├── App.vue                     ← layout root con <RouterView/>
│       ├── env.d.ts
│       ├── router/
│       │   └── index.ts                ← rutas: /, /inscripcion, /success, /callback
│       ├── stores/
│       │   └── inscripcion.ts          ← Pinia: form state, step actual, errores, submit
│       ├── services/
│       │   ├── api.ts                  ← axios instance + interceptores
│       │   └── inscripciones.ts        ← funciones tipadas (createInscripcion, getStatus, ...)
│       ├── types/
│       │   └── api.ts                  ← interfaces TS espejo del backend
│       ├── composables/
│       │   ├── useInstagramBrowser.ts  ← detectar IG webview
│       │   └── useCountdown.ts         ← countdown cierre de inscripciones
│       ├── assets/
│       │   └── styles/
│       │       ├── main.css            ← variables :root, reset, fuentes
│       │       └── components.css      ← clases compartidas (.btn-gold, .clip-card, ...)
│       ├── views/
│       │   ├── HomeView.vue            ← landing completa
│       │   ├── InscripcionView.vue     ← shell del funnel multi-step
│       │   ├── SuccessView.vue         ← confirmación post-submit (modo manual / digital)
│       │   └── CallbackView.vue        ← post-Bold con polling
│       └── components/
│           ├── ui/
│           │   ├── AppNav.vue
│           │   ├── AppFooter.vue
│           │   ├── BoldButton.vue
│           │   ├── CountdownTimer.vue
│           │   ├── IgBanner.vue
│           │   ├── ClipCard.vue        ← wrapper con clip-path angular
│           │   └── BaseInput.vue       ← input con label flotante consistente
│           ├── landing/
│           │   ├── HeroSection.vue
│           │   ├── StatsSection.vue
│           │   ├── ProgramSection.vue
│           │   ├── InstructoresSection.vue
│           │   ├── GallerySection.vue
│           │   ├── FaqSection.vue
│           │   └── CtaSection.vue
│           └── inscripcion/
│               ├── StepIndicator.vue
│               ├── PersonalDataStep.vue
│               ├── CourseStep.vue
│               ├── PaymentStep.vue
│               ├── ConfirmationStep.vue
│               └── FileUpload.vue
```

### A ELIMINAR (lista exhaustiva)

```
cmd/bot/                                ← ENTERO
cmd/media-upload/                       ← ENTERO
internal/admin/                         ← ENTERO
internal/ai/                            ← ENTERO
internal/domain/                        ← ENTERO
internal/knowledge/                     ← ENTERO
internal/mcp/                           ← ENTERO
internal/messenger/                     ← ENTERO
internal/queue/                         ← ENTERO
internal/voice/                         ← ENTERO
internal/webhook/                       ← ENTERO
internal/inscripcion/                   ← ENTERO (después de mover lo útil a internal/api/)
internal/storage/leads.go
internal/storage/conversation.go
internal/storage/media.go
internal/storage/audio.go               (si existe)
docs/                                   ← revisar si guarda algo del bot
data/                                   ← si era SQLite del bot
dist/                                   ← antiguo
bot.exe                                 ← binario obsoleto
PLAN.md                                 ← obsoleto
```

---

## 6. CONTRATO API

### Tipos compartidos (TypeScript ↔ Go)

```typescript
// frontend/src/types/api.ts

export type Modalidad = "reserva" | "completo";
export type MetodoPago = "tarjeta" | "nequi" | "bancolombia" | "pse" | "transferencia";
export type FechaCurso = "MAYO 9 y 10" | "MAYO 23 y 24";
export type Status =
  | "pendiente"
  | "comprobante recibido, en validación"
  | "pagado"
  | "pago rechazado";

export interface ConfigResponse {
  modalidades: { id: Modalidad; label: string; price_cop: number }[];
  metodos: { id: MetodoPago; label: string }[];
  fechas: FechaCurso[];
  card_surcharge_pct: number;     // 5
  reserva_cop: number;             // 150000
}

export interface CreateInscripcionRequest {
  email: string;
  nombre_piloto: string;
  edad: number;
  tipo_documento: string;
  numero_documento: string;
  telefono: string;
  ciudad: string;
  eps: string;
  grupo_sanguineo: string;
  familiar_nombre: string;
  familiar_telefono: string;
  instagram_user?: string;
  modalidad: Modalidad;
  metodo_pago: MetodoPago;
  fecha_curso: FechaCurso;
}

export interface CreateInscripcionResponse {
  id: string;                     // INS-xxxxxxxx
  status: Status;
  monto_cop: number;
  checkout_url?: string;          // presente si metodo es digital
  requires_comprobante: boolean;  // true si transferencia
}

export interface InscripcionStatusResponse {
  id: string;
  status: Status;
  monto_cop: number;
  fecha_curso: FechaCurso;
  metodo_pago: MetodoPago;
  modalidad: Modalidad;
  nombre_piloto: string;
}
```

### Endpoints

| Método | Ruta | Auth | Body | Response |
|---|---|---|---|---|
| GET | `/health` | — | — | `{status:"ok", time}` |
| GET | `/api/config` | — | — | `ConfigResponse` |
| POST | `/api/inscripciones` | — | `CreateInscripcionRequest` (JSON) | `CreateInscripcionResponse` |
| GET | `/api/inscripciones/:id` | — | — | `InscripcionStatusResponse` |
| POST | `/api/inscripciones/:id/comprobante` | — | `multipart/form-data` (campo `file`) | `{ok:true}` |
| POST | `/webhook/bold` | HMAC header `x-bold-signature` | Bold event | `{ok:true}` |
| GET | `/assets/*` | — | — | bytes (image/video) |
| GET | `/*` | — | — | `index.html` (SPA fallback) |

### Reglas de negocio (preservar exactamente del handler.go actual)

1. **Cálculo de monto:** `monto = modalidad.price_cop`. Si `metodo_pago === "tarjeta"`, sumar `5%` (`CardSurchargePct`).
2. **Validaciones:** edad `[8,90]`, email regex, todos los campos obligatorios excepto `ciudad` e `instagram_user`.
3. **Status inicial:**
   - Digital → `pendiente`
   - Transferencia con comprobante → `comprobante recibido, en validación`
4. **ID format:** `INS-` + 8 random bytes hex (16 chars).
5. **Bold callback URL:** `${PUBLIC_URL}/callback?ref=${id}` (nota: la SPA route, NO `/inscripcion/callback`).
6. **Webhook Bold:** verificar HMAC SHA-256 con `BOLD_WEBHOOK_SECRET`, eventos `SALE_APPROVED` → `pagado`, `SALE_REJECTED` → `pago rechazado`. Idempotente.
7. **Telegram notify:** sola goroutine por evento, timeout 10s, foto si hay comprobante, texto markdown si no.

---

## 7. MODELO DE DATOS

Schema actual se conserva. Verificar `migrations/001_inscripciones.sql`:

```sql
CREATE TABLE IF NOT EXISTS inscripciones (
  id                 TEXT PRIMARY KEY,
  email              TEXT NOT NULL,
  metodo_pago        TEXT NOT NULL,
  fecha_curso        TEXT NOT NULL,
  plan               TEXT NOT NULL,
  monto_cop          INTEGER NOT NULL,
  nombre_piloto      TEXT NOT NULL,
  edad               INTEGER NOT NULL,
  tipo_documento     TEXT NOT NULL,
  numero_documento   TEXT NOT NULL,
  telefono           TEXT NOT NULL,
  ciudad             TEXT,
  eps                TEXT NOT NULL,
  grupo_sanguineo    TEXT NOT NULL,
  familiar_nombre    TEXT NOT NULL,
  familiar_telefono  TEXT NOT NULL,
  instagram_user     TEXT,
  comprobante_path   TEXT,
  status             TEXT NOT NULL DEFAULT 'pendiente',
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inscripciones_email ON inscripciones(email);
CREATE INDEX IF NOT EXISTS idx_inscripciones_status ON inscripciones(status);
CREATE INDEX IF NOT EXISTS idx_inscripciones_created_at ON inscripciones(created_at DESC);
```

Tablas a **eliminar** (si existen): `leads`, `conversations`, `messages`, `media`, `audio_store`, `queue_jobs`.

---

## 8. VARIABLES DE ENTORNO

### `.env.example` (nuevo, mínimo)

```bash
# Server
PORT=3000
ENV=development
PUBLIC_URL=http://localhost:3000

# Database
DATABASE_URL=postgres://postgres:postgres@localhost:5432/scuderia?sslmode=disable

# Bold (pasarela de pago)
BOLD_API_KEY=
BOLD_WEBHOOK_SECRET=

# Telegram (notificaciones al director)
TELEGRAM_BOT_TOKEN=
TELEGRAM_CHAT_ID=

# Storage
UPLOADS_DIR=./uploads

# Admin (solo para endpoints de debug — opcional)
ADMIN_TOKEN=
```

### Railway (production)

```
DATABASE_URL          ← provisto por plugin Postgres
PUBLIC_URL            ← https://${RAILWAY_PUBLIC_DOMAIN}
PORT                  ← inyectado por Railway
BOLD_API_KEY          ← secret
BOLD_WEBHOOK_SECRET   ← secret
TELEGRAM_BOT_TOKEN    ← secret
TELEGRAM_CHAT_ID      ← secret
ENV=production
UPLOADS_DIR=/data/uploads   ← volumen persistente Railway
```

---

# FASES DE EJECUCIÓN

> Cada fase es **autocontenida**, **compila**, y **es verificable** antes de pasar a la siguiente. Atlas debe completar la fase, ejecutar el "Entregable" y marcar como hecho antes de continuar.

---

## FASE 1 — LIMPIEZA DEL BACKEND
### "Antes de construir el templo, despejar el terreno."

**Objetivo:** Eliminar todo el código del bot y dejar el backend Go compilando con solo lo necesario para las inscripciones, pero todavía con la estructura actual de `internal/inscripcion/` (la reorganizaremos en Fase 2).
**Duración estimada:** 3-4 horas
**Dependencias:** Ninguna. Crear branch `feat/web-platform`.

### Tareas

1. **Backup defensivo**
   - [ ] `git checkout -b feat/web-platform`
   - [ ] `git tag pre-vue-migration` en `master` para poder volver.

2. **Eliminar directorios completos del bot**
   - [ ] `rm -rf cmd/bot/`
   - [ ] `rm -rf cmd/media-upload/`
   - [ ] `rm -rf internal/admin/`
   - [ ] `rm -rf internal/ai/`
   - [ ] `rm -rf internal/domain/`
   - [ ] `rm -rf internal/knowledge/`
   - [ ] `rm -rf internal/mcp/`
   - [ ] `rm -rf internal/messenger/`
   - [ ] `rm -rf internal/queue/`
   - [ ] `rm -rf internal/voice/`
   - [ ] `rm -rf internal/webhook/`

3. **Eliminar archivos sueltos**
   - [ ] `rm internal/storage/leads.go`
   - [ ] `rm internal/storage/conversation.go`
   - [ ] `rm internal/storage/media.go`
   - [ ] Si existen: `rm internal/storage/audio.go`, `rm internal/storage/queue.go`.
   - [ ] `rm bot.exe` (si está committed)
   - [ ] `rm PLAN.md` (obsoleto)
   - [ ] `rm -rf data/ dist/` (si solo eran del bot)

4. **Limpiar `internal/config/config.go`**
   - [ ] Eliminar campos: `AppID, AppSecret, PageAccessToken, InstagramAccountID, WebhookVerifyToken, AnthropicAPIKey, ElevenLabsAPIKey, ElevenLabsVoiceID, OpenAIAPIKey, GoogleSheetID, TestSenderID, CommentTriggerKeyword, GitHubToken, GitHubRepo, MCPSecret, BotPrompt, SystemPrompt, WorkerConcurrency, WorkerMaxAttempts, ClaudePriceInputPerMTok, ClaudePriceOutputPerMTok, SheetsWebhookURL, SheetsSharedToken, BotEnabled`.
   - [ ] Conservar: `Port, Env, PublicURL, DatabaseURL, AdminToken, UploadsDir, TelegramBotToken, TelegramChatID, BoldWebhookSecret, BoldAPIKey`.
   - [ ] Ajustar `validate()` para que solo `DATABASE_URL` sea required.

5. **Reescribir `internal/server/server.go`**
   - [ ] Eliminar imports de `admin`, `domain`, `mcp`, `queue`, `webhook`.
   - [ ] Eliminar struct field `Dependencies.Messenger, AI, Voice, AudioStore, Leads, Conversation, Queue, WebhookHandler`.
   - [ ] Conservar solo `Inscripciones *storage.InscripcionesRepo` y `DB *sql.DB` (por ahora).
   - [ ] Eliminar todas las rutas excepto `/health` y las de `inscripcion`.
   - [ ] Eliminar bloque MCP y bloque admin.
   - [ ] Conservar el bloque `if deps.Inscripciones != nil { ... }` tal cual.

6. **Crear nuevo entry point `cmd/server/main.go`**
   ```go
   package main

   import (
       "database/sql"
       "github.com/joho/godotenv"
       "go.uber.org/zap"
       _ "github.com/jackc/pgx/v5/stdlib"

       "github.com/kart-academy/instagram-bot/internal/config"
       "github.com/kart-academy/instagram-bot/internal/server"
       "github.com/kart-academy/instagram-bot/internal/storage"
   )

   func main() {
       _ = godotenv.Load()
       logger, _ := zap.NewProduction()
       defer logger.Sync()

       cfg, err := config.Load()
       if err != nil { logger.Fatal("config", zap.Error(err)) }

       db, err := sql.Open("pgx", cfg.DatabaseURL)
       if err != nil { logger.Fatal("db open", zap.Error(err)) }
       defer db.Close()
       if err := db.Ping(); err != nil { logger.Fatal("db ping", zap.Error(err)) }

       inscRepo := storage.NewInscripcionesRepo(&storage.DB{...})

       deps := server.Dependencies{
           Inscripciones: inscRepo,
           DB: db,
       }
       srv := server.New(cfg, deps, logger)
       if err := srv.Start(); err != nil {
           logger.Fatal("server", zap.Error(err))
       }
   }
   ```
   - [ ] Adaptar al constructor real de `storage.NewInscripcionesRepo`.
   - [ ] Eliminar `cmd/bot/main.go` ya borrado.

7. **Limpiar `MediaStorage`**
   - [ ] Si `internal/storage/media.go` se eliminó, también eliminar la llamada `storage.NewMediaStorage(...)` en `server.go`.
   - [ ] El handler de inscripciones usa `media` para servir assets desde DB. Por ahora, comentar/eliminar `ServeMedia` y cambiar a servir solo desde embed (lo arreglaremos en Fase 2).

8. **`go mod tidy` + compilación**
   - [ ] `go mod tidy`
   - [ ] Eliminar dependencias huérfanas del `go.mod` (si quedan).
   - [ ] `go build ./cmd/server/`
   - [ ] Resolver cualquier import roto.

### Detalle Crítico

El `internal/inscripcion/handler.go` referencia `h.media.Get(...)`. Cuando elimines `media.go`, tienes dos opciones:
- **A:** stub un `MediaStorage` vacío que siempre retorna `nil` → fallback a embed.
- **B:** Eliminar `ServeMedia` y rutas `/inscripcion/assets/:key`, dejar solo los handlers individuales (`ServeLogo`, `ServeBancolombiaLogo`, `ServeNequiLogo`).

**Recomendación:** opción B. Es más limpio. Las imágenes pesadas (videos, AndresMelo.png) se moverán al frontend en Fase 2.

### Entregable

- [ ] `go build ./cmd/server/` compila sin errores.
- [ ] `./server` arranca, conecta a Postgres local, expone `/health`, `/inscripcion`, `/inscripcion/callback`, `/webhook/bold`.
- [ ] `curl http://localhost:3000/health` → `{"status":"ok",...}`.
- [ ] `curl http://localhost:3000/inscripcion` → renderiza el form HTML viejo (todavía lo usamos como "modo legacy" hasta Fase 4).
- [ ] El árbol del repo NO contiene ningún directorio `ai/`, `mcp/`, `voice/`, etc.

---

## FASE 2 — REORGANIZACIÓN A `internal/api/`
### "Cada cosa en su sitio. Sólo entonces se mueve rápido."

**Objetivo:** Refactorizar el handler monolítico de `internal/inscripcion/` a un paquete `internal/api/` orientado a JSON, dejando el HTML embebido temporalmente como fallback `/legacy`. Migrar storage a `pgxpool`.
**Duración estimada:** 4-5 horas
**Dependencias:** Fase 1 completa.

### Tareas

1. **Crear estructura `internal/api/`**
   - [ ] `mkdir internal/api/`
   - [ ] Crear `types.go` con los DTOs JSON (mirror del contrato en sección 6).
   - [ ] Crear `health.go` con `func Health(c *fiber.Ctx) error`.
   - [ ] Crear `config.go` con `func GetConfig(c *fiber.Ctx) error` que retorna modalidades, métodos, fechas, surcharge.
   - [ ] Crear `inscripciones.go`:
     - `Handler struct { repo *storage.InscripcionesRepo, cfg Config, logger, telegram *TelegramClient, bold *BoldClient }`
     - `func (h *Handler) Create(c *fiber.Ctx)` — recibe JSON, valida, inserta, llama Bold si digital.
     - `func (h *Handler) Get(c *fiber.Ctx)` — `:id` → JSON status.
     - `func (h *Handler) UploadComprobante(c *fiber.Ctx)` — multipart, solo si transferencia.
   - [ ] Mover `bold_api.go` → `internal/api/bold_api.go` (renombrar Handler receiver a `BoldClient`).
   - [ ] Mover `bold_webhook.go` → `internal/api/bold_webhook.go`. Conservar lógica HMAC. La función pasa a usar la goroutine de Telegram extraída.
   - [ ] Crear `internal/api/telegram.go`: extraer `tgSendMessage`, `tgSendPhoto`, `tgEscape`, `formatCOP`, `notifyDirector`, `buildTelegramMessage` desde `inscripcion/handler.go`. Wrap en `type TelegramClient struct {...}`.
   - [ ] Crear `internal/api/validation.go`: extraer `validEmail`, `validDate`, helpers de parseo.

2. **Migrar storage a pgxpool**
   - [ ] `go get github.com/jackc/pgx/v5/pgxpool` (ya está como indirect, hacerlo direct).
   - [ ] Reescribir `internal/storage/postgres.go`:
     ```go
     type DB struct { Pool *pgxpool.Pool }
     func New(ctx context.Context, dsn string) (*DB, error) { ... }
     func (db *DB) Close() { db.Pool.Close() }
     ```
   - [ ] Reescribir `internal/storage/inscripciones.go` para usar `*pgxpool.Pool` en vez de `*sql.DB`.
     - `Insert(ctx, rec) error` con `pool.Exec(ctx, ...)`.
     - `UpdateStatus(ctx, id, status) error`.
     - `GetByID(ctx, id) (*InscripcionRecord, error)` usando `pool.QueryRow(ctx, ...).Scan(...)`. Manejo de `pgx.ErrNoRows`.
   - [ ] La struct `InscripcionRecord` se mantiene idéntica.

3. **Reescribir `cmd/server/main.go` con pgxpool**
   ```go
   ctx := context.Background()
   db, err := storage.New(ctx, cfg.DatabaseURL)
   inscRepo := storage.NewInscripcionesRepo(db)
   telegram := api.NewTelegramClient(cfg.TelegramBotToken, cfg.TelegramChatID, logger)
   bold := api.NewBoldClient(cfg.BoldAPIKey, cfg.PublicURL, logger)
   apiHandler := api.NewHandler(inscRepo, cfg, telegram, bold, logger)
   ```

4. **Reescribir `internal/server/server.go`**
   - [ ] Routing limpio:
     ```go
     app.Get("/health", api.Health)
     app.Get("/api/config", apiHandler.GetConfig)
     app.Post("/api/inscripciones", apiHandler.Create)
     app.Get("/api/inscripciones/:id", apiHandler.Get)
     app.Post("/api/inscripciones/:id/comprobante", apiHandler.UploadComprobante)
     app.Post("/webhook/bold", apiHandler.BoldWebhook)
     ```
   - [ ] CORS middleware (`github.com/gofiber/fiber/v2/middleware/cors`):
     - En dev: `AllowOrigins: "http://localhost:5173"` (Vite default).
     - En prod: `AllowOrigins: "*"` (mismo origen, no aplica realmente, pero seguro).
   - [ ] **Ruta legacy temporal**: mantener `/legacy/inscripcion` sirviendo el HTML viejo hasta Fase 4. Usa el handler viejo internamente. Esto es solo para no romper tráfico durante el desarrollo.
   - [ ] Aún no hay SPA fallback (Fase 6).

5. **Mover assets a `frontend/public/assets/`**
   - [ ] `mkdir -p frontend/public/assets/`
   - [ ] `mv internal/inscripcion/assets/* frontend/public/assets/`
   - [ ] Eliminar `//go:embed assets/*` del código Go viejo.
   - [ ] Eliminar `internal/inscripcion/` ENTERO (las plantillas viejas se irán a Fase 4 tras migrar a Vue).
   - [ ] **Excepción:** si en Fase 4 todavía no terminamos la migración del form, conservar `legacy_form.html` en `internal/api/legacy/` para servir bajo `/legacy/inscripcion`.

6. **Validación contractual**
   - [ ] El handler `Create` debe retornar exactamente la shape `CreateInscripcionResponse`.
   - [ ] El handler `BoldWebhook` debe ser idempotente: si `status` ya es `pagado` y llega otro `SALE_APPROVED`, no envía Telegram dos veces (consultar status antes de update).

### Detalle Crítico

```go
// internal/api/inscripciones.go — esqueleto del Create
func (h *Handler) Create(c *fiber.Ctx) error {
    var req types.CreateInscripcionRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
    }
    rec, err := req.ToRecord()  // valida y convierte
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"error": err.Error()})
    }
    rec.ID = newID()
    rec.Status = "pendiente"
    rec.MontoCOP = computeMonto(rec)

    ctx, cancel := context.WithTimeout(c.UserContext(), 15*time.Second)
    defer cancel()
    if err := h.repo.Insert(ctx, rec); err != nil {
        h.logger.Error("insert", zap.Error(err))
        return c.Status(500).JSON(fiber.Map{"error": "db error"})
    }

    var checkoutURL string
    if methodIsDigital(rec.MetodoPago) {
        url, err := h.bold.CreateLink(ctx, ...)
        if err != nil {
            h.logger.Error("bold", zap.Error(err))
        } else {
            checkoutURL = url
        }
    }

    go h.telegram.NotifyNewInscripcion(*rec, checkoutURL)

    return c.JSON(types.CreateInscripcionResponse{
        ID: rec.ID,
        Status: rec.Status,
        MontoCOP: rec.MontoCOP,
        CheckoutURL: checkoutURL,
        RequiresComprobante: rec.MetodoPago == "transferencia",
    })
}
```

### Entregable

- [ ] `go build ./cmd/server/` compila.
- [ ] `curl -X POST http://localhost:3000/api/inscripciones -H "Content-Type: application/json" -d '{"email":"test@test.com",...}'` → 200 con JSON válido.
- [ ] `curl http://localhost:3000/api/config` → JSON con modalidades.
- [ ] `curl http://localhost:3000/api/inscripciones/INS-xxx` → status del registro.
- [ ] Webhook Bold sigue funcionando (probar con un POST firmado de prueba).
- [ ] Telegram envía notificación al insertar.
- [ ] El módulo `internal/inscripcion/` ha sido **eliminado**.
- [ ] El módulo `internal/api/` contiene 8 archivos: `types.go, health.go, config.go, inscripciones.go, bold_api.go, bold_webhook.go, telegram.go, validation.go`.

---

## FASE 3 — SCAFFOLDING DEL FRONTEND VUE 3
### "El esqueleto antes que la piel."

**Objetivo:** Crear el proyecto Vue 3 + Vite + TS, configurar router, Pinia, axios, definir el contrato de tipos, y dejar una landing "Hola mundo" funcional consumiendo `/api/config`.
**Duración estimada:** 3-4 horas
**Dependencias:** Fase 2 completa (API JSON funcional).

### Tareas

1. **Crear proyecto Vite**
   - [ ] `cd D:/instagram-bot && npm create vite@latest frontend -- --template vue-ts`
   - [ ] `cd frontend && npm install`
   - [ ] Verificar `npm run dev` arranca en `:5173`.

2. **Instalar dependencias**
   - [ ] `npm install vue-router@4 pinia@2 axios@1.7`
   - [ ] `npm install -D @types/node`

3. **Configurar `vite.config.ts`**
   ```ts
   import { defineConfig } from 'vite'
   import vue from '@vitejs/plugin-vue'
   import path from 'path'

   export default defineConfig({
     plugins: [vue()],
     resolve: { alias: { '@': path.resolve(__dirname, './src') } },
     server: {
       port: 5173,
       proxy: {
         '/api': 'http://localhost:3000',
         '/webhook': 'http://localhost:3000',
         '/assets': 'http://localhost:3000',
       },
     },
     build: {
       outDir: 'dist',
       emptyOutDir: true,
     },
   })
   ```

4. **Configurar `tsconfig.json`** con paths `@/*` → `src/*`.

5. **Crear `src/main.ts`**
   ```ts
   import { createApp } from 'vue'
   import { createPinia } from 'pinia'
   import App from './App.vue'
   import router from './router'
   import './assets/styles/main.css'

   const app = createApp(App)
   app.use(createPinia())
   app.use(router)
   app.mount('#app')
   ```

6. **Crear router (`src/router/index.ts`)**
   ```ts
   import { createRouter, createWebHistory } from 'vue-router'

   const router = createRouter({
     history: createWebHistory(),
     routes: [
       { path: '/', component: () => import('@/views/HomeView.vue') },
       { path: '/inscripcion', component: () => import('@/views/InscripcionView.vue') },
       { path: '/success', component: () => import('@/views/SuccessView.vue') },
       { path: '/callback', component: () => import('@/views/CallbackView.vue') },
       { path: '/:pathMatch(.*)*', redirect: '/' },
     ],
     scrollBehavior() { return { top: 0 } },
   })
   export default router
   ```

7. **Crear servicios API**
   - [ ] `src/services/api.ts`:
     ```ts
     import axios from 'axios'
     export const api = axios.create({ baseURL: '/api', timeout: 15000 })
     api.interceptors.response.use(r => r, err => {
       console.error('API error:', err.response?.data || err.message)
       return Promise.reject(err)
     })
     ```
   - [ ] `src/services/inscripciones.ts`:
     ```ts
     import { api } from './api'
     import type { ConfigResponse, CreateInscripcionRequest, CreateInscripcionResponse, InscripcionStatusResponse } from '@/types/api'

     export const fetchConfig = () => api.get<ConfigResponse>('/config').then(r => r.data)
     export const createInscripcion = (req: CreateInscripcionRequest) =>
       api.post<CreateInscripcionResponse>('/inscripciones', req).then(r => r.data)
     export const getInscripcion = (id: string) =>
       api.get<InscripcionStatusResponse>(`/inscripciones/${id}`).then(r => r.data)
     export const uploadComprobante = (id: string, file: File) => {
       const fd = new FormData()
       fd.append('file', file)
       return api.post(`/inscripciones/${id}/comprobante`, fd).then(r => r.data)
     }
     ```

8. **Crear types compartidos** (`src/types/api.ts`) según sección 6.

9. **Crear Pinia store** (`src/stores/inscripcion.ts`)
   ```ts
   import { defineStore } from 'pinia'
   import type { CreateInscripcionRequest } from '@/types/api'

   export const useInscripcionStore = defineStore('inscripcion', {
     state: () => ({
       step: 1 as 1 | 2 | 3 | 4,
       form: <Partial<CreateInscripcionRequest>>{},
       errors: <Record<string,string>>{},
       submitting: false,
       result: null as null | { id: string; checkout_url?: string },
     }),
     actions: {
       next() { if (this.step < 4) this.step++ },
       prev() { if (this.step > 1) this.step-- },
       updateField<K extends keyof CreateInscripcionRequest>(k: K, v: CreateInscripcionRequest[K]) {
         this.form[k] = v
       },
       reset() { this.step = 1; this.form = {}; this.errors = {}; this.result = null },
     },
   })
   ```

10. **Crear `src/App.vue` mínimo**
    ```vue
    <template>
      <RouterView />
    </template>
    ```

11. **Crear `src/views/HomeView.vue` placeholder**
    ```vue
    <script setup lang="ts">
    import { onMounted, ref } from 'vue'
    import { fetchConfig } from '@/services/inscripciones'
    import type { ConfigResponse } from '@/types/api'

    const config = ref<ConfigResponse | null>(null)
    onMounted(async () => { config.value = await fetchConfig() })
    </script>
    <template>
      <main>
        <h1>Scudería St4ge</h1>
        <pre v-if="config">{{ config }}</pre>
      </main>
    </template>
    ```

12. **Crear styles base** (`src/assets/styles/main.css`)
    - Variables CSS (`:root { --bg, --gold, --red, --text, --border }`).
    - `@import url('https://fonts.googleapis.com/css2?family=Barlow+Condensed:wght@400;600;700;800&family=DM+Sans:wght@400;500;700&display=swap');`
    - Reset `* { box-sizing: border-box; margin: 0; }`.
    - `body { background: var(--bg); color: var(--text); font-family: 'DM Sans', sans-serif; }`.

### Detalle Crítico

**Proxy de Vite es esencial.** Durante desarrollo, el frontend corre en `:5173` y el backend en `:3000`. Sin el proxy, las llamadas `/api/*` desde el frontend romperían CORS. El backend ya tiene CORS configurado para `:5173` (Fase 2), pero el proxy es más limpio y simula producción (mismo origen).

### Entregable

- [ ] `cd frontend && npm run dev` arranca y muestra HomeView con el JSON de `/api/config`.
- [ ] Navegar a `/inscripcion`, `/success`, `/callback` no rompe (las views pueden estar vacías).
- [ ] `npm run build` produce `frontend/dist/` con `index.html` + `assets/`.
- [ ] No hay errores de TypeScript.

---

## FASE 4 — LANDING PAGE (HomeView)
### "La cara del piloto que verán antes de subirse."

**Objetivo:** Convertir `landing.html` (737 líneas de HTML+CSS server-side) a una jerarquía de componentes Vue, conservando 100% de la identidad visual y los videos/imágenes ahora servidos desde `/assets/`.
**Duración estimada:** 6-8 horas
**Dependencias:** Fase 3 completa.

### Tareas

1. **Lectura forense del HTML legacy**
   - [ ] Atlas debe leer `internal/inscripcion/landing.html` (si todavía existe en `legacy/`) o reconstruir desde el git history (`git show pre-vue-migration:internal/inscripcion/landing.html`).
   - [ ] Identificar las **secciones** del landing: hero, stats, programa, instructores, galería de videos, FAQ, CTA, footer.

2. **Extraer CSS global** → `src/assets/styles/main.css`
   - [ ] Variables (`:root`).
   - [ ] Tipografías (`@import` de Google Fonts).
   - [ ] Clases utilitarias compartidas: `.clip-card` (clip-path), `.btn-gold`, `.btn-outline`, `.section`, `.container`.

3. **Crear componentes UI base** (`src/components/ui/`)
   - [ ] `AppNav.vue` — header con logo + nav links + CTA "Inscríbete".
   - [ ] `AppFooter.vue` — copyright + redes.
   - [ ] `ClipCard.vue` — wrapper con `clip-path: polygon(10px 0%, 100% 0%, calc(100% - 10px) 100%, 0% 100%)` y `<slot/>`.
   - [ ] `IgBanner.vue` — detecta `navigator.userAgent.includes('Instagram')`, ofrece "Abrir en navegador" (link al mismo URL pero forzando externa).
   - [ ] `CountdownTimer.vue` — countdown a `MAYO 9 y 10`. Props: `:targetDate`. Emite cuando termina.

4. **Crear secciones del landing** (`src/components/landing/`)
   - [ ] `HeroSection.vue` — title, subtitle, CTA → `/inscripcion`. Background video o imagen.
   - [ ] `StatsSection.vue` — grid de números (km recorridos, pilotos formados, etc.). Animación de conteo on-scroll.
   - [ ] `ProgramSection.vue` — bullets de qué incluye el curso, precios (consume `useStore` para el config si quieres dinamizar).
   - [ ] `InstructoresSection.vue` — cards con AndresMelo.png, SantiGutierrez.png, etc.
   - [ ] `GallerySection.vue` — video1-4.mp4 en grid responsivo, autoplay muted loop.
   - [ ] `FaqSection.vue` — accordion con preguntas frecuentes.
   - [ ] `CtaSection.vue` — última llamada a la acción + CountdownTimer.

5. **Ensamblar `HomeView.vue`**
   ```vue
   <script setup lang="ts">
   import IgBanner from '@/components/ui/IgBanner.vue'
   import AppNav from '@/components/ui/AppNav.vue'
   import HeroSection from '@/components/landing/HeroSection.vue'
   /* ... resto */
   </script>
   <template>
     <IgBanner />
     <AppNav />
     <main>
       <HeroSection />
       <StatsSection />
       <ProgramSection />
       <InstructoresSection />
       <GallerySection />
       <FaqSection />
       <CtaSection />
     </main>
     <AppFooter />
   </template>
   ```

6. **Comprobar paths de assets**
   - [ ] Las imágenes referenciadas como `/assets/logo.jpg` funcionan en dev (proxy Vite) y en prod (servidor Go las sirve desde `frontend/dist/assets/` después del build).
   - [ ] Videos: `<video autoplay muted loop playsinline><source src="/assets/video1.mp4"></video>`.

7. **Responsividad**
   - [ ] Probar en móvil (DevTools → iPhone 12).
   - [ ] El hero, gallery y stats deben adaptarse a 375px sin scroll horizontal.

### Detalle Crítico

```css
/* main.css — la firma visual */
:root {
  --bg: #090909;
  --gold: #f5c100;
  --red: #e63200;
  --text: #f0ede5;
  --text-dim: #a8a59c;
  --border: rgba(245, 193, 0, 0.12);
}

.clip-card {
  clip-path: polygon(10px 0%, 100% 0%, calc(100% - 10px) 100%, 0% 100%);
  background: linear-gradient(180deg, #121212 0%, #0a0a0a 100%);
  border: 1px solid var(--border);
}

.btn-gold {
  background: var(--gold);
  color: #000;
  font-family: 'Barlow Condensed', sans-serif;
  font-weight: 800;
  font-size: 18px;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  padding: 16px 32px;
  clip-path: polygon(10px 0%, 100% 0%, calc(100% - 10px) 100%, 0% 100%);
  transition: transform 0.15s ease, filter 0.15s ease;
}
.btn-gold:hover { transform: translateY(-2px); filter: brightness(1.1); }
```

### Entregable

- [ ] `npm run dev` muestra la landing visualmente idéntica al HTML viejo.
- [ ] Comparación lado a lado: abrir `/legacy/inscripcion` vs `/` → diferencias <5% (espaciados, fuentes pueden variar microscópicamente).
- [ ] Todos los videos cargan y reproducen.
- [ ] El botón "Inscríbete" navega a `/inscripcion` sin recargar.
- [ ] Lighthouse: Performance >85, Accessibility >90.

---

## FASE 5 — FORMULARIO MULTI-STEP (InscripcionView)
### "El embudo donde el visitante se vuelve piloto."

**Objetivo:** Migrar `form.html` (679 líneas) a un funnel Vue 3 multi-step con validación reactiva, persistencia en Pinia, llamada al API y manejo de los 5 métodos de pago.
**Duración estimada:** 8-10 horas
**Dependencias:** Fase 4 completa.

### Tareas

1. **Estructura de pasos**
   - **Step 1 — PersonalDataStep**: nombre_piloto, edad, tipo_documento, numero_documento, telefono, email, ciudad, instagram_user.
   - **Step 2 — CourseStep**: fecha_curso, modalidad, eps, grupo_sanguineo, familiar_nombre, familiar_telefono.
   - **Step 3 — PaymentStep**: metodo_pago. Muestra el monto calculado dinámicamente (incluye 5% surcharge si tarjeta).
   - **Step 4 — ConfirmationStep**: resumen + botón "Confirmar inscripción". Si transferencia, también `<FileUpload>` para comprobante.

2. **Componente `StepIndicator.vue`**
   - [ ] Props: `:current`, `:total`, `:labels`.
   - [ ] Muestra los 4 pasos con estados (done, current, pending) y línea de progreso.
   - [ ] Permite click en pasos previos para volver (no en futuros).

3. **Componente `BaseInput.vue`** (reutilizable)
   - [ ] Props: `:modelValue`, `:label`, `:type`, `:error`, `:required`.
   - [ ] Emite `update:modelValue`.
   - [ ] Estilos: input transparente, border bottom gold, error en red.

4. **Validación reactiva**
   - [ ] Cada step exporta una función `validate(): Record<string,string>` que retorna errores.
   - [ ] El botón "Siguiente" llama validate, popula `store.errors`, y solo avanza si está vacío.
   - [ ] Reglas:
     - Email: regex `/^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$/i`.
     - Edad: `8 <= n <= 90`.
     - Teléfono: solo dígitos, 10 chars.
     - Documento: solo dígitos.
     - Todos los campos required excepto ciudad e instagram_user.

5. **Cálculo dinámico del monto**
   ```ts
   const monto = computed(() => {
     const mod = config.value?.modalidades.find(m => m.id === form.modalidad)
     if (!mod) return 0
     const base = mod.price_cop
     return form.metodo_pago === 'tarjeta'
       ? base + Math.floor(base * config.value!.card_surcharge_pct / 100)
       : base
   })
   ```

6. **Componente `FileUpload.vue`**
   - [ ] Drag-and-drop O click-to-select.
   - [ ] Validar: ≤10MB, tipos `image/jpeg, image/png, application/pdf`.
   - [ ] Preview si imagen, ícono si PDF.
   - [ ] Emite `update:file`.

7. **Submit final** (`ConfirmationStep.vue`)
   ```ts
   async function submit() {
     store.submitting = true
     try {
       const res = await createInscripcion(store.form as CreateInscripcionRequest)
       store.result = res
       if (res.requires_comprobante && file.value) {
         await uploadComprobante(res.id, file.value)
       }
       if (res.checkout_url) {
         // Redirección same-window (clave para IG webview)
         window.location.href = res.checkout_url
       } else {
         router.push({ path: '/success', query: { ref: res.id } })
       }
     } catch (e: any) {
       error.value = e.response?.data?.error || 'Error al enviar'
     } finally {
       store.submitting = false
     }
   }
   ```

8. **Manejo del IG webview**
   - [ ] Si detectamos IG (`useInstagramBrowser`), mostrar `IgBanner` recomendando abrir en navegador externo (algunos pagos PSE no funcionan en IG webview).

9. **Diseño visual**
   - [ ] Conservar el `clip-path` en cards de cada step.
   - [ ] Botones gold para "Siguiente"/"Confirmar", outline para "Volver".
   - [ ] Resumen en Step 4 con todos los campos en formato legible.

### Detalle Crítico

```vue
<!-- InscripcionView.vue — orquestador -->
<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useInscripcionStore } from '@/stores/inscripcion'
import { fetchConfig } from '@/services/inscripciones'
import StepIndicator from '@/components/inscripcion/StepIndicator.vue'
import PersonalDataStep from '@/components/inscripcion/PersonalDataStep.vue'
import CourseStep from '@/components/inscripcion/CourseStep.vue'
import PaymentStep from '@/components/inscripcion/PaymentStep.vue'
import ConfirmationStep from '@/components/inscripcion/ConfirmationStep.vue'

const store = useInscripcionStore()
const config = ref<ConfigResponse | null>(null)
onMounted(async () => { config.value = await fetchConfig() })
</script>
<template>
  <div class="funnel">
    <StepIndicator :current="store.step" :labels="['Datos','Curso','Pago','Confirmar']" />
    <Transition name="fade" mode="out-in">
      <PersonalDataStep v-if="store.step===1" key="1" />
      <CourseStep v-else-if="store.step===2" key="2" :config="config" />
      <PaymentStep v-else-if="store.step===3" key="3" :config="config" />
      <ConfirmationStep v-else key="4" :config="config" />
    </Transition>
  </div>
</template>
```

### Entregable

- [ ] El usuario completa los 4 pasos y al "Confirmar" se inserta en DB.
- [ ] Si el método es digital, redirige a Bold checkout.
- [ ] Si es transferencia, sube el comprobante y redirige a `/success`.
- [ ] La validación bloquea el avance con campos inválidos.
- [ ] El monto calculado coincide con el que devuelve el API.
- [ ] El estado se mantiene si recargas (opcional: persistir Pinia en localStorage).

---

## FASE 6 — SUCCESS, CALLBACK, BUILD Y DESPLIEGUE
### "El piloto cruza la meta. Verifica el tiempo."

**Objetivo:** Implementar las views post-flujo, embeber el bundle Vue en el binario Go con `embed.FS`, configurar el Dockerfile multi-stage, validar Railway, eliminar lo legacy.
**Duración estimada:** 4-6 horas
**Dependencias:** Fase 5 completa.

### Tareas

1. **`SuccessView.vue`** (post-submit con transferencia)
   - [ ] Lee `?ref=ID` de la query.
   - [ ] Llama `getInscripcion(id)`.
   - [ ] Muestra: "Comprobante recibido, en validación", ID destacado, próximos pasos, link de WhatsApp.
   - [ ] Style: ícono ✅, card con clip-path.

2. **`CallbackView.vue`** (post-Bold)
   - [ ] Lee `?ref=ID&bold-tx-status=...` de la query.
   - [ ] Estado inicial: derivar de `bold-tx-status`:
     - `approved` → estado "confirmando" mientras llega el webhook.
     - `rejected` → estado "rechazado" inmediato.
     - vacío → "verificando".
   - [ ] **Polling**: cada 2s consulta `getInscripcion(id)` hasta que `status === 'pagado'` o `status === 'pago rechazado'`. Máximo 30 intentos (~60s).
   - [ ] Tres estados visuales:
     - **Verificando** (default): spinner + "Confirmación en curso..."
     - **Confirmado**: ✅ + "¡Pago confirmado!" + ID + mensaje WhatsApp.
     - **Rechazado**: ⚠️ + "Pago rechazado" + botón "Volver al formulario".
   - [ ] El polling se detiene al desmontar (cleanup en `onUnmounted`).

3. **Embed del frontend en el binario Go**
   - [ ] Crear `internal/spa/spa.go`:
     ```go
     package spa

     import (
         "embed"
         "io/fs"
         "github.com/gofiber/fiber/v2"
         "github.com/gofiber/fiber/v2/middleware/filesystem"
     )

     //go:embed all:dist
     var distFS embed.FS

     func Register(app *fiber.App) {
         sub, _ := fs.Sub(distFS, "dist")
         app.Use("/", filesystem.New(filesystem.Config{
             Root:         http.FS(sub),
             Index:        "index.html",
             NotFoundFile: "index.html",  // SPA fallback
         }))
     }
     ```
   - [ ] **Decisión clave:** o copiamos `frontend/dist` a `internal/spa/dist` antes de compilar (más explícito) **o** ajustamos el embed path con un build flag.
   - [ ] **Recomendado:** target Makefile que hace `npm run build && cp -r frontend/dist internal/spa/dist && go build`.

4. **Dockerfile multi-stage**
   ```dockerfile
   # Stage 1: build frontend
   FROM node:22-alpine AS frontend
   WORKDIR /app/frontend
   COPY frontend/package*.json ./
   RUN npm ci
   COPY frontend/ ./
   RUN npm run build

   # Stage 2: build backend
   FROM golang:1.25-alpine AS backend
   WORKDIR /app
   RUN apk add --no-cache git
   COPY go.mod go.sum ./
   RUN go mod download
   COPY . .
   COPY --from=frontend /app/frontend/dist ./internal/spa/dist
   RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server/

   # Stage 3: runtime
   FROM alpine:3.20
   RUN apk add --no-cache ca-certificates tzdata
   WORKDIR /app
   COPY --from=backend /server /app/server
   RUN mkdir -p /data/uploads
   ENV UPLOADS_DIR=/data/uploads
   EXPOSE 3000
   CMD ["/app/server"]
   ```

5. **`railway.toml`**
   ```toml
   [build]
   builder = "DOCKERFILE"
   dockerfilePath = "Dockerfile"

   [deploy]
   healthcheckPath = "/health"
   healthcheckTimeout = 30
   restartPolicyType = "ON_FAILURE"
   restartPolicyMaxRetries = 3

   [[deploy.environmentVariables]]
   name = "PUBLIC_URL"
   value = "https://${{ RAILWAY_PUBLIC_DOMAIN }}"
   ```

6. **Volumen persistente Railway** (para uploads)
   - [ ] Configurar volumen montado en `/data` en el dashboard de Railway.
   - [ ] `UPLOADS_DIR=/data/uploads`.

7. **Eliminar legacy**
   - [ ] Borrar `internal/api/legacy/` (HTML viejo).
   - [ ] Borrar ruta `/legacy/inscripcion`.
   - [ ] Verificar que no queda ningún `//go:embed *.html`.

8. **CI/Build local**
   - [ ] Makefile:
     ```makefile
     dev-frontend:
         cd frontend && npm run dev
     dev-backend:
         go run ./cmd/server/
     build-frontend:
         cd frontend && npm ci && npm run build
         rm -rf internal/spa/dist
         cp -r frontend/dist internal/spa/dist
     build: build-frontend
         go build -o server ./cmd/server/
     run: build
         ./server
     docker-build:
         docker build -t scuderia-st4ge .
     ```

9. **Smoke tests pre-deploy**
   - [ ] `make build && ./server` → arrancar local con DATABASE_URL apuntando a Postgres local.
   - [ ] Visitar `http://localhost:3000/` → ve la landing.
   - [ ] Click "Inscríbete" → navega a `/inscripcion` sin recargar.
   - [ ] Llenar form con método "transferencia" + comprobante de prueba → `/success` muestra ID.
   - [ ] Llenar form con método "tarjeta" → redirige a Bold sandbox.
   - [ ] `curl http://localhost:3000/health` → 200.
   - [ ] `curl -X POST http://localhost:3000/webhook/bold ...` con HMAC válido → 200.

10. **Deploy a Railway**
    - [ ] `git push origin feat/web-platform`.
    - [ ] Crear PR a `master`.
    - [ ] Merge → Railway auto-deploy.
    - [ ] Verificar logs: `Server starting on :3000`.
    - [ ] Visitar `https://${RAILWAY_PUBLIC_DOMAIN}/` y completar un flujo end-to-end con monto de prueba ($1000 COP).

### Detalle Crítico

**El SPA fallback es indispensable.** Vue Router usa `createWebHistory`, lo que significa que `/callback?ref=xxx` debe servir `index.html` (no 404). El middleware `filesystem` con `NotFoundFile: "index.html"` lo logra. **Pero las rutas API y `/assets/*` deben tener prioridad** — registrarlas ANTES del middleware SPA en `server.go`.

```go
// server.go — orden CRÍTICO
app.Get("/health", api.Health)
app.Get("/api/config", apiHandler.GetConfig)
app.Post("/api/inscripciones", apiHandler.Create)
// ... todas las rutas API
app.Post("/webhook/bold", apiHandler.BoldWebhook)
// ... assets desde frontend/public ya están en dist/assets, los sirve el SPA filesystem.

// SPA fallback — DEBE IR AL FINAL
spa.Register(app)
```

### Entregable

- [ ] El sitio está accesible en `https://${RAILWAY_PUBLIC_DOMAIN}/`.
- [ ] Flujo completo funciona: landing → formulario → pago → callback → confirmación.
- [ ] Telegram recibe notificación al insertar y al confirmar pago.
- [ ] Health check de Railway pasa.
- [ ] El binario producido (`server`) pesa <30 MB.
- [ ] Lighthouse en producción: Performance >80, SEO >90.

---

## 9. ESTIMACIÓN DE COSTOS

| Servicio | Plan | Costo mensual estimado |
|---|---|---|
| Railway (backend + Postgres + volumen) | Hobby | $5 (créditos) o $5-15 según uso |
| Bold (transacciones) | Por transacción | ~3% por venta digital + IVA |
| Telegram Bot API | Free | $0 |
| Dominio (opcional) | .com via Cloudflare | ~$10/año |
| **Total infra** | | **~$5-20/mes** |

---

## 10. RIESGOS Y MITIGACIONES

| # | Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|---|
| 1 | Webhook de Bold falla por firma HMAC mal construida tras refactor | Media | Alto | Conservar el código actual de `verifyBoldSignature` byte por byte. Test unitario con payload real grabado. |
| 2 | IG in-app browser bloquea redirect a Bold | Media | Alto | Mostrar `IgBanner` insistente. Usar `window.location.href` (no `window.open`). Documentar en FAQ. |
| 3 | Polling agresivo en CallbackView congestiona el backend | Baja | Medio | Backoff exponencial: 2s, 3s, 5s, 8s... Máximo 30 intentos. |
| 4 | Pérdida de comprobantes al redeploy de Railway (filesystem efímero) | Alta | Alto | **Volumen persistente Railway montado en /data** + futuro: migrar a S3/R2. |
| 5 | Bundle Vue muy pesado degrada LCP en móviles 3G | Media | Medio | Lazy-load de views (ya está). Code splitting por route. Comprimir videos a 720p. |
| 6 | Telegram rate-limit (~30 msg/s) en pico de inscripciones | Baja | Bajo | Las inscripciones son orgánicas; improbable. Si pasa: cola interna. |
| 7 | Pgxpool exhausted bajo carga | Baja | Medio | `pool_max_conns=20`, timeout 15s en handlers. |
| 8 | Migración rompe el `/inscripcion` actual durante el desarrollo | Alta | Alto | Branch separada (`feat/web-platform`), `pre-vue-migration` tag, deploy a staging primero. |

---

## 11. RESUMEN EJECUTIVO

| Fase | Nombre | Horas | Entregable verificable |
|---|---|---|---|
| 1 | Limpieza del backend | 3-4h | `go build` compila, solo módulo de inscripciones queda |
| 2 | Reorganización a `internal/api/` | 4-5h | API REST JSON funcional, pgxpool |
| 3 | Scaffolding Vue 3 | 3-4h | `npm run dev` muestra config del backend |
| 4 | Landing page | 6-8h | HomeView visualmente idéntico al HTML viejo |
| 5 | Formulario multi-step | 8-10h | Flujo completo con Bold + Telegram funciona en local |
| 6 | Success, callback, deploy | 4-6h | Producción en Railway con dominio público |
| **Total** | | **28-37h** | **Plataforma web profesional desplegada** |

### Stack final
- **Backend:** Go 1.25 + Fiber v2 + pgxpool + zap
- **Frontend:** Vue 3.5 + Vite 6 + TypeScript 5.6 + Pinia + Vue Router + axios
- **Infra:** Docker multi-stage + Railway + PostgreSQL 16 + volumen persistente

### Primer paso a ejecutar (Atlas)

```bash
cd D:/instagram-bot
git checkout master && git pull
git tag pre-vue-migration
git checkout -b feat/web-platform
git push -u origin feat/web-platform
# → Comenzar FASE 1, tarea 2 (eliminación de directorios del bot)
```

---

> "Te di el plano. El kart lo construyes tú."
> — Prometheus
