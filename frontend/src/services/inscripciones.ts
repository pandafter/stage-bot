import { api } from './api'
import type {
  ConfigResponse,
  CreateInscripcionRequest,
  CreateInscripcionResponse,
  InscripcionStatusResponse,
} from '@/types/api'

export const fetchConfig = () =>
  api.get<ConfigResponse>('/config').then((r) => r.data)

export const createInscripcion = (req: CreateInscripcionRequest) =>
  api.post<CreateInscripcionResponse>('/inscripciones', req).then((r) => r.data)

export const getInscripcion = (id: string) =>
  api.get<InscripcionStatusResponse>(`/inscripciones/${id}`).then((r) => r.data)

export const uploadComprobante = (id: string, file: File) => {
  const fd = new FormData()
  fd.append('file', file)
  return api
    .post(`/inscripciones/${id}/comprobante`, fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data)
}
