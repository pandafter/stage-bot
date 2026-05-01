export type ModalidadId = 'reserva' | 'completo'
export type MetodoPagoId = 'tarjeta' | 'nequi' | 'bancolombia' | 'pse' | 'transferencia'
export type Status =
  | 'pendiente'
  | 'comprobante recibido, en validación'
  | 'pagado'
  | 'pago rechazado'
  | 'pago anulado'

export interface Modalidad {
  id: ModalidadId
  label: string
  price_cop: number
}

export interface Metodo {
  id: MetodoPagoId
  label: string
}

export interface ConfigResponse {
  modalidades: Modalidad[]
  metodos: Metodo[]
  fechas: string[]
  card_surcharge_pct: number
  reserva_cop: number
  precio_completo: number
  precio_descuento: number
}

export interface CreateInscripcionRequest {
  email: string
  nombre_piloto: string
  edad: number
  tipo_documento: string
  numero_documento: string
  telefono: string
  ciudad: string
  eps: string
  grupo_sanguineo: string
  familiar_nombre: string
  familiar_telefono: string
  instagram_user?: string
  modalidad: ModalidadId
  metodo_pago: MetodoPagoId
  fecha_curso: string
}

export interface CreateInscripcionResponse {
  id: string
  status: Status
  monto_cop: number
  checkout_url?: string
  requires_comprobante: boolean
}

export interface InscripcionStatusResponse {
  id: string
  status: Status
  monto_cop: number
  fecha_curso: string
  metodo_pago: MetodoPagoId
  modalidad: ModalidadId
  nombre_piloto: string
}
