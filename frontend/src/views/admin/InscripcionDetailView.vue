<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { inscripcionesAdminService } from '../../services/admin'
import type { AdminInscripcion } from '../../types/admin'

const route = useRoute()
const router = useRouter()

const ins = ref<AdminInscripcion | null>(null)
const loading = ref(true)
const saving = ref(false)
const newStatus = ref('')
const note = ref('')
const saved = ref(false)

const statusOptions = [
  { label: 'Pendiente', value: 'pendiente' },
  { label: 'Pagado / Confirmado', value: 'pagado' },
  { label: 'Comprobante recibido', value: 'comprobante_pendiente' },
  { label: 'Rechazado', value: 'rechazado' },
  { label: 'Cancelado', value: 'cancelado' },
]

const statusConfig: Record<string, { label: string; classes: string }> = {
  pendiente: { label: 'Pendiente', classes: 'bg-yellow-50 text-yellow-700 ring-1 ring-yellow-200' },
  pagado: { label: 'Pagado', classes: 'bg-green-50 text-green-700 ring-1 ring-green-200' },
  comprobante_pendiente: { label: 'Comprobante recibido', classes: 'bg-blue-50 text-blue-700 ring-1 ring-blue-200' },
  rechazado: { label: 'Rechazado', classes: 'bg-red-50 text-red-700 ring-1 ring-red-200' },
  cancelado: { label: 'Cancelado', classes: 'bg-gray-100 text-gray-600 ring-1 ring-gray-200' },
}

onMounted(async () => {
  try {
    ins.value = await inscripcionesAdminService.get(route.params.id as string)
    newStatus.value = ins.value?.status ?? 'pendiente'
  } finally {
    loading.value = false
  }
})

async function updateStatus() {
  if (!ins.value) return
  saving.value = true
  try {
    await inscripcionesAdminService.updateStatus(ins.value.id, newStatus.value, note.value)
    ins.value = { ...ins.value, status: newStatus.value }
    saved.value = true
    setTimeout(() => { saved.value = false }, 2000)
  } finally {
    saving.value = false
  }
}

function formatCOP(amount: number) {
  return new Intl.NumberFormat('es-CO', { style: 'currency', currency: 'COP', maximumFractionDigits: 0 }).format(amount)
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString('es-CO', {
    day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit'
  })
}

type InfoRow = { label: string; value: string | number | undefined; highlight?: boolean }

function infoRows(): InfoRow[] {
  if (!ins.value) return []
  return [
    { label: 'Plan', value: ins.value.plan, highlight: true },
    { label: 'Monto', value: formatCOP(ins.value.monto_cop), highlight: true },
    { label: 'Fecha del curso', value: ins.value.fecha_curso },
    { label: 'Método de pago', value: ins.value.metodo_pago },
    { label: 'Teléfono', value: ins.value.telefono },
    { label: 'Ciudad', value: ins.value.ciudad },
    { label: 'Tipo de documento', value: ins.value.tipo_documento },
    { label: 'Número de documento', value: ins.value.numero_documento },
    { label: 'EPS', value: ins.value.eps },
    { label: 'Grupo sanguíneo', value: ins.value.grupo_sanguineo },
    { label: 'Familiar (nombre)', value: ins.value.familiar_nombre },
    { label: 'Familiar (teléfono)', value: ins.value.familiar_telefono },
    { label: 'Instagram', value: ins.value.instagram_user },
    { label: 'Registrado', value: formatDate(ins.value.created_at) },
  ]
}
</script>

