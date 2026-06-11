<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useFormConfigStore } from '../../stores/admin/formConfig'
import type { FormDate, FormPlan } from '../../types/admin'

const store = useFormConfigStore()
const activeTab = ref('dates')
onMounted(() => store.loadAll())

// ── Fechas ────────────────────────────────────────────────────────
const showDateModal = ref(false)
const editingDate = ref<FormDate | null>(null)
const dateForm = ref({ label: '', starts_on: '', capacity: 10, is_enabled: true, sort_order: 0 })

function openCreateDate() {
  editingDate.value = null
  dateForm.value = { label: '', starts_on: '', capacity: 10, is_enabled: true, sort_order: store.dates.length }
  showDateModal.value = true
}

function openEditDate(d: FormDate) {
  editingDate.value = d
  dateForm.value = { label: d.label, starts_on: d.starts_on, capacity: d.capacity ?? 0, is_enabled: d.is_enabled, sort_order: d.sort_order }
  showDateModal.value = true
}

async function saveDate() {
  if (!dateForm.value.starts_on || !dateForm.value.label) return
  if (editingDate.value) {
    await store.updateDate(editingDate.value.id, dateForm.value)
  } else {
    await store.createDate(dateForm.value)
  }
  showDateModal.value = false
}

async function toggleDate(d: FormDate) {
  await store.updateDate(d.id, { is_enabled: !d.is_enabled })
}

// ── Planes ────────────────────────────────────────────────────────
const showPlanModal = ref(false)
const editingPlan = ref<FormPlan | null>(null)
const planForm = ref({ key: '', name: '', description: '', price_cop: 0, features: [] as string[], bold_link: '', is_enabled: true, sort_order: 0 })

function openCreatePlan() {
  editingPlan.value = null
  planForm.value = { key: '', name: '', description: '', price_cop: 0, features: [], bold_link: '', is_enabled: true, sort_order: store.plans.length }
  showPlanModal.value = true
}

function openEditPlan(p: FormPlan) {
  editingPlan.value = p
  planForm.value = { key: p.key, name: p.name, description: p.description, price_cop: p.price_cop, features: p.features ?? [], bold_link: p.bold_link ?? '', is_enabled: p.is_enabled, sort_order: p.sort_order }
  showPlanModal.value = true
}

async function savePlan() {
  if (!planForm.value.name || !planForm.value.key) return
  if (editingPlan.value) {
    await store.updatePlan(editingPlan.value.id, planForm.value)
  } else {
    await store.createPlan(planForm.value)
  }
  showPlanModal.value = false
}

async function togglePlan(p: FormPlan) {
  await store.updatePlan(p.id, { is_enabled: !p.is_enabled })
}

function formatCOP(n: number) {
  return new Intl.NumberFormat('es-CO', { style: 'currency', currency: 'COP', maximumFractionDigits: 0 }).format(n)
}

const tabs = [
  { key: 'dates', label: 'Fechas de cursos' },
  { key: 'plans', label: 'Planes' },
  { key: 'methods', label: 'Métodos de pago' },
]
</script>

