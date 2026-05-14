<script setup lang="ts">
import { onMounted, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useInscripcionesAdminStore } from '../../stores/admin/inscripciones'
import { inscripcionesAdminService } from '../../services/admin'

const router = useRouter()
const store = useInscripcionesAdminStore()

onMounted(() => store.load())
watch([() => store.filters, () => store.currentPage], () => store.load(), { deep: true })

const statusOptions = [
  { label: 'Todos los estados', value: '' },
  { label: 'Pendiente', value: 'pendiente' },
  { label: 'Pagado', value: 'pagado' },
  { label: 'Comprobante recibido', value: 'comprobante_pendiente' },
  { label: 'Rechazado', value: 'rechazado' },
  { label: 'Cancelado', value: 'cancelado' },
]

const statusConfig: Record<string, { label: string; classes: string }> = {
  pendiente: { label: 'Pendiente', classes: 'bg-yellow-50 text-yellow-700 ring-1 ring-yellow-200' },
  pagado: { label: 'Pagado', classes: 'bg-green-50 text-green-700 ring-1 ring-green-200' },
  comprobante_pendiente: { label: 'Comprobante', classes: 'bg-blue-50 text-blue-700 ring-1 ring-blue-200' },
  rechazado: { label: 'Rechazado', classes: 'bg-red-50 text-red-700 ring-1 ring-red-200' },
  cancelado: { label: 'Cancelado', classes: 'bg-gray-100 text-gray-600 ring-1 ring-gray-200' },
}

const totalPages = computed(() => Math.ceil(store.total / (store.filters.limit ?? 20)))