<template>
  <div class="p-6 lg:p-8">
    <!-- Back -->
    <button
      @click="router.back()"
      class="flex items-center gap-2 text-sm text-gray-500 hover:text-gray-700 mb-6 transition-colors"
    >
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
      </svg>
      Volver a inscripciones
    </button>

    <!-- Loading -->
    <div v-if="loading" class="space-y-4">
      <div class="bg-white rounded-xl border border-gray-200 p-6">
        <div class="flex items-center gap-4 mb-6">
          <div class="w-12 h-12 rounded-xl bg-gray-200 animate-pulse" />
          <div class="space-y-2">
            <div class="h-5 w-48 bg-gray-200 rounded animate-pulse" />
            <div class="h-3.5 w-32 bg-gray-100 rounded animate-pulse" />
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div v-for="i in 8" :key="i" class="space-y-1.5">
            <div class="h-2.5 w-20 bg-gray-100 rounded animate-pulse" />
            <div class="h-4 w-full bg-gray-200 rounded animate-pulse" />
          </div>
        </div>
      </div>
    </div>

    <div v-else-if="ins" class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Info principal -->
      <div class="lg:col-span-2 space-y-5">
        <!-- Header card -->
        <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
          <div class="flex items-start justify-between mb-6">
            <div class="flex items-center gap-4">
              <div class="w-12 h-12 rounded-xl bg-blue-100 flex items-center justify-center shrink-0">
                <span class="text-lg font-bold text-blue-600 uppercase">
                  {{ ins.nombre_piloto.charAt(0) }}
                </span>
              </div>
              <div>
                <h1 class="text-xl font-bold text-gray-900">{{ ins.nombre_piloto }}</h1>
                <p class="text-gray-400 text-sm mt-0.5">{{ ins.email }}</p>
              </div>
            </div>
            <span
              class="inline-flex items-center px-3 py-1 rounded-full text-sm font-semibold"
              :class="statusConfig[ins.status]?.classes ?? 'bg-gray-100 text-gray-600'"
            >
              {{ statusConfig[ins.status]?.label ?? ins.status }}
            </span>
          </div>

          <div class="grid grid-cols-2 gap-x-8 gap-y-4">
            <div
              v-for="row in infoRows()"
              :key="row.label"
              class="py-2 border-b border-gray-100 last:border-0"
            >
              <p class="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-0.5">{{ row.label }}</p>
              <p class="text-sm font-medium text-gray-900" :class="row.highlight ? 'text-base' : ''">
                {{ row.value || '—' }}
              </p>
            </div>
          </div>
        </div>

        <!-- Comprobante -->
        <div v-if="ins.comprobante_path" class="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
          <h3 class="text-sm font-semibold text-gray-700 mb-3">Comprobante de pago</h3>
          <a
            :href="`/uploads/${ins.comprobante_path}`"
            target="_blank"
            class="inline-flex items-center gap-2 px-4 py-2.5 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
            Ver comprobante
          </a>
        </div>
      </div>

      <!-- Panel lateral -->
      <div class="space-y-5">
        <!-- Cambiar estado -->
        <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
          <h3 class="text-sm font-semibold text-gray-700 mb-4">Cambiar estado</h3>
          <div class="space-y-3">
            <div>
              <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">
                Nuevo estado
              </label>
              <select
                v-model="newStatus"
                class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option v-for="opt in statusOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
              </select>
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">
                Nota <span class="text-gray-400 font-normal normal-case">(opcional)</span>
              </label>
              <textarea
                v-model="note"
                rows="3"
                placeholder="Razón del cambio..."
                class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
              />
            </div>
            <button
              @click="updateStatus"
              :disabled="saving || newStatus === ins.status"
              class="w-full py-2.5 rounded-lg text-sm font-semibold transition-colors disabled:opacity-50"
              :class="saved
                ? 'bg-green-500 text-white'
                : 'bg-blue-600 text-white hover:bg-blue-700'"
            >
              <span v-if="saving" class="flex items-center justify-center gap-2">
                <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
                </svg>
                Guardando...
              </span>
              <span v-else-if="saved">✓ Estado actualizado</span>
              <span v-else>Actualizar estado</span>
            </button>
          </div>
        </div>

        <!-- ID -->
        <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-5">
          <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">ID de inscripción</p>
          <code class="text-xs text-gray-500 break-all leading-relaxed">{{ ins.id }}</code>
        </div>
      </div>
    </div>
  </div>
</template>
