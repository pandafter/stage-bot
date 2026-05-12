# PROMETHEUS PLAN — Admin Panel + CMS Multi-Tenant

> "El fuego no se reparte: se enseña a encenderlo. Construimos hoy el panel para St4ge, pero diseñamos el motor para mil escuderías."

**Proyecto:** Scudería St4ge — Plataforma Web (Admin Panel + CMS)
**Fecha:** 2026-05-12
**Autor:** Prometheus
**Estado del codebase:** `feat/web-platform` (clean) — Go 1.25 + Fiber + pgxpool + Vue 3 + Vite
**Repo:** `D:\Instagram-bot`

> Este plan reemplaza al anterior (migración inicial a plataforma web, ya completada). Aquí se diseña la siguiente etapa: el panel de administración y la base multi-tenant.

---

## 1. VISIÓN GLOBAL

Construir un **Admin Panel + CMS embebido** dentro del binario Go actual que permita:

1. **Hoy (St4ge):** Editar todo el contenido del landing y la config del formulario sin tocar código ni redeploy.
2. **Mañana (SaaS template):** Levantar una instancia clon para cualquier otra empresa cambiando únicamente datos en BD (tema, copy, formulario, dominio).

**No se reemplaza nada.** El `/api/config` actual sigue respondiendo igual — solo que ahora lee de BD con fallback a los defaults hardcoded actuales. Cero downtime, cero ruptura.

---

## 2. PRINCIPIOS DE DISEÑO

1. **Multi-tenant desde el día 1, sin pagar el costo hoy.** Toda tabla nueva lleva `tenant_id`. Hoy hay un único tenant (`st4ge`) resuelto por host/default. Mañana, resolución por subdominio o header.
2. **Schema-on-read flexible vía JSONB.** El contenido del CMS es JSONB por sección. Esto evita migraciones cada vez que St4ge cambia la forma de un componente. Los campos críticos (precios, fechas, status) van en columnas tipadas.
3. **Defaults compilados como red de seguridad.** Si BD está vacía o el CMS no tiene una sección, el sistema sirve los valores hardcoded actuales en Go. Nunca un landing roto.
4. **Un solo binario, un solo bundle.** El admin es una sub-ruta `/admin/*` en el mismo Vue SPA. Sin micro-frontends. Code-splitting por route ya lo hace Vite.
5. **Auth admin simple, no enterprise.** JWT firmado con secreto en env, usuario/password en env (hash bcrypt). Sin OAuth, sin reset de password, sin RBAC complejo. **Hasta que se necesite.**
6. **Optimistic cache + invalidación brutal.** El landing público cachea `/api/config` 60s en memoria. Cuando el admin guarda, bump del `etag` y el frontend público refetcha al próximo load.
7. **NO over-engineer la V1.** Lo que se posterga está explícito en la sección 11.

---

## 3. ARQUITECTURA

```
                         ┌─────────────────────────────────────┐
                         │     SPA Vue (un solo bundle)         │
                         │                                      │
                         │  /            → Landing (público)    │
                         │  /inscripcion → Form (público)       │
                         │  /admin/*     → Admin (auth JWT)     │
                         └────────────┬─────────────────────────┘
                                      │ HTTP
                  ┌───────────────────┼──────────────────────────┐
                  │                   │                          │
            /api/config        /api/inscripciones        /api/admin/*
            (público,          (público)                 (JWT required)
             cacheado 60s)                                       │
                  │                   │                          │
                  └─────────┬─────────┴──────────────┬───────────┘
                            │                        │
                  ┌─────────▼───────────┐  ┌─────────▼──────────┐
                  │   Fiber Handlers     │  │  Admin Handlers     │
                  │   (existentes)       │  │  + Middleware JWT   │
                  └─────────┬───────────┘  └─────────┬──────────┘
                            │                        │
                            └────────┬───────────────┘
                                     │
                          ┌──────────▼──────────┐
                          │   pgxpool (Postgres)  │
                          │                       │
                          │  tenants              │
                          │  admin_users          │
                          │  cms_sections (JSONB) │
                          │  cms_media            │
                          │  form_dates           │
                          │  pricing_plans        │
                          │  payment_methods      │
                          │  inscripciones (✓)    │
                          └──────────┬──────────┘
                                     │
                          ┌──────────▼──────────┐
                          │   R2 / Filesystem    │
                          │   (imágenes CMS,     │
                          │   comprobantes)      │
                          └─────────────────────┘
```

**Resolución de tenant (hoy → mañana):**
- Hoy: `tenant_id = "st4ge"` resuelto por middleware desde `cfg.DefaultTenant`.
- Mañana: middleware lee `Host` header → query `tenants.domain` → inyecta `tenant_id` al ctx.

---