<template>
  <div class="p-6 lg:p-8">
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900">Configuración del formulario</h1>
      <p class="text-sm text-gray-500 mt-1">Gestiona fechas, planes y métodos de pago del formulario de inscripción.</p>
    </div>

    <!-- Tabs -->
    <div class="flex border-b border-gray-200 mb-6">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        @click="activeTab = tab.key"
        class="px-5 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px"
        :class="activeTab === tab.key
          ? 'border-blue-600 text-blue-600'
          : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'"
      >
        {{ tab.label }}
      </button>
    </div>

    <div v-if="store.loading" class="flex items-center justify-center h-40 text-gray-400 text-sm">
      Cargando...
    </div>

    <!-- FECHAS -->
    <div v-else-if="activeTab === 'dates'">
      <div class="flex justify-end mb-4">
        <button
          @click="openCreateDate"
          class="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          Nueva fecha
        </button>
      </div>
      <div class="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        <table class="w-full text-sm">
          <thead>
            <tr class="bg-gray-50 border-b border-gray-200">
              <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Nombre</th>
              <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Fecha inicio</th>
              <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Capacidad</th>
              <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Estado</th>
              <th class="px-5 py-3.5 w-28" />
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100">
            <tr v-if="store.dates.length === 0">
              <td colspan="5" class="px-5 py-12 text-center text-gray-400 text-sm">No hay fechas configuradas.</td>
            </tr>
            <tr v-for="d in store.dates" :key="d.id" class="hover:bg-gray-50 transition-colors">
              <td class="px-5 py-3.5 font-semibold text-gray-900">{{ d.label }}</td>
              <td class="px-5 py-3.5 text-gray-600">{{ d.starts_on }}</td>
              <td class="px-5 py-3.5 text-gray-600">{{ d.capacity ?? '∞' }}</td>
              <td class="px-5 py-3.5">
                <button
                  @click="toggleDate(d)"
                  class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold transition-colors"
                  :class="d.is_enabled
                    ? 'bg-green-50 text-green-700 ring-1 ring-green-200 hover:bg-green-100'
                    : 'bg-gray-100 text-gray-500 ring-1 ring-gray-200 hover:bg-gray-200'"
                >
                  {{ d.is_enabled ? 'Activa' : 'Inactiva' }}
                </button>
              </td>
              <td class="px-5 py-3.5 text-right">
                <button @click="openEditDate(d)" class="text-blue-600 text-xs font-medium hover:underline mr-4">
                  Editar
                </button>
                <button @click="store.deleteDate(d.id)" class="text-red-500 text-xs font-medium hover:underline">
                  Eliminar
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- PLANES -->
    <div v-else-if="activeTab === 'plans'">
      <div class="flex justify-end mb-4">
        <button
          @click="openCreatePlan"
          class="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          Nuevo plan
        </button>
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        <div v-if="store.plans.length === 0" class="col-span-full text-center py-12 text-gray-400 text-sm">
          No hay planes configurados.
        </div>
        <div
          v-for="p in store.plans"
          :key="p.id"
          class="bg-white rounded-xl border border-gray-200 shadow-sm p-5 flex flex-col"
        >
          <div class="flex items-start justify-between mb-2">
            <p class="font-bold text-gray-900 text-base">{{ p.name }}</p>
            <span
              class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold"
              :class="p.is_enabled
                ? 'bg-green-50 text-green-700 ring-1 ring-green-200'
                : 'bg-gray-100 text-gray-500 ring-1 ring-gray-200'"
            >
              {{ p.is_enabled ? 'Activo' : 'Inactivo' }}
            </span>
          </div>
          <code class="text-xs text-gray-400 mb-2">{{ p.key }}</code>
          <p class="text-gray-500 text-sm mb-3 flex-1 leading-relaxed">{{ p.description || '—' }}</p>
          <p class="text-2xl font-bold text-blue-600 mb-3">{{ formatCOP(p.price_cop) }}</p>
          <div class="mb-4">
            <span
              v-if="p.bold_link"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200"
              :title="p.bold_link"
            >
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" /></svg>
              Link Bold configurado
            </span>
            <span
              v-else
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-gray-50 text-gray-400 ring-1 ring-gray-200"
            >
              Link Bold dinámico (API)
            </span>
          </div>
          <div class="flex gap-2">
            <button
              @click="openEditPlan(p)"
              class="flex-1 py-2 border border-gray-300 rounded-lg text-sm font-medium hover:bg-gray-50 transition-colors"
            >
              Editar
            </button>
            <button
              @click="togglePlan(p)"
              class="px-3 py-2 border border-gray-300 rounded-lg text-sm text-gray-500 hover:bg-gray-50 transition-colors"
            >
              {{ p.is_enabled ? 'Desactivar' : 'Activar' }}
            </button>
            <button
              @click="store.deletePlan(p.id)"
              class="px-3 py-2 border border-red-200 text-red-500 rounded-lg text-sm hover:bg-red-50 transition-colors"
            >
              Eliminar
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- MÉTODOS -->
    <div v-else-if="activeTab === 'methods'">
      <div class="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        <table class="w-full text-sm">
          <thead>
            <tr class="bg-gray-50 border-b border-gray-200">
              <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Método</th>
              <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Clave</th>
              <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Recargo</th>
              <th class="px-5 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Estado</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100">
            <tr v-if="store.methods.length === 0">
              <td colspan="4" class="px-5 py-12 text-center text-gray-400 text-sm">No hay métodos configurados.</td>
            </tr>
            <tr v-for="m in store.methods" :key="m.id" class="hover:bg-gray-50 transition-colors">
              <td class="px-5 py-3.5 font-semibold text-gray-900">{{ m.label }}</td>
              <td class="px-5 py-3.5">
                <code class="text-xs text-gray-500 bg-gray-100 px-2 py-0.5 rounded">{{ m.key }}</code>
              </td>
              <td class="px-5 py-3.5 text-gray-600">
                <span v-if="m.surcharge_pct > 0" class="text-orange-600 font-medium">+{{ m.surcharge_pct }}%</span>
                <span v-else class="text-green-600 font-medium">Sin recargo</span>
              </td>
              <td class="px-5 py-3.5">
                <button
                  @click="store.toggleMethod(m.id, !m.is_enabled)"
                  class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold transition-colors"
                  :class="m.is_enabled
                    ? 'bg-green-50 text-green-700 ring-1 ring-green-200 hover:bg-green-100'
                    : 'bg-gray-100 text-gray-500 ring-1 ring-gray-200 hover:bg-gray-200'"
                >
                  {{ m.is_enabled ? 'Activo' : 'Inactivo' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="text-xs text-gray-400 mt-3">Los métodos se configuran vía seed de BD. Aquí solo se activan/desactivan.</p>
    </div>

    <!-- Modal fecha -->
    <Teleport to="body">
      <div v-if="showDateModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/40 backdrop-blur-sm">
        <div class="bg-white rounded-2xl w-full max-w-md shadow-xl">
          <div class="flex items-center justify-between px-6 py-5 border-b border-gray-100">
            <h3 class="font-semibold text-gray-900">{{ editingDate ? 'Editar fecha' : 'Nueva fecha de curso' }}</h3>
            <button @click="showDateModal = false" class="text-gray-400 hover:text-gray-600">
              <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <div class="px-6 py-5 space-y-4">
            <div>
              <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">
                Nombre visible <span class="text-red-500">*</span>
              </label>
              <input
                v-model="dateForm.label"
                type="text"
                placeholder="ej. Mayo 9 y 10"
                class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent"
              />
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">
                Fecha de inicio <span class="text-red-500">*</span>
              </label>
              <input
                v-model="dateForm.starts_on"
                type="date"
                class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent"
              />
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">Capacidad</label>
              <input
                v-model.number="dateForm.capacity"
                type="number"
                min="0"
                class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent"
              />
            </div>
            <label class="flex items-center gap-3 cursor-pointer">
              <input v-model="dateForm.is_enabled" type="checkbox" class="w-4 h-4 rounded text-blue-600" />
              <span class="text-sm text-gray-700">Fecha activa (visible en el formulario)</span>
            </label>
          </div>
          <div class="flex gap-3 px-6 py-4 border-t border-gray-100">
            <button
              @click="saveDate"
              :disabled="store.saving"
              class="flex-1 py-2.5 bg-blue-600 text-white rounded-lg text-sm font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {{ store.saving ? 'Guardando...' : 'Guardar' }}
            </button>
            <button
              @click="showDateModal = false"
              class="px-5 py-2.5 border border-gray-300 rounded-lg text-sm font-medium hover:bg-gray-50 transition-colors"
            >
              Cancelar
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Modal plan -->
    <Teleport to="body">
      <div v-if="showPlanModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/40 backdrop-blur-sm">
        <div class="bg-white rounded-2xl w-full max-w-lg shadow-xl max-h-[90vh] overflow-y-auto">
          <div class="flex items-center justify-between px-6 py-5 border-b border-gray-100 sticky top-0 bg-white">
            <h3 class="font-semibold text-gray-900">{{ editingPlan ? 'Editar plan' : 'Nuevo plan' }}</h3>
            <button @click="showPlanModal = false" class="text-gray-400 hover:text-gray-600">
              <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <div class="px-6 py-5 space-y-4">
            <div>
              <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">
                Clave interna <span class="text-red-500">*</span>
              </label>
              <input
                v-model="planForm.key"
                type="text"
                placeholder="ej. reserva"
                class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent"
              />
              <p class="text-xs text-gray-400 mt-1">Identificador único sin espacios (solo letras y guiones).</p>
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">
                Nombre <span class="text-red-500">*</span>
              </label>
              <input
                v-model="planForm.name"
                type="text"
                placeholder="ej. Solo reserva del cupo"
                class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent"
              />
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">Descripción</label>
              <textarea
                v-model="planForm.description"
                rows="2"
                class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent resize-none"
              />
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">
                Precio (COP) <span class="text-red-500">*</span>
              </label>
              <input
                v-model.number="planForm.price_cop"
                type="number"
                min="0"
                class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent"
              />
              <p v-if="planForm.price_cop > 0" class="text-xs text-gray-400 mt-1">
                {{ formatCOP(planForm.price_cop) }}
              </p>
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">
                Link de pago Bold
              </label>
              <input
                v-model="planForm.bold_link"
                type="url"
                placeholder="https://checkout.bold.co/payment/LNK_..."
                class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent"
              />
              <p class="text-xs text-gray-400 mt-1">
                Opcional. Si lo dejas vacío se genera un link dinámico por inscripción vía API de Bold.
                Si lo configuras, todos los pagos de este plan irán a este link fijo.
              </p>
            </div>
            <label class="flex items-center gap-3 cursor-pointer">
              <input v-model="planForm.is_enabled" type="checkbox" class="w-4 h-4 rounded text-blue-600" />
              <span class="text-sm text-gray-700">Plan activo (visible en el formulario)</span>
            </label>
          </div>
          <div class="flex gap-3 px-6 py-4 border-t border-gray-100">
            <button
              @click="savePlan"
              :disabled="store.saving"
              class="flex-1 py-2.5 bg-blue-600 text-white rounded-lg text-sm font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {{ store.saving ? 'Guardando...' : 'Guardar' }}
            </button>
            <button
              @click="showPlanModal = false"
              class="px-5 py-2.5 border border-gray-300 rounded-lg text-sm font-medium hover:bg-gray-50 transition-colors"
            >
              Cancelar
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