async function exportCsv() {
  const blob = await inscripcionesAdminService.exportCsv()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `inscripciones_${new Date().toISOString().split('T')[0]}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('es-CO', { day: '2-digit', month: 'short', year: 'numeric' })
}

function formatCOP(amount: number) {
  return new Intl.NumberFormat('es-CO', { style: 'currency', currency: 'COP', maximumFractionDigits: 0 }).format(amount)
}
</script>

<template>
  <div class="p-6 lg:p-8">
    <!-- Header -->
    <div class="flex items-start justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Inscripciones</h1>
        <p class="text-sm text-gray-500 mt-1">
          <span v-if="!store.loading">{{ store.total }} registro{{ store.total !== 1 ? 's' : '' }}</span>
          <span v-else class="inline-block w-12 h-3.5 bg-gray-200 rounded animate-pulse" />
        </p>
      </div>
      <button
        @click="exportCsv"
        class="flex items-center gap-2 px-4 py-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 transition-colors"
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
        Exportar CSV
      </button>
    </div>

    <!-- Filtros -->
    <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-4 mb-4">
      <div class="flex flex-wrap gap-3 items-end">
        <div>
          <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">Estado</label>
          <select
            :value="store.filters.status ?? ''"
            @change="store.setFilter('status', ($event.target as HTMLSelectElement).value)"
            class="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white"
          >
            <option v-for="opt in statusOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
          </select>
        </div>
        <div>
          <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">Desde</label>
          <input
            type="date"
            :value="store.filters.date_from ?? ''"
            @change="store.setFilter('date_from', ($event.target as HTMLInputElement).value)"
            class="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div>
          <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">Hasta</label>
          <input
            type="date"
            :value="store.filters.date_to ?? ''"
            @change="store.setFilter('date_to', ($event.target as HTMLInputElement).value)"
            class="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div class="flex-1 min-w-52">
          <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">Buscar</label>
          <div class="relative">
            <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              type="text"
              placeholder="Email, nombre o documento..."
              :value="store.filters.search ?? ''"
              @input="store.setFilter('search', ($event.target as HTMLInputElement).value)"
              class="w-full border border-gray-300 rounded-lg pl-9 pr-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>
        <button
          @click="store.resetFilters"
          class="px-3 py-2 text-sm text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
        >
          Limpiar filtros
        </button>
      </div>
    </div>

    <!-- Tabla -->
    <div class="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
      <!-- Loading skeleton -->
      <div v-if="store.loading">
        <div class="px-5 py-4 border-b border-gray-100 bg-gray-50">
          <div class="grid grid-cols-6 gap-4">
            <div v-for="i in 6" :key="i" class="h-3 bg-gray-200 rounded animate-pulse" />
          </div>
        </div>
        <div v-for="i in 5" :key="i" class="px-5 py-4 border-b border-gray-100 last:border-0">
          <div class="grid grid-cols-6 gap-4 items-center">
            <div class="space-y-1.5">
              <div class="h-3.5 bg-gray-200 rounded animate-pulse w-3/4" />
              <div class="h-2.5 bg-gray-100 rounded animate-pulse w-1/2" />
            </div>
            <div v-for="j in 5" :key="j" class="h-3 bg-gray-100 rounded animate-pulse" />
          </div>
        </div>
      </div>

      <table v-else class="w-full text-sm">
        <thead>
          <tr class="bg-gray-50 border-b border-gray-200">
            <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Piloto</th>
            <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Plan</th>
            <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Monto</th>
            <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Fecha curso</th>
            <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Estado</th>
            <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Registrado</th>
            <th class="px-5 py-3.5 w-8" />
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-if="store.items.length === 0">
            <td colspan="7" class="px-5 py-16 text-center">
              <svg class="w-10 h-10 text-gray-300 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                  d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
              <p class="text-gray-400 text-sm">No hay inscripciones que coincidan con los filtros.</p>
            </td>
          </tr>
          <tr
            v-for="ins in store.items"
            :key="ins.id"
            class="hover:bg-gray-50 cursor-pointer transition-colors"
            @click="router.push(`/admin/inscripciones/${ins.id}`)"
          >
            <td class="px-5 py-3.5">
              <p class="font-semibold text-gray-900">{{ ins.nombre_piloto }}</p>
              <p class="text-xs text-gray-400 mt-0.5">{{ ins.email }}</p>
            </td>
            <td class="px-5 py-3.5 text-gray-600">{{ ins.plan }}</td>
            <td class="px-5 py-3.5 font-semibold text-gray-900">{{ formatCOP(ins.monto_cop) }}</td>
            <td class="px-5 py-3.5 text-gray-500">{{ ins.fecha_curso }}</td>
            <td class="px-5 py-3.5">
              <span
                class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold"
                :class="statusConfig[ins.status]?.classes ?? 'bg-gray-100 text-gray-600'"
              >
                {{ statusConfig[ins.status]?.label ?? ins.status }}
              </span>
            </td>
            <td class="px-5 py-3.5 text-gray-400 text-xs">{{ formatDate(ins.created_at) }}</td>
            <td class="px-5 py-3.5 text-right">
              <svg class="w-4 h-4 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Paginación -->
      <div v-if="totalPages > 1" class="flex items-center justify-between px-5 py-3.5 border-t border-gray-100 bg-gray-50">
        <p class="text-sm text-gray-500">
          Página <span class="font-medium text-gray-700">{{ store.currentPage }}</span> de
          <span class="font-medium text-gray-700">{{ totalPages }}</span>
        </p>
        <div class="flex gap-2">
          <button
            :disabled="store.currentPage <= 1"
            @click="store.currentPage--"
            class="flex items-center gap-1.5 px-3 py-1.5 border border-gray-300 rounded-lg text-sm font-medium disabled:opacity-40 hover:bg-white transition-colors bg-white"
          >
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
            Anterior
          </button>
          <button
            :disabled="store.currentPage >= totalPages"
            @click="store.currentPage++"
            class="flex items-center gap-1.5 px-3 py-1.5 border border-gray-300 rounded-lg text-sm font-medium disabled:opacity-40 hover:bg-white transition-colors bg-white"
          >
            Siguiente
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
