import axios from 'axios'

export const api = axios.create({
  baseURL: '/api',
  timeout: 20000,
})

api.interceptors.response.use(
  (r) => r,
  (err) => {
    console.error('[api]', err.response?.data || err.message)
    return Promise.reject(err)
  },
)
