export interface AdminUser {
  id: string
  username: string
  email?: string
  role: 'admin' | 'editor'
  created_at: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: AdminUser
}

export interface CmsSection {
  key: string
  label: string
  data: Record<string, unknown>
  is_published: boolean
  updated_at: string
}

export interface MediaAsset {
  id: number
  filename: string
  url: string
  alt_text: string
  tags: string[]
  uploaded_at: string
  size: number
}

export interface FormDate {
  id: number
  date: string
  capacity: number
  available_slots: number
  is_active: boolean
  display_order: number
}

export interface FormPlan {
  id: number
  name: string
  price_cop: number
  description: string
  is_active: boolean
  display_order: number
}

export interface FormMethod {
  id: number
  code: string
  label: string
  is_active: boolean
  surcharge_pct: number
}

export interface InscripcionAdmin {
  id: string
  nombre_piloto: string
  email: string
  modalidad: string
  metodo_pago: string
  status: string
  monto_cop: number
  created_at: string
}

export interface Theme {
  primary_color: string
  secondary_color: string
  font_family: string
  logo_url?: string
  favicon_url?: string
}