## 4. STACK TECNOLÓGICO

| Capa | Tecnología | Por qué |
|------|------------|---------|
| Backend | Go 1.25 + Fiber v2 (sin cambios) | Ya está, funciona. |
| ORM | pgx raw (sin cambios) | Mantener patrón actual; no introducir sqlc todavía. |
| Auth admin | `golang-jwt/jwt/v5` + `bcrypt` | Standard, sin deps grandes. |
| Validación | `go-playground/validator` (ya en uso) | — |
| Frontend admin | Vue 3 + `<script setup>` + Pinia | Mismo stack. |
| UI admin | **PrimeVue** | Componentes admin-ready (tablas, modals, datepickers, color pickers). Maduro y tree-shakeable. |
| Editor rich-text | `@tiptap/vue-3` | Headless, ligero. |
| Upload imágenes | input file + drag-drop nativo | Sin Filepond/Uppy en V1. |
| Storage assets | **Cloudflare R2** (S3-compatible) | Egress gratis, $0.015/GB. Fallback: filesystem `UPLOADS_DIR`. |
| Sortable | `vuedraggable@next` | Para reordenar (FAQ, galería). |
| Migraciones | Append al slice en `postgres.go` (patrón actual) | Sin golang-migrate todavía. |

**No se introducen:** Redis, microservicios, mensajería, GraphQL, ORM pesados.

---

## 5. ESTRUCTURA DEL PROYECTO

Lo nuevo está marcado con `+`. Lo modificado con `~`.

```
D:\Instagram-bot\
├── cmd/
│   ├── server/main.go                       ~ DI: añadir AdminHandler, CMSRepo, TenantsRepo
│   └── hashpass/main.go                      + helper para generar bcrypt hash offline
├── internal/
│   ├── api/
│   │   ├── config.go                        ~ leer de CMSRepo con fallback a defaults
│   │   ├── inscripciones.go                 (sin cambios)
│   │   ├── bold_webhook.go                  (sin cambios)
│   │   ├── instagram.go                     (sin cambios)
│   │   └── admin/                           + nuevo subpaquete
│   │       ├── types.go                     + DTOs request/response
│   │       ├── auth.go                      + login, refresh, JWT
│   │       ├── middleware.go                + RequireJWT, tenant resolver
│   │       ├── cms.go                       + GET/PUT secciones CMS
│   │       ├── media.go                     + upload, list, delete imágenes
│   │       ├── form_config.go               + CRUD fechas, precios, métodos
│   │       ├── inscripciones.go             + list con filtros, update status manual
│   │       ├── theme.go                     + GET/PUT tema
│   │       └── tenants.go                   + GET /me (sin UI completa en V1)
│   ├── storage/
│   │   ├── postgres.go                      ~ añadir migraciones nuevas
│   │   ├── cms.go                           + CMSRepo (sections, media)
│   │   ├── tenants.go                       + TenantsRepo
│   │   ├── admin_users.go                   + AdminUsersRepo
│   │   ├── form_config.go                   + FormConfigRepo (fechas, plans, métodos)
│   │   └── inscripciones.go                 ~ añadir List(filters) + UpdateStatusAdmin
│   ├── auth/                                + nuevo paquete
│   │   ├── jwt.go                           + sign, verify
│   │   └── password.go                      + bcrypt helpers
│   ├── media/                                + nuevo paquete
│   │   ├── store.go                          + interface MediaStore
│   │   ├── r2.go                             + R2 implementation
│   │   └── fs.go                             + filesystem implementation
│   ├── tenant/                               + nuevo paquete
│   │   └── resolver.go                       + middleware Host → tenant_id
│   ├── config/config.go                     ~ añadir JWTSecret, AdminUser, AdminPasswordHash, R2 creds, DefaultTenant
│   └── server/server.go                     ~ registrar rutas /api/admin/*
└── frontend/
    └── src/
        ├── views/
        │   ├── HomeView.vue                  ~ leer secciones desde store CMS
        │   ├── InscripcionView.vue           ~ usa config ya extendida
        │   └── admin/                        + nueva carpeta
        │       ├── LoginView.vue             + form login
        │       ├── DashboardView.vue         + métricas resumidas
        │       ├── CmsLandingView.vue        + tabs por sección
        │       ├── CmsMediaView.vue          + galería de assets
        │       ├── FormConfigView.vue        + fechas, precios, métodos
        │       ├── InscripcionesView.vue     + tabla con filtros
        │       ├── InscripcionDetailView.vue + detalle + cambio de status
        │       └── ThemeView.vue             + colores, fuentes, logo
        ├── components/
        │   ├── landing/*                     ~ recibir props desde stores (no hardcode)
        │   └── admin/                        + nueva carpeta
        │       ├── AdminLayout.vue           + sidebar + topbar + outlet
        │       ├── SectionEditor.vue         + editor schema-driven
        │       ├── ImageUpload.vue           + drag-drop wrapper
        │       ├── RichTextEditor.vue        + wrapper Tiptap
        │       ├── SortableList.vue          + reordenar (FAQ, gallery)
        │       └── ConfirmDialog.vue
        ├── stores/
        │   ├── inscripcion.ts                (sin cambios)
        │   ├── cms.ts                        + cache de secciones públicas
        │   └── admin/
        │       ├── auth.ts                   + token, user, login/logout
        │       ├── cms.ts                    + estado editable de secciones
        │       ├── formConfig.ts             + fechas, precios, métodos
        │       └── inscripciones.ts          + lista con filtros
        ├── services/
        │   ├── api.ts                        ~ interceptor JWT
        │   ├── admin.ts                      + cliente para /api/admin/*
        │   └── cms.ts                        + cliente para secciones públicas
        ├── types/
        │   ├── api.ts                        (sin cambios)
        │   ├── cms.ts                        + tipos CMS
        │   └── admin.ts                      + tipos admin (auth, filters)
        └── router/index.ts                   ~ añadir rutas /admin/* con guard JWT
```

