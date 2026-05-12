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

const statusColors: Record<string, string> = {
  pendiente: 'bg-yellow-100 text-yellow-800',
  pagado: 'bg-green-100 text-green-800',
  comprobante_pendiente: 'bg-blue-100 text-blue-800',
  rechazado: 'bg-red-100 text-red-800',
  cancelado: 'bg-gray-100 text-gray-800',
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
  return new Date(iso).toLocaleString('es-CO')
}

type InfoRow = [string, string | number | undefined]

function infoRows(): InfoRow[] {
  if (!ins.value) return []
  return [
    ['Plan', ins.value.plan],
    ['Monto', formatCOP(ins.value.monto_cop)],
    ['Fecha del curso', ins.value.fecha_curso],
    ['Metodo de pago', ins.value.metodo_pago],
    ['Telefono', ins.value.telefono],
    ['Ciudad', ins.value.ciudad],
    ['Tipo de documento', ins.value.tipo_documento],
    ['Numero de documento', ins.value.numero_documento],
    ['EPS', ins.value.eps],
    ['Grupo sanguineo', ins.value.grupo_sanguineo],
    ['Familiar (nombre)', ins.value.familiar_nombre],
    ['Familiar (telefono)', ins.value.familiar_telefono],
    ['Instagram', ins.value.instagram_user],
    ['Registrado', formatDate(ins.value.created_at)],
  ]
}
</script>

<template>
  <div class="p-6">
    <button @click="router.back()" class="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm">
      <i class="pi pi-arrow-left" />
      Volver a inscripciones
    </button>

    <div v-if="loading" class="flex items-center justify-center h-40 text-gray-400">Cargando...</div>

    <div v-else-if="ins" class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Info principal -->
      <div class="lg:col-span-2 space-y-4">
        <div class="bg-white rounded-xl shadow-sm border p-6">
          <div class="flex items-start justify-between mb-4">
            <div>
              <h1 class="text-xl font-bold">{{ ins.nombre_piloto }}</h1>
              <p class="text-gray-500">{{ ins.email }}</p>
            </div>
            <span class="px-3 py-1 rounded-full text-sm font-medium"
              :class="statusColors[ins.status] ?? 'bg-gray-100'">
              {{ ins.status }}
            </span>
          </div>

          <div class="grid grid-cols-2 gap-4 text-sm">
            <div v-for="[label, value] in infoRows()" :key="label" class="border-b pb-3">
              <p class="text-xs text-gray-400 uppercase tracking-wide">{{ label }}</p>
              <p class="font-medium mt-0.5">{{ value || '—' }}</p>
            </div>
          </div>
        </div>

        <!-- Comprobante -->
        <div v-if="ins.comprobante_path" class="bg-white rounded-xl shadow-sm border p-6">
          <h3 class="font-semibold mb-3">Comprobante de pago</h3>
          <a :href="`/uploads/${ins.comprobante_path}`" target="_blank"
            class="flex items-center gap-2 text-blue-600 hover:underline text-sm">
            <i class="pi pi-file" />
            Ver comprobante
          </a>
        </div>
      </div>

      <!-- Panel de acciones -->
      <div class="space-y-4">
        <div class="bg-white rounded-xl shadow-sm border p-6">
          <h3 class="font-semibold mb-4">Cambiar estado</h3>
          <div class="space-y-3">
            <div>
              <label class="block text-xs font-medium text-gray-500 mb-1">Nuevo estado</label>
              <select v-model="newStatus"
                class="w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
                <option v-for="opt in statusOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
              </select>
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-500 mb-1">Nota (opcional)</label>
              <textarea v-model="note" rows="3" placeholder="Razon del cambio..."
                class="w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <button @click="updateStatus" :disabled="saving || newStatus === ins.status"
              class="w-full py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50 text-sm">
              {{ saving ? 'Guardando...' : saved ? 'Guardado' : 'Actualizar estado' }}
            </button>
          </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm border p-6">
          <h3 class="font-semibold mb-2">ID de inscripcion</h3>
          <code class="text-xs text-gray-500 break-all">{{ ins.id }}</code>
        </div>
      </div>
    </div>
  </div>
</template>
