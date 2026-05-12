<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminAuthStore } from '../../stores/admin/auth'
import { adminAuthService } from '../../services/admin'

const router = useRouter()
const auth = useAdminAuthStore()
const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function login() {
  loading.value = true
  error.value = ''
  try {
    const res = await adminAuthService.login({ username: username.value, password: password.value })
    auth.setAuth(res)
    router.push('/admin/dashboard')
  } catch {
    error.value = 'Usuario o contraseña incorrectos'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-100">
    <div class="bg-white rounded-xl shadow-lg p-8 w-full max-w-sm">
      <h1 class="text-2xl font-bold mb-6 text-center">Admin — Scudería St4ge</h1>
      <form @submit.prevent="login" class="space-y-4">
        <div>
          <label class="block text-sm font-medium mb-1">Usuario</label>
          <input v-model="username" type="text" required
            class="w-full border rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Contraseña</label>
          <input v-model="password" type="password" required
            class="w-full border rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <p v-if="error" class="text-red-500 text-sm">{{ error }}</p>
        <button type="submit" :disabled="loading"
          class="w-full bg-blue-600 text-white py-2 rounded-lg font-semibold hover:bg-blue-700 disabled:opacity-50">
          {{ loading ? 'Ingresando...' : 'Ingresar' }}
        </button>
      </form>
    </div>
  </div>
</template>