---

## 6. ESQUEMA DE BASE DE DATOS

Todas las tablas nuevas llevan `tenant_id`. Hoy se inserta `'st4ge'` por defecto.

```sql
-- 1. Tenants (multi-tenant root)
CREATE TABLE IF NOT EXISTS tenants (
  id           TEXT PRIMARY KEY,              -- slug, ej. 'st4ge'
  name         TEXT NOT NULL,
  domain       TEXT UNIQUE,                   -- 'scuderiast4ge.com' (null en dev)
  theme        JSONB NOT NULL DEFAULT '{}',   -- colores, fuentes, logo_url
  features     JSONB NOT NULL DEFAULT '{}',   -- flags: bold_enabled, telegram_enabled
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Seed: INSERT INTO tenants (id, name) VALUES ('st4ge','Scudería St4ge') ON CONFLICT DO NOTHING;

-- 2. Admin users (un usuario por tenant en V1)
CREATE TABLE IF NOT EXISTS admin_users (
  id            BIGSERIAL PRIMARY KEY,
  tenant_id     TEXT NOT NULL REFERENCES tenants(id),
  username      TEXT NOT NULL,
  password_hash TEXT NOT NULL,                -- bcrypt cost 12
  role          TEXT NOT NULL DEFAULT 'owner',
  last_login_at TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, username)
);

-- 3. CMS sections
CREATE TABLE IF NOT EXISTS cms_sections (
  tenant_id     TEXT NOT NULL REFERENCES tenants(id),
  section_key   TEXT NOT NULL,                -- 'hero','stats','program','instructores','gallery','faq','cta'
  data          JSONB NOT NULL,
  is_published  BOOLEAN NOT NULL DEFAULT TRUE,
  version       INTEGER NOT NULL DEFAULT 1,   -- bump on update → cache-bust ETag
  updated_by    BIGINT REFERENCES admin_users(id),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, section_key)
);

-- 4. CMS media
CREATE TABLE IF NOT EXISTS cms_media (
  id           BIGSERIAL PRIMARY KEY,
  tenant_id    TEXT NOT NULL REFERENCES tenants(id),
  filename     TEXT NOT NULL,
  url          TEXT NOT NULL,
  storage_key  TEXT NOT NULL,
  mime_type    TEXT NOT NULL,
  size_bytes   BIGINT NOT NULL,
  width        INTEGER,
  height       INTEGER,
  alt_text     TEXT,
  tags         TEXT[],
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cms_media_tenant ON cms_media(tenant_id, created_at DESC);

-- 5. Form: fechas de curso
CREATE TABLE IF NOT EXISTS form_dates (
  id           BIGSERIAL PRIMARY KEY,
  tenant_id    TEXT NOT NULL REFERENCES tenants(id),
  label        TEXT NOT NULL,
  starts_on    DATE NOT NULL,
  is_enabled   BOOLEAN NOT NULL DEFAULT TRUE,
  capacity     INTEGER,
  sort_order   INTEGER NOT NULL DEFAULT 0,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_form_dates_tenant ON form_dates(tenant_id, sort_order);

-- 6. Form: planes/modalidades
CREATE TABLE IF NOT EXISTS pricing_plans (
  id             BIGSERIAL PRIMARY KEY,
  tenant_id      TEXT NOT NULL REFERENCES tenants(id),
  key            TEXT NOT NULL,
  name           TEXT NOT NULL,
  description    TEXT,
  price_cop      INTEGER NOT NULL,
  features       JSONB NOT NULL DEFAULT '[]',
  image_media_id BIGINT REFERENCES cms_media(id),
  is_enabled     BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order     INTEGER NOT NULL DEFAULT 0,
  UNIQUE (tenant_id, key)
);

-- 7. Form: métodos de pago
CREATE TABLE IF NOT EXISTS payment_methods (
  id            BIGSERIAL PRIMARY KEY,
  tenant_id     TEXT NOT NULL REFERENCES tenants(id),
  key           TEXT NOT NULL,
  label         TEXT NOT NULL,
  is_enabled    BOOLEAN NOT NULL DEFAULT TRUE,
  surcharge_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
  sort_order    INTEGER NOT NULL DEFAULT 0,
  config        JSONB NOT NULL DEFAULT '{}',
  UNIQUE (tenant_id, key)
);

-- 8. Inscripciones: añadir tenant_id (3 pasos seguros en prod)
ALTER TABLE inscripciones ADD COLUMN IF NOT EXISTS tenant_id TEXT;
UPDATE inscripciones SET tenant_id = 'st4ge' WHERE tenant_id IS NULL;
ALTER TABLE inscripciones ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE inscripciones ADD CONSTRAINT fk_inscripciones_tenant
  FOREIGN KEY (tenant_id) REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_inscripciones_tenant_status
  ON inscripciones(tenant_id, status, created_at DESC);

-- 9. Audit log (Fase 9)
CREATE TABLE IF NOT EXISTS admin_audit_log (
  id          BIGSERIAL PRIMARY KEY,
  tenant_id   TEXT NOT NULL,
  admin_id    BIGINT NOT NULL,
  action      TEXT NOT NULL,        -- 'cms.update', 'inscripcion.status_change', etc.
  entity      TEXT NOT NULL,        -- 'cms_sections', 'inscripciones', ...
  entity_id   TEXT,
  diff        JSONB,
  at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_tenant_at ON admin_audit_log(tenant_id, at DESC);
```

