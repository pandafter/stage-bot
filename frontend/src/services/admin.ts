import axios from 'axios'
import type { LoginRequest, LoginResponse } from '../types/admin'

const adminApi = axios.create({
  baseURL: '/api/admin',
})

adminApi.interceptors.request.use((config) => {
  const token = localStorage.getItem('admin_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

adminApi.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('admin_token')
      window.location.href = '/admin/login'
    }
    return Promise.reject(error)
  }
)

export const adminAuthService = {
  login: (data: LoginRequest) =>
    adminApi.post<LoginResponse>('/auth/login', data).then(r => r.data),
  me: () => adminApi.get('/me').then(r => r.data),
}

export const cmsService = {
  getSections: () => adminApi.get('/cms/sections').then(r => r.data),
  getSection: (key: string) => adminApi.get(`/cms/sections/${key}`).then(r => r.data),
  updateSection: (key: string, data: Record<string, unknown>, isPublished = true) =>
    adminApi.put(`/cms/sections/${key}`, { data, is_published: isPublished }).then(r => r.data),
}

export const mediaService = {
  list: (params?: { tag?: string; page?: number }) =>
    adminApi.get('/media', { params }).then(r => r.data),
  upload: (file: File, altText = '') => {
    const form = new FormData()
    form.append('file', file)
    form.append('alt_text', altText)
    return adminApi.post('/media', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }).then(r => r.data)
  },
  delete: (id: number) => adminApi.delete(`/media/${id}`),
  update: (id: number, data: { alt_text?: string; tags?: string[] }) =>
    adminApi.patch(`/media/${id}`, data).then(r => r.data),
}

export const formConfigService = {
  listDates: () => adminApi.get('/form/dates').then(r => r.data),
  createDate: (data: Record<string, unknown>) => adminApi.post('/form/dates', data).then(r => r.data),
  updateDate: (id: number, data: Record<string, unknown>) => adminApi.patch(`/form/dates/${id}`, data).then(r => r.data),
  deleteDate: (id: number) => adminApi.delete(`/form/dates/${id}`),
  reorderDates: (ids: number[]) => adminApi.post('/form/dates/reorder', { ids }).then(r => r.data),

  listPlans: () => adminApi.get('/form/plans').then(r => r.data),
  createPlan: (data: Record<string, unknown>) => adminApi.post('/form/plans', data).then(r => r.data),
  updatePlan: (id: number, data: Record<string, unknown>) => adminApi.patch(`/form/plans/${id}`, data).then(r => r.data),
  deletePlan: (id: number) => adminApi.delete(`/form/plans/${id}`),

  listMethods: () => adminApi.get('/form/methods').then(r => r.data),
  updateMethod: (id: number, data: Record<string, unknown>) => adminApi.patch(`/form/methods/${id}`, data).then(r => r.data),
}

export const inscripcionesAdminService = {
  list: (params?: Record<string, unknown>) => adminApi.get('/inscripciones', { params }).then(r => r.data),
  get: (id: string) => adminApi.get(`/inscripciones/${id}`).then(r => r.data),
  updateStatus: (id: string, status: string, note?: string) =>
    adminApi.patch(`/inscripciones/${id}/status`, { status, note }).then(r => r.data),
  exportCsv: () => adminApi.get('/inscripciones/export.csv', { responseType: 'blob' }).then(r => r.data),
}

export const themeService = {
  get: () => adminApi.get('/theme').then(r => r.data),
  update: (theme: Record<string, unknown>) => adminApi.put('/theme', theme).then(r => r.data),
}

export default adminApi
