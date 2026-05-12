import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { AdminUser, LoginResponse } from '../../types/admin'

export const useAdminAuthStore = defineStore('adminAuth', () => {
  const token = ref<string | null>(localStorage.getItem('admin_token'))
  const user = ref<AdminUser | null>(null)

  const isAuthenticated = computed(() => !!token.value)

  function setAuth(response: LoginResponse) {
    token.value = response.token
    user.value = response.user
    localStorage.setItem('admin_token', response.token)
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem('admin_token')
  }

  return { token, user, isAuthenticated, setAuth, logout }
})