**Forma de los JSONB por sección** (contratos validados server-side con structs tipados):

```json
// section_key = 'hero'
{ "title": "Aprende karting con los mejores",
  "subtitle": "Academia oficial...",
  "cta_text": "Inscríbete",
  "cta_href": "/inscripcion",
  "bg_media_id": 12 }

// section_key = 'stats'
{ "items": [ { "icon": "trophy", "value": "+200", "label": "pilotos formados" } ] }

// section_key = 'program'
{ "intro": "...", "plans_ref": "pricing_plans" }  // planes vienen de su tabla

// section_key = 'instructores'
{ "items": [ { "name": "...", "role": "...", "bio_html": "...", "photo_media_id": 7 } ] }

// section_key = 'gallery'
{ "media_ids": [12,13,14,15] }

// section_key = 'faq'
{ "items": [ { "q": "...", "a_html": "..." } ] }

// section_key = 'cta'
{ "title": "...", "subtitle": "...", "bg_media_id": 9, "cta_text": "...", "cta_href": "..." }
```

---

## 7. ENDPOINTS NUEVOS

### Públicos (extienden los existentes)
```
GET /api/config       — ahora incluye: { ...defaults_actuales, cms: {...}, theme: {...} }
                        Cache-Control: public, max-age=60, must-revalidate
                        ETag: "<tenant>:<max_version>"
GET /api/media/:id    — redirect 302 a URL pública
```

### Admin (todos requieren `Authorization: Bearer <JWT>`)
```
POST   /api/admin/auth/login              — { username, password } → { token, user }
POST   /api/admin/auth/refresh            — refresca token
GET    /api/admin/me                       — info del admin logueado

# CMS sections
GET    /api/admin/cms/sections             — todas las secciones del tenant
GET    /api/admin/cms/sections/:key
PUT    /api/admin/cms/sections/:key        — { data: {...} } → bump version

# Media
GET    /api/admin/media                    — paginado, filtros ?tag=hero
POST   /api/admin/media                    — multipart upload → { id, url }
DELETE /api/admin/media/:id                — bloquea si referenciado
PATCH  /api/admin/media/:id                — { alt_text, tags }

# Form config
GET/POST    /api/admin/form/dates
PATCH/DELETE /api/admin/form/dates/:id
POST   /api/admin/form/dates/reorder       — { ids: [3,1,2] }

GET/POST    /api/admin/form/plans
PATCH/DELETE /api/admin/form/plans/:id

GET    /api/admin/form/methods
PATCH  /api/admin/form/methods/:id

# Inscripciones (vista admin)
GET    /api/admin/inscripciones            — ?status=&date_from=&date_to=&plan=&search=
GET    /api/admin/inscripciones/:id
PATCH  /api/admin/inscripciones/:id/status — { status, note? }
GET    /api/admin/inscripciones/export.csv

# Theme
GET    /api/admin/theme
PUT    /api/admin/theme

# Tenants
GET    /api/admin/tenants/me
```

