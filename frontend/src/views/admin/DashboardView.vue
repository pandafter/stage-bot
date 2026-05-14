<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { inscripcionesAdminService } from '../../services/admin'

const stats = ref({ total: 0, pendientes: 0, pagadas: 0, rechazadas: 0 })
const loading = ref(true)

onMounted(async () => {
  try {
    const [all, pend, paid, rejected] = await Promise.all([
      inscripcionesAdminService.list({ limit: 1 }),
      inscripcionesAdminService.list({ status: 'pendiente', limit: 1 }),
      inscripcionesAdminService.list({ status: 'pagado', limit: 1 }),
      inscripcionesAdminService.list({ status: 'rechazado', limit: 1 }),
    ])
    stats.value = {
      total: all.total,
      pendientes: pend.total,
      pagadas: paid.total,
      rechazadas: rejected.total,
    }
  } finally {
    loading.value = false
  }
})

const cards = [
  {
    key: 'total',
    label: 'Total inscripciones',
    color: 'border-blue-500',
    bg: 'bg-blue-50',
    text: 'text-blue-600',
    icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2',
  },
  {
    key: 'pendientes',
    label: 'Pendientes de pago',
    color: 'border-yellow-500',
    bg: 'bg-yellow-50',
    text: 'text-yellow-600',
    icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z',
  },
  {
    key: 'pagadas',
    label: 'Confirmadas / Pagadas',
    color: 'border-green-500',
    bg: 'bg-green-50',
    text: 'text-green-600',
    icon: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z',
  },
  {
    key: 'rechazadas',
    label: 'Rechazadas',
    color: 'border-red-500',
    bg: 'bg-red-50',
    text: 'text-red-600',
    icon: 'M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z',
  },
]

const quickLinks = [
  { label: 'Ver inscripciones', to: '/admin/inscripciones', color: 'bg-blue-600 hover:bg-blue-700' },
  { label: 'Editar landing', to: '/admin/landing', color: 'bg-violet-600 hover:bg-violet-700' },
  { label: 'Gestionar fechas', to: '/admin/form', color: 'bg-emerald-600 hover:bg-emerald-700' },
  { label: 'Biblioteca de imágenes', to: '/admin/media', color: 'bg-orange-500 hover:bg-orange-600' },
]
</script>

<template>
  <div class="p-6 lg:p-8 max-w-5xl">
    <div class="mb-8">
      <h1 class="text-2xl font-bold text-gray-900">Dashboard</h1>
      <p class="text-sm text-gray-500 mt-1">Resumen general de la plataforma</p>
    </div>

    <!-- Stats -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      <div
        v-for="card in cards"
        :key="card.key"
        class="bg-white rounded-xl border border-gray-200 shadow-sm p-5 flex flex-col gap-3"
      >
        <div class="flex items-center justify-between">
          <p class="text-sm font-medium text-gray-500">{{ card.label }}</p>
          <div class="w-8 h-8 rounded-lg flex items-center justify-center" :class="card.bg">
            <svg class="w-4 h-4" :class="card.text" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="card.icon" />
            </svg>
          </div>
        </div>
        <div>
          <p v-if="loading" class="text-3xl font-bold text-gray-200 animate-pulse">—</p>
          <p v-else class="text-3xl font-bold text-gray-900">
            {{ stats[card.key as keyof typeof stats] }}
          </p>
        </div>
        <div class="h-1 rounded-full w-full" :class="card.bg">
          <div class="h-1 rounded-full" :class="card.color.replace('border-', 'bg-')"
            :style="{ width: loading ? '0%' : '100%', transition: 'width 0.8s ease' }" />
        </div>
      </div>
    </div>

    <!-- Acciones rápidas -->
    <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
      <h2 class="text-sm font-semibold text-gray-700 mb-4">Acciones rápidas</h2>
      <div class="flex flex-wrap gap-3">
        <router-link
          v-for="link in quickLinks"
          :key="link.to"
          :to="link.to"
          class="px-4 py-2 rounded-lg text-sm font-medium text-white transition-colors"
          :class="link.color"
        >
          {{ link.label }}
        </router-link>
      </div>
    </div>
  </div>
</template>
