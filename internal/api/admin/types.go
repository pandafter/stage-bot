package admin

import "time"

// Auth
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Token string    `json:"token"`
	User  AdminUser `json:"user"`
}

type AdminUser struct {
	ID       int64  `json:"id"`
	TenantID string `json:"tenant_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// CMS sections
type SectionResponse struct {
	TenantID    string         `json:"tenant_id"`
	SectionKey  string         `json:"section_key"`
	Data        map[string]any `json:"data"`
	IsPublished bool           `json:"is_published"`
	Version     int            `json:"version"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type UpsertSectionRequest struct {
	Data        map[string]any `json:"data" validate:"required"`
	IsPublished *bool          `json:"is_published"`
}

// Media
type MediaItem struct {
	ID         int64     `json:"id"`
	Filename   string    `json:"filename"`
	URL        string    `json:"url"`
	StorageKey string    `json:"storage_key"`
	MimeType   string    `json:"mime_type"`
	SizeBytes  int64     `json:"size_bytes"`
	AltText    string    `json:"alt_text"`
	Tags       []string  `json:"tags"`
	CreatedAt  time.Time `json:"created_at"`
}

type UpdateMediaRequest struct {
	AltText string   `json:"alt_text"`
	Tags    []string `json:"tags"`
}

// Form dates
type FormDate struct {
	ID        int64     `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Label     string    `json:"label" validate:"required"`
	StartsOn  string    `json:"starts_on" validate:"required"` // "2026-07-12"
	IsEnabled bool      `json:"is_enabled"`
	Capacity  *int      `json:"capacity"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateFormDateRequest struct {
	Label     string `json:"label" validate:"required"`
	StartsOn  string `json:"starts_on" validate:"required"`
	IsEnabled bool   `json:"is_enabled"`
	Capacity  *int   `json:"capacity"`
	SortOrder int    `json:"sort_order"`
}

type UpdateFormDateRequest struct {
	Label     *string `json:"label"`
	StartsOn  *string `json:"starts_on"`
	IsEnabled *bool   `json:"is_enabled"`
	Capacity  *int    `json:"capacity"`
	SortOrder *int    `json:"sort_order"`
}

type ReorderRequest struct {
	IDs []int64 `json:"ids" validate:"required"`
}

// Pricing plans
type PricingPlan struct {
	ID          int64          `json:"id"`
	TenantID    string         `json:"tenant_id"`
	Key         string         `json:"key"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	PriceCOP    int            `json:"price_cop"`
	Features    []string       `json:"features"`
	ImageURL    string         `json:"image_url"`
	IsEnabled   bool           `json:"is_enabled"`
	SortOrder   int            `json:"sort_order"`
}

type CreatePlanRequest struct {
	Key         string   `json:"key" validate:"required"`
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description"`
	PriceCOP    int      `json:"price_cop" validate:"required,min=0"`
	Features    []string `json:"features"`
	IsEnabled   bool     `json:"is_enabled"`
	SortOrder   int      `json:"sort_order"`
}

// Payment methods
type PaymentMethod struct {
	ID           int64   `json:"id"`
	TenantID     string  `json:"tenant_id"`
	Key          string  `json:"key"`
	Label        string  `json:"label"`
	IsEnabled    bool    `json:"is_enabled"`
	SurchargePct float64 `json:"surcharge_pct"`
	SortOrder    int     `json:"sort_order"`
}

type UpdatePaymentMethodRequest struct {
	Label        *string  `json:"label"`
	IsEnabled    *bool    `json:"is_enabled"`
	SurchargePct *float64 `json:"surcharge_pct"`
	SortOrder    *int     `json:"sort_order"`
}

// Inscripciones admin
type InscripcionFilter struct {
	Status   string `query:"status"`
	DateFrom string `query:"date_from"`
	DateTo   string `query:"date_to"`
	Plan     string `query:"plan"`
	Search   string `query:"search"`
	Page     int    `query:"page"`
	Limit    int    `query:"limit"`
}

type UpdateInscripcionStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=pendiente pagado comprobante_pendiente rechazado cancelado"`
	Note   string `json:"note"`
}

// Theme
type Theme struct {
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
	FontHeading    string `json:"font_heading"`
	FontBody       string `json:"font_body"`
	LogoURL        string `json:"logo_url"`
	FaviconURL     string `json:"favicon_url"`
}