---

## 8. FASES DE EJECUCIÓN

---

### FASE 0 — RECONOCIMIENTO Y BASE DE TIPOS
### "Antes de prender el fuego, mira de qué madera es la leña."

**Objetivo:** Sentar la base de tipos compartidos y la estructura del proyecto sin tocar runtime.
**Duración:** 3 h
**Dependencias:** ninguna.

#### Tareas
- [ ] Crear branch: `feat/admin-cms` desde `feat/web-platform`.
- [ ] Crear subdirectorios `internal/api/admin/`, `internal/auth/`, `internal/media/`, `internal/tenant/`.
- [ ] Crear subdirectorios `frontend/src/views/admin/`, `frontend/src/components/admin/`, `frontend/src/stores/admin/`.
- [ ] Definir tipos TS en `frontend/src/types/cms.ts` y `admin.ts`.
- [ ] Definir structs Go en `internal/api/admin/types.go`.
- [ ] Añadir vars al `.env.example`: `JWT_SECRET`, `ADMIN_USERNAME`, `ADMIN_PASSWORD_HASH`, `DEFAULT_TENANT`, `R2_ACCOUNT_ID`, `R2_ACCESS_KEY`, `R2_SECRET_KEY`, `R2_BUCKET`, `R2_PUBLIC_URL`.
- [ ] Añadir esos campos a `internal/config/config.go`.

#### Entregable
`go build ./...` y `npm run build` siguen verdes. Cero cambios funcionales.

---

### FASE 1 — MIGRACIONES Y MULTI-TENANT FOUNDATION
### "Echamos los cimientos de mil instancias en una sola noche."

**Objetivo:** Crear las tablas nuevas, seed del tenant `st4ge`, backfill de `inscripciones.tenant_id`.
**Duración:** 4 h
**Dependencias:** Fase 0.

#### Tareas
- [ ] Añadir todas las migraciones de §6 al slice en `internal/storage/postgres.go`.
- [ ] Migración de seed: `INSERT INTO tenants ... ON CONFLICT DO NOTHING`.
- [ ] Seed de `payment_methods`, `pricing_plans`, `form_dates` con valores actuales extraídos de `internal/api/types.go`.
- [ ] Seed de `cms_sections` con la data hardcoded actual de cada componente Vue (extraída a JSON).
- [ ] Crear repos:
  - `internal/storage/tenants.go` — `Get(id)`, `GetByDomain(host)`
  - `internal/storage/admin_users.go` — `GetByUsername`, `UpdateLastLogin`, `Create`
  - `internal/storage/cms.go` — `GetSections(tenant)`, `GetSection(tenant,key)`, `UpsertSection`
  - `internal/storage/form_config.go` — repos para dates/plans/methods
- [ ] `internal/tenant/resolver.go`: middleware que setea `c.Locals("tenant_id", cfg.DefaultTenant)`. En V1 siempre default.
- [ ] Test manual: arrancar server, verificar migraciones y seeds.

#### Detalle crítico
Backfill de `inscripciones.tenant_id` debe ser **idempotente**. Patrón: `ADD COLUMN IF NOT EXISTS` → `UPDATE WHERE NULL` → `SET NOT NULL` solo si no quedan nulls. Backup antes de correr en prod.

#### Entregable
Tablas en Railway, seeds aplicados, `inscripciones.tenant_id='st4ge'` en todas las filas. Endpoints existentes siguen funcionando.

---

### FASE 2 — AUTH ADMIN (JWT) + MIDDLEWARE
### "Una llave, una puerta, una cerradura — pero forjadas para abrir mil más."

**Objetivo:** Login funcional con JWT y middleware de protección.
**Duración:** 4 h
**Dependencias:** Fase 1.

#### Tareas
- [ ] `internal/auth/password.go`: `Hash(pwd)` y `Verify(pwd, hash)` (bcrypt cost 12).
- [ ] `internal/auth/jwt.go`: `Sign(claims)`, `Verify(token)`. Claims: `{ sub: admin_id, tenant: tenant_id, exp, iat }`. TTL 8h.
- [ ] `internal/api/admin/auth.go`: `Login(c)` valida contra `admin_users`, emite JWT.
- [ ] `internal/api/admin/middleware.go`: `RequireJWT()` valida bearer e inyecta `admin_id` y `tenant_id` al `c.Locals`.
- [ ] Bootstrap admin: si `admin_users` está vacío para el tenant y `ADMIN_PASSWORD_HASH` está en env, crear el usuario al startup.
- [ ] Registrar `POST /api/admin/auth/login` (sin middleware) y el grupo `/api/admin` con `RequireJWT()`.
- [ ] `GET /api/admin/me` como smoke test.
- [ ] `cmd/hashpass/main.go`: helper que recibe `-pwd` y emite bcrypt hash.

