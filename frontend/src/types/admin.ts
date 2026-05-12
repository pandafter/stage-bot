export interface AdminUser {
  id: number
  tenant_id: string
  username: string
  role: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: AdminUser
}

export interface MediaItem {
  id: number
  filename: string
  url: string
  storage_key: string
  mime_type: string
  size_bytes: number
  alt_text: string
  tags: string[]
  created_at: string
}

export interface FormDate {
  id: number
  tenant_id: string
  label: string
  starts_on: string
  is_enabled: boolean
  capacity?: number
  sort_order: number
  created_at: string
}

export interface PricingPlan {
  id: number
  tenant_id: string
  key: string
  name: string
  description: string
  price_cop: number
  features: string[]
  image_url: string
  is_enabled: boolean
  sort_order: number
}

export interface PaymentMethod {
  id: number
  tenant_id: string
  key: string
  label: string
  is_enabled: boolean
  surcharge_pct: number
  sort_order: number
}

export interface InscripcionFilter {
  status?: string
  date_from?: string
  date_to?: string
  plan?: string
  search?: string
  page?: number
  limit?: number
}

export interface AdminInscripcion {
  id: string
  email: string
  nombre_piloto: string
  metodo_pago: string
  plan: string
  monto_cop: number
  status: string
  created_at: string
  fecha_curso: string
}
