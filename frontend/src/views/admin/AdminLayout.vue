<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAdminAuthStore } from '../../stores/admin/auth'

const router = useRouter()
const auth = useAdminAuthStore()

const navItems = [
  { label: 'Dashboard', icon: 'pi pi-home', to: '/admin/dashboard' },
  { label: 'Landing', icon: 'pi pi-pencil', to: '/admin/landing' },
  { label: 'Media', icon: 'pi pi-images', to: '/admin/media' },
  { label: 'Formulario', icon: 'pi pi-calendar', to: '/admin/form' },
  { label: 'Inscripciones', icon: 'pi pi-users', to: '/admin/inscripciones' },
  { label: 'Tema', icon: 'pi pi-palette', to: '/admin/theme' },
]

function logout() {
  auth.logout()
  router.push('/admin/login')
}
</script>

<template>
  <div class="flex h-screen bg-gray-100">
    <!-- Sidebar -->
    <aside class="w-64 bg-gray-900 text-white flex flex-col">
      <div class="p-6 border-b border-gray-700">
        <p class="text-xs text-gray-400 uppercase tracking-wider">Admin Panel</p>
        <p class="font-bold text-lg">Scudería St4ge</p>
      </div>
      <nav class="flex-1 p-4 space-y-1">
        <router-link v-for="item in navItems" :key="item.to" :to="item.to"
          class="flex items-center gap-3 px-4 py-2.5 rounded-lg text-gray-300 hover:bg-gray-700 hover:text-white transition-colors"
          active-class="bg-blue-600 text-white">
          <i :class="item.icon" class="text-sm" />
          {{ item.label }}
        </router-link>
      </nav>
      <div class="p-4 border-t border-gray-700">
        <button @click="logout" class="flex items-center gap-2 text-gray-400 hover:text-white text-sm w-full">
          <i class="pi pi-sign-out" />
          Cerrar sesión
        </button>
      </div>
    </aside>
    <!-- Main -->
    <main class="flex-1 overflow-auto">
      <router-view />
    </main>
  </div>
</template>