#### Detalle crítico
**Generar password hash offline** y pegarlo en Railway env. NUNCA pedir password en plano por env.

#### Entregable
`POST /api/admin/auth/login` devuelve JWT. `GET /api/admin/me` con Bearer responde 200.

---

### FASE 3 — CMS API: SECCIONES Y MEDIA
### "Las palabras del landing dejan de ser tatuajes en el código y se vuelven tinta lavable."

**Objetivo:** Endpoints para leer/escribir secciones del CMS y subir/listar media.
**Duración:** 6 h
**Dependencias:** Fase 2.

#### Tareas
- [ ] `internal/media/store.go`: interface `MediaStore` con `Put(ctx, key, reader, mime)`, `Delete`, `URL`.
- [ ] `internal/media/fs.go`: implementación filesystem sobre `cfg.UploadsDir`.
- [ ] `internal/media/r2.go`: implementación R2 vía `aws-sdk-go-v2`. Factory escoge según presencia de `R2_BUCKET`.
- [ ] `internal/api/admin/cms.go`: `GetSections`, `GetSection`, `PutSection` (upsert + version bump).
- [ ] `internal/api/admin/media.go`: `Upload` (multipart, valida mime `image/*`, máx 5MB, redimensiona con `disintegration/imaging` a max 2000px ancho), `List` paginado, `Delete` (bloquea si referenciado en JSONB).
- [ ] **Modificar `GetConfig` público**: compone respuesta con `cms_sections` + defaults fallback. Cache in-process 60s con ETag = `max(version)`.
- [ ] Servir `/uploads/*` estático si MediaStore es FS.

#### Detalle crítico
Validar cada `data` de sección con **structs Go tipados** (`HeroData`, `StatsData`, ...) usando `validator`. JSONB inválido → 422. Esto evita basura en BD.

#### Entregable
`PUT /api/admin/cms/sections/hero` con `{ data: {...} }` actualiza la BD. `GET /api/config` devuelve el contenido nuevo.

---

### FASE 4 — FRONTEND ADMIN: SHELL + LOGIN
### "Construimos primero la entrada del taller, después el taller mismo."

**Objetivo:** Login funcional en `/admin/login`, layout autenticado en `/admin/*`.
**Duración:** 5 h
**Dependencias:** Fase 3.

#### Tareas
- [ ] Instalar `primevue`, `@primevue/themes`, `@tiptap/vue-3`, `@tiptap/starter-kit`, `vuedraggable@next`.
- [ ] `services/admin.ts`: axios con interceptor que adjunta `Bearer` desde el store; 401 → redirige a `/admin/login`.
- [ ] `stores/admin/auth.ts`: state `{ token, user }`, persistencia en `localStorage`, acciones `login`, `logout`.
- [ ] `views/admin/LoginView.vue`: form simple.
- [ ] `components/admin/AdminLayout.vue`: sidebar (Dashboard, Landing, Media, Formulario, Inscripciones, Tema), topbar con username/logout, `<router-view>`.
- [ ] `router/index.ts`: rutas `/admin/login`, `/admin/*` con guard `beforeEach`.
- [ ] `views/admin/DashboardView.vue`: tarjetas con conteos básicos.

#### Entregable
`/admin` redirige a login si no autenticado. Login exitoso → sidebar + dashboard.

---

### FASE 5 — FRONTEND ADMIN: CMS DEL LANDING
### "Le damos a St4ge el martillo y el yunque."

**Objetivo:** Editor completo por sección del landing.
**Duración:** 10 h (la fase más grande)
**Dependencias:** Fase 4.

#### Tareas
- [ ] `stores/admin/cms.ts`: carga, edición y guardado con dirty tracking.
- [ ] `components/admin/SectionEditor.vue`: wrapper schema-driven con save/discard.
- [ ] `components/admin/ImageUpload.vue`: drag-drop, preview, integra con `/api/admin/media`, devuelve `media_id`.
- [ ] `components/admin/RichTextEditor.vue`: Tiptap básico (bold, italic, link, list).
- [ ] `components/admin/SortableList.vue`: `vuedraggable` para reordenar.
- [ ] `views/admin/CmsLandingView.vue`: TabView con uno por sección (Hero, Stats, Program, Instructores, Gallery, FAQ, CTA).
- [ ] `views/admin/CmsMediaView.vue`: galería con upload, filtros, copy URL, eliminar.
- [ ] Modificar componentes públicos (`HeroSection.vue`, etc.) para leer del `stores/cms.ts` con **fallback a defaults**. Cero ruptura.

