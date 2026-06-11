<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAdminAuthStore } from '../../stores/admin/auth'

const router = useRouter()
const route = useRoute()
const auth = useAdminAuthStore()

const navItems = [
  { label: 'Dashboard', to: '/admin/dashboard', icon: 'home' },
  { label: 'Landing CMS', to: '/admin/landing', icon: 'layout' },
  { label: 'Imágenes', to: '/admin/media', icon: 'image' },
  { label: 'Formulario', to: '/admin/form', icon: 'calendar' },
  { label: 'Inscripciones', to: '/admin/inscripciones', icon: 'users' },
]

const currentPage = computed(() => {
  const item = navItems.find(n => route.path.startsWith(n.to))
  return item?.label ?? 'Admin'
})

function logout() {
  auth.logout()
  router.push('/admin/login')
}
</script>

<template>
  <div class="admin-shell flex h-screen bg-gray-50 overflow-hidden">
    <!-- Sidebar -->
    <aside class="w-60 bg-white border-r border-gray-200 flex flex-col shrink-0">
      <!-- Brand -->
      <div class="px-5 py-5 border-b border-gray-100">
        <div class="flex items-center gap-3">
          <div class="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center shrink-0">
            <svg class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <div class="min-w-0">
            <p class="text-sm font-bold text-gray-900 truncate">Scudería St4ge</p>
            <p class="text-xs text-gray-400">Admin Panel</p>
          </div>
        </div>
      </div>

      <!-- Nav -->
      <nav class="flex-1 px-3 py-4 space-y-0.5 overflow-y-auto">
        <router-link
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors text-gray-600 hover:bg-gray-100 hover:text-gray-900"
          active-class="bg-blue-50 text-blue-700 hover:bg-blue-50 hover:text-blue-700"
        >
          <!-- Home -->
          <svg v-if="item.icon === 'home'" class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
          </svg>
          <!-- Layout -->
          <svg v-else-if="item.icon === 'layout'" class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
          </svg>
          <!-- Image -->
          <svg v-else-if="item.icon === 'image'" class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <!-- Calendar -->
          <svg v-else-if="item.icon === 'calendar'" class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <!-- Users -->
          <svg v-else-if="item.icon === 'users'" class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
          </svg>
          {{ item.label }}
        </router-link>
      </nav>

      <!-- User / Logout -->
      <div class="px-3 py-4 border-t border-gray-100">
        <div class="flex items-center gap-3 px-3 py-2 mb-1">
          <div class="w-7 h-7 rounded-full bg-blue-100 flex items-center justify-center shrink-0">
            <span class="text-xs font-bold text-blue-700 uppercase">
              {{ auth.user?.username?.charAt(0) ?? 'A' }}
            </span>
          </div>
          <p class="text-sm font-medium text-gray-700 truncate">{{ auth.user?.username ?? 'Admin' }}</p>
        </div>
        <button
          @click="logout"
          class="flex items-center gap-2 w-full px-3 py-2 rounded-lg text-sm text-gray-500 hover:bg-gray-100 hover:text-gray-700 transition-colors"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
          </svg>
          Cerrar sesión
        </button>
      </div>
    </aside>

    <!-- Main -->
    <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
      <!-- Top bar -->
      <header class="bg-white border-b border-gray-200 px-6 py-3.5 flex items-center shrink-0">
        <h2 class="text-sm font-semibold text-gray-900">{{ currentPage }}</h2>
      </header>

      <!-- Content -->
      <main class="flex-1 overflow-y-auto">
        <router-view />
      </main>
    </div>
  </div>
</template>
