<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import adminApi from '../../services/admin'

interface FechaCount { fecha: string; count: number }
interface Stats {
  total: number
  pendientes: number
  pagadas: number
  rechazadas: number
  monto_recaudado: number
  por_fecha: FechaCount[]
}

const stats = ref<Stats>({ total: 0, pendientes: 0, pagadas: 0, rechazadas: 0, monto_recaudado: 0, por_fecha: [] })
const loading = ref(true)

onMounted(async () => {
  try {
    const res = await adminApi.get<Stats>('/inscripciones/stats')
    stats.value = res.data
  } finally {
    loading.value = false
  }
})

function formatCOP(n: number) {
  return new Intl.NumberFormat('es-CO', { style: 'currency', currency: 'COP', maximumFractionDigits: 0 }).format(n)
}

const totalInscripciones = computed(() => stats.value.total)

const cards = [
  {
    key: 'total' as const,
    label: 'Total inscripciones',
    color: 'border-blue-500',
    bg: 'bg-blue-50',
    text: 'text-blue-600',
    icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2',
  },
  {
    key: 'pendientes' as const,
    label: 'Pendientes de pago',
    color: 'border-yellow-500',
    bg: 'bg-yellow-50',
    text: 'text-yellow-600',
    icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z',
  },
  {
    key: 'pagadas' as const,
    label: 'Confirmadas / Pagadas',
    color: 'border-green-500',
    bg: 'bg-green-50',
    text: 'text-green-600',
    icon: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z',
  },
  {
    key: 'rechazadas' as const,
    label: 'Rechazadas / Anuladas',
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
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
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
            {{ stats[card.key] }}
          </p>
        </div>
        <div class="h-1 rounded-full w-full" :class="card.bg">
          <div class="h-1 rounded-full" :class="card.color.replace('border-', 'bg-')"
            :style="{ width: loading ? '0%' : '100%', transition: 'width 0.8s ease' }" />
        </div>
      </div>
    </div>

    <!-- Monto recaudado -->
    <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-5 mb-6 flex items-center justify-between">
      <div>
        <p class="text-sm font-medium text-gray-500">Monto recaudado (pagos confirmados)</p>
        <p v-if="loading" class="text-3xl font-bold text-gray-200 animate-pulse mt-1">—</p>
        <p v-else class="text-3xl font-bold text-emerald-600 mt-1">{{ formatCOP(stats.monto_recaudado) }}</p>
      </div>
      <div class="w-12 h-12 rounded-xl bg-emerald-50 flex items-center justify-center">
        <svg class="w-6 h-6 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </div>
    </div>

    <!-- Por fecha -->
    <div v-if="!loading && stats.por_fecha.length > 0" class="bg-white rounded-xl border border-gray-200 shadow-sm p-5 mb-6">
      <h2 class="text-sm font-semibold text-gray-700 mb-4">Inscripciones por fecha de curso</h2>
      <div class="space-y-3">
        <div v-for="fc in stats.por_fecha" :key="fc.fecha" class="flex items-center gap-4">
          <span class="text-sm text-gray-700 w-40 truncate font-medium">{{ fc.fecha }}</span>
          <div class="flex-1 bg-gray-100 rounded-full h-2.5">
            <div
              class="bg-blue-500 h-2.5 rounded-full transition-all duration-700"
              :style="{ width: totalInscripciones > 0 ? (fc.count / totalInscripciones * 100) + '%' : '0%' }"
            />
          </div>
          <span class="text-sm font-semibold text-gray-900 w-8 text-right">{{ fc.count }}</span>
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