#### Detalle crítico
- Save = live (sin preview en V1).
- Cache-bust automático vía ETag: el frontend público recibe nuevo ETag y refetchea al próximo load.

#### Entregable
Admin edita el hero, guarda, abre `/`, refresca y ve el cambio.

---

### FASE 6 — FORMULARIO: FECHAS, PRECIOS, MÉTODOS
### "El formulario respira en tiempo real con cada decisión del director."

**Objetivo:** Admin gestiona fechas, planes y métodos de pago.
**Duración:** 6 h
**Dependencias:** Fase 5.

#### Tareas
- [ ] Backend: CRUD para `form_dates`, `pricing_plans`, `payment_methods`.
- [ ] Modificar `GetConfig` público: `Modalidades`, `Fechas`, `Metodos` salen de estas tablas con fallback a hardcoded si están vacías.
- [ ] `views/admin/FormConfigView.vue`: tres tabs (Fechas, Planes, Métodos), DataTable PrimeVue + modal crear/editar.
- [ ] Toggle `is_enabled` inline.
- [ ] Reordenar drag-drop (`sort_order`).
- [ ] **Soft-disable** en lugar de borrar planes referenciados por inscripciones existentes.

#### Entregable
Director desactiva una fecha pasada y agrega una nueva. Al recargar `/inscripcion` ve las nuevas opciones.

---

### FASE 7 — INSCRIPCIONES: TABLA + DETALLE + STATUS MANUAL
### "Cada inscripción cuenta su historia, y el director la escucha sin abrir la base de datos."

**Objetivo:** Vista de inscripciones con filtros y cambio manual de status.
**Duración:** 5 h
**Dependencias:** Fase 4 (no requiere CMS).

#### Tareas
- [ ] Backend:
  - `InscripcionesRepo.List(filters)` con paginación.
  - `internal/api/admin/inscripciones.go`: `List`, `Get`, `UpdateStatus` (con nota opcional), `ExportCSV`.
  - Status válidos: `pendiente`, `pagado`, `comprobante_pendiente`, `rechazado`, `cancelado`.
- [ ] Frontend:
  - `stores/admin/inscripciones.ts`.
  - `views/admin/InscripcionesView.vue`: DataTable con filtros (status, rango fechas, plan, búsqueda), botón "Exportar CSV".
  - `views/admin/InscripcionDetailView.vue`: todos los campos, link al comprobante, dropdown status + nota.
- [ ] Notificar Telegram cuando admin cambia status (reusa cliente existente).

#### Entregable
Director ve lista, filtra por `comprobante_pendiente`, abre uno, revisa, marca como `pagado` y recibe ping en Telegram.

---

### FASE 8 — TEMA Y MULTI-TENANT READY
### "El mismo motor pintado de otro color sigue siendo un motor."

**Objetivo:** Editor de tema y verificación de viabilidad multi-tenant.
**Duración:** 4 h
**Dependencias:** Fase 5.

#### Tareas
- [ ] Backend: `GET/PUT /api/admin/theme` escribe a `tenants.theme`.
- [ ] `GetConfig` público incluye `theme`.
- [ ] Frontend: en `main.ts` o `App.vue`, al cargar config, inyectar CSS vars dinámicas (`--color-primary`, `--font-heading`, ...) en `:root`. Refactor de Tailwind config para que los componentes del landing usen esas vars.
- [ ] `views/admin/ThemeView.vue`: ColorPicker, font selectors, upload de logo/favicon.
- [ ] **Test multi-tenant:** crear tenant `'demo'` en BD, seedear secciones distintas, levantar server con `DEFAULT_TENANT=demo` → verificar que el landing cambia 100%.
- [ ] Documento `docs/MULTI_TENANT.md` con pasos para clonar para un nuevo cliente.

#### Entregable
Cambiar color primario en `/admin/theme` y verlo reflejado en el landing tras refresh. Documento de multi-tenant publicado.

---

### FASE 9 — HARDENING + DEPLOY
### "El fuego está encendido. Ahora lo enseñamos a sobrevivir al viento."

**Objetivo:** Hardening, observabilidad y deploy a Railway.
**Duración:** 4 h
**Dependencias:** Fases 1–8.

#### Tareas
- [ ] Rate limit a `/api/admin/auth/login` (5/min/IP) con `fiber/middleware/limiter`.
- [ ] Tabla `admin_audit_log` (definida en §6). Insert en cada PUT/DELETE admin.
- [ ] Logs estructurados (zap) en todas las rutas admin: quién, qué, cuándo.
- [ ] CSP headers para `/admin/*`.
- [ ] Smoke tests manuales documentados en `docs/ADMIN_SMOKE.md`.
- [ ] Actualizar `Dockerfile` si hace falta runtime env nueva.
- [ ] Actualizar `.env.example` y `CLAUDE.md` con sección "Admin".
- [ ] Probar en Railway con tenant nuevo desde cero (sin seed).
- [ ] Crear primer admin de producción con `cmd/hashpass`.

