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
  { label: 'Todos', value: '' },
  { label: 'Pendiente', value: 'pendiente' },
  { label: 'Pagado', value: 'pagado' },
  { label: 'Comprobante recibido', value: 'comprobante_pendiente' },
  { label: 'Rechazado', value: 'rechazado' },
  { label: 'Cancelado', value: 'cancelado' },
]

const statusColors: Record<string, string> = {
  pendiente: 'bg-yellow-100 text-yellow-800',
  pagado: 'bg-green-100 text-green-800',
  comprobante_pendiente: 'bg-blue-100 text-blue-800',
  rechazado: 'bg-red-100 text-red-800',
  cancelado: 'bg-gray-100 text-gray-800',
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
  <div class="p-6">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold">Inscripciones</h1>
        <p class="text-sm text-gray-500 mt-1">{{ store.total }} en total</p>
      </div>
      <button @click="exportCsv"
        class="flex items-center gap-2 px-4 py-2 border rounded-lg text-sm font-medium hover:bg-gray-50">
        <i class="pi pi-download" />
        Exportar CSV
      </button>
    </div>

    <!-- Filtros -->
    <div class="bg-white rounded-xl shadow-sm border p-4 mb-4 flex flex-wrap gap-3 items-end">
      <div>
        <label class="block text-xs font-medium text-gray-500 mb-1">Estado</label>
        <select :value="store.filters.status ?? ''"
          @change="store.setFilter('status', ($event.target as HTMLSelectElement).value)"
          class="border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
          <option v-for="opt in statusOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
      </div>
      <div>
        <label class="block text-xs font-medium text-gray-500 mb-1">Desde</label>
        <input type="date" :value="store.filters.date_from ?? ''"
          @change="store.setFilter('date_from', ($event.target as HTMLInputElement).value)"
          class="border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
      </div>
      <div>
        <label class="block text-xs font-medium text-gray-500 mb-1">Hasta</label>
        <input type="date" :value="store.filters.date_to ?? ''"
          @change="store.setFilter('date_to', ($event.target as HTMLInputElement).value)"
          class="border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
      </div>
      <div class="flex-1 min-w-48">
        <label class="block text-xs font-medium text-gray-500 mb-1">Buscar</label>
        <input type="text" placeholder="Email, nombre o documento..."
          :value="store.filters.search ?? ''"
          @input="store.setFilter('search', ($event.target as HTMLInputElement).value)"
          class="w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
      </div>
      <button @click="store.resetFilters" class="px-3 py-2 text-sm text-gray-500 hover:text-gray-700 underline">
        Limpiar
      </button>
    </div>

    <!-- Tabla -->
    <div class="bg-white rounded-xl shadow-sm border overflow-hidden">
      <div v-if="store.loading" class="flex items-center justify-center h-40 text-gray-400">
        Cargando...
      </div>
      <table v-else class="w-full text-sm">
        <thead class="bg-gray-50 border-b">
          <tr>
            <th class="px-4 py-3 text-left font-medium text-gray-600">Piloto</th>
            <th class="px-4 py-3 text-left font-medium text-gray-600">Plan</th>
            <th class="px-4 py-3 text-left font-medium text-gray-600">Monto</th>
            <th class="px-4 py-3 text-left font-medium text-gray-600">Fecha curso</th>
            <th class="px-4 py-3 text-left font-medium text-gray-600">Estado</th>
            <th class="px-4 py-3 text-left font-medium text-gray-600">Creado</th>
            <th class="px-4 py-3" />
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-if="store.items.length === 0">
            <td colspan="7" class="px-4 py-12 text-center text-gray-400">No hay inscripciones que coincidan con los filtros.</td>
          </tr>
          <tr v-for="ins in store.items" :key="ins.id"
            class="hover:bg-gray-50 cursor-pointer"
            @click="router.push(`/admin/inscripciones/${ins.id}`)">
            <td class="px-4 py-3">
              <p class="font-medium">{{ ins.nombre_piloto }}</p>
              <p class="text-gray-400 text-xs">{{ ins.email }}</p>
            </td>
            <td class="px-4 py-3 text-gray-700">{{ ins.plan }}</td>
            <td class="px-4 py-3 font-medium">{{ formatCOP(ins.monto_cop) }}</td>
            <td class="px-4 py-3 text-gray-600">{{ ins.fecha_curso }}</td>
            <td class="px-4 py-3">
              <span class="px-2 py-1 rounded-full text-xs font-medium"
                :class="statusColors[ins.status] ?? 'bg-gray-100 text-gray-600'">
                {{ ins.status }}
              </span>
            </td>
            <td class="px-4 py-3 text-gray-400">{{ formatDate(ins.created_at) }}</td>
            <td class="px-4 py-3 text-right">
              <i class="pi pi-chevron-right text-gray-300" />
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Paginacion -->
      <div v-if="totalPages > 1" class="flex items-center justify-between px-4 py-3 border-t bg-gray-50">
        <p class="text-sm text-gray-500">
          Pagina {{ store.currentPage }} de {{ totalPages }}
        </p>
        <div class="flex gap-2">
          <button :disabled="store.currentPage <= 1"
            @click="store.currentPage--"
            class="px-3 py-1 border rounded text-sm disabled:opacity-40 hover:bg-white">
            &larr; Anterior
          </button>
          <button :disabled="store.currentPage >= totalPages"
            @click="store.currentPage++"
            class="px-3 py-1 border rounded text-sm disabled:opacity-40 hover:bg-white">
            Siguiente &rarr;
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