#### Entregable
Sistema en producción con audit log y rate limit. St4ge edita su landing sin intervención técnica.

---

## 9. ESTIMACIÓN DE COSTOS

| Recurso | Costo estimado |
|---------|----------------|
| Railway (Postgres + server) | sin cambio respecto a hoy |
| Cloudflare R2 | ~$0 hasta 10 GB / $0.015 por GB adicional. Egress GRATIS. |
| Fallback filesystem | $0 (volumen Railway, pero efímero) |
| Tiempo de desarrollo | **51 horas** / ~7 días dev full-time |

**Recomendación:** R2 desde el día 1. Setup trivial, costo despreciable, evita problemas con volúmenes Railway efímeros.

---

## 10. RIESGOS Y MITIGACIONES

| # | Riesgo | Impacto | Mitigación |
|---|--------|---------|------------|
| 1 | Migración de `inscripciones.tenant_id` falla en prod | Alto | 3 pasos (ADD NULL → UPDATE → SET NOT NULL); backup previo. |
| 2 | JWT secret leak | Alto | 32-byte random, solo en Railway env, rotar al sospechar, TTL 8h. |
| 3 | Admin escribe JSON inválido y rompe el landing | Medio | Validación server-side con structs tipados por sección; defaults fallback en frontend. |
| 4 | Uploads enormes saturan disco/R2 | Bajo | 5MB cap, validación mime, redimensionar max 2000px server-side. |
| 5 | Cache 60s impide ver cambios inmediatos | Bajo | ETag por `max(version)`; botón "Limpiar cache" en admin si urge. |
| 6 | Filtración entre tenants | Crítico | Todo query admin filtra por `tenant_id` desde `c.Locals`. Code review obligatorio en handlers admin. |

---

## 11. LO QUE *NO* SE HACE EN V1 (anti over-engineering)

1. **RBAC granular.** Un solo rol `owner` por tenant.
2. **Multi-usuario admin con invitaciones por email.** Un admin por tenant, bootstrap.
3. **Reset de password / forgot password.** Se regenera hash y se actualiza env/BD manualmente.
4. **OAuth.** No.
5. **Form fields personalizados (`form_fields`).** Campos fijos en V1.
6. **Preview antes de publicar.** Save = live. Versionado/rollback queda para V1.1.
7. **Editor visual WYSIWYG completo tipo Webflow.** Solo editor por sección.
8. **i18n.** Admin en español.
9. **Webhooks salientes (Zapier).** No.
10. **Dashboard analytics propio.** Conectar GA4 vía script configurable basta.
11. **Resolución de tenant por subdominio.** Middleware listo, V1 usa `DEFAULT_TENANT`. Se activa en V1.1.
12. **Tests automatizados extensivos.** Smoke tests manuales + unit tests solo en `internal/auth`. Atlas decide cuánto más añadir.

---

## 12. RESUMEN EJECUTIVO

| Fase | Nombre | Horas | Entregable clave |
|------|--------|-------|------------------|
| 0 | Reconocimiento y tipos | 3 | Skeleton + tipos, build verde |
| 1 | Migraciones + Multi-tenant | 4 | Tablas + seeds + `tenant_id` en inscripciones |
| 2 | Auth JWT | 4 | Login funcional, `/api/admin/me` ok |
| 3 | CMS API + Media | 6 | `PUT /api/admin/cms/sections/:key` operativo |
| 4 | Frontend shell + login | 5 | `/admin` con sidebar |
| 5 | CMS UI del landing | 10 | Director edita todo el landing |
| 6 | Form config UI | 6 | Fechas, planes y métodos editables |
| 7 | Inscripciones UI | 5 | Tabla con filtros + status manual |
| 8 | Tema + multi-tenant ready | 4 | Cambio de colores live + doc multi-tenant |
| 9 | Hardening + deploy | 4 | Producción con audit log + rate limit |
| **TOTAL** | | **51 h** | Plataforma SaaS template-ready en producción |

**Stack confirmado:** Go 1.25 + Fiber + pgx + Vue 3 + PrimeVue + Tiptap + Cloudflare R2.

**Primer paso ejecutable (Atlas):**
> Crear branch `feat/admin-cms` desde `feat/web-platform`, ejecutar Fase 0 (3h). Al finalizar, abrir PR draft con la estructura de archivos vacía y los tipos definidos. Esto desbloquea Fase 1 en paralelo entre backend (migraciones) y frontend (tipos TS).

---

> "Prometeo no enciende el fuego: enciende la capacidad de mantenerlo."
