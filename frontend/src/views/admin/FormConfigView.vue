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
const dateForm = ref({
  date: '',
  capacity: 0,
  is_active: true,
  display_order: 0,
})

function openCreateDate() {
  editingDate.value = null
  dateForm.value = { date: '', capacity: 0, is_active: true, display_order: store.dates.length }
  showDateModal.value = true
}

function openEditDate(d: FormDate) {
  editingDate.value = d
  dateForm.value = { date: d.date, capacity: d.capacity, is_active: d.is_active, display_order: d.display_order }
  showDateModal.value = true
}

async function saveDate() {
  if (!dateForm.value.date) return
  if (editingDate.value) {
    await store.updateDate(editingDate.value.id, dateForm.value)
  } else {
    await store.createDate(dateForm.value)
  }
  showDateModal.value = false
}

async function toggleDate(d: FormDate) {
  await store.updateDate(d.id, { is_active: !d.is_active })
}

// ── Planes ────────────────────────────────────────────────────────
const showPlanModal = ref(false)
const editingPlan = ref<FormPlan | null>(null)
const planForm = ref({
  name: '',
  description: '',
  price_cop: 0,
  is_active: true,
  display_order: 0,
})

function openCreatePlan() {
  editingPlan.value = null
  planForm.value = { name: '', description: '', price_cop: 0, is_active: true, display_order: store.plans.length }
  showPlanModal.value = true
}

function openEditPlan(p: FormPlan) {
  editingPlan.value = p
  planForm.value = { name: p.name, description: p.description, price_cop: p.price_cop, is_active: p.is_active, display_order: p.display_order }
  showPlanModal.value = true
}

async function savePlan() {
  if (!planForm.value.name) return
  if (editingPlan.value) {
    await store.updatePlan(editingPlan.value.id, planForm.value)
  } else {
    await store.createPlan(planForm.value)
  }
  showPlanModal.value = false
}

async function togglePlan(p: FormPlan) {
  await store.updatePlan(p.id, { is_active: !p.is_active })
}

function formatCOP(n: number) {
  return new Intl.NumberFormat('es-CO', { style: 'currency', currency: 'COP', maximumFractionDigits: 0 }).format(n)
}
</script>

<template>
  <div class="p-6">
    <div class="mb-6">
      <h1 class="text-2xl font-bold">Configuración del formulario</h1>
      <p class="text-sm text-gray-500 mt-1">Gestiona fechas, planes y métodos de pago del formulario de inscripción.</p>
    </div>

    <!-- Tabs -->
    <div class="flex gap-1 border-b mb-6">
      <button
        v-for="tab in [{ key: 'dates', label: 'Fechas de cursos' }, { key: 'plans', label: 'Planes' }, { key: 'methods', label: 'Métodos de pago' }]"
        :key="tab.key"
        @click="activeTab = tab.key"
        class="px-4 py-2 text-sm font-medium rounded-t-lg"
        :class="activeTab === tab.key ? 'bg-white border border-b-white text-blue-600 -mb-px' : 'text-gray-500 hover:text-gray-700'"
      >
        {{ tab.label }}
      </button>
    </div>

    <div v-if="store.loading" class="flex items-center justify-center h-40 text-gray-400">Cargando...</div>

    <!-- FECHAS -->
    <div v-else-if="activeTab === 'dates'">
      <div class="flex justify-end mb-4">
        <button @click="openCreateDate"
          class="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700">
          + Nueva fecha
        </button>
      </div>

      <div class="bg-white rounded-xl shadow-sm border overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 border-b">
            <tr>
              <th class="px-4 py-3 text-left font-medium text-gray-600">Fecha</th>
              <th class="px-4 py-3 text-left font-medium text-gray-600">Capacidad</th>
              <th class="px-4 py-3 text-left font-medium text-gray-600">Cupos disponibles</th>
              <th class="px-4 py-3 text-left font-medium text-gray-600">Estado</th>
              <th class="px-4 py-3" />
            </tr>
          </thead>
          <tbody class="divide-y">
            <tr v-if="store.dates.length === 0">
              <td colspan="5" class="px-4 py-10 text-center text-gray-400">No hay fechas configuradas.</td>
            </tr>
            <tr v-for="d in store.dates" :key="d.id" class="hover:bg-gray-50">
              <td class="px-4 py-3 font-medium">{{ d.date }}</td>
              <td class="px-4 py-3 text-gray-600">{{ d.capacity }}</td>
              <td class="px-4 py-3 text-gray-600">{{ d.available_slots }}</td>
              <td class="px-4 py-3">
                <button
                  @click="toggleDate(d)"
                  class="px-2 py-1 rounded-full text-xs font-medium transition-colors"
                  :class="d.is_active ? 'bg-green-100 text-green-700 hover:bg-green-200' : 'bg-gray-100 text-gray-500 hover:bg-gray-200'"
                >
                  {{ d.is_active ? 'Activa' : 'Inactiva' }}
                </button>
              </td>
              <td class="px-4 py-3 text-right">
                <button @click="openEditDate(d)" class="text-blue-500 hover:underline text-xs mr-3">Editar</button>
                <button @click="store.deleteDate(d.id)" class="text-red-500 hover:underline text-xs">Eliminar</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- PLANES -->
    <div v-else-if="activeTab === 'plans'">
      <div class="flex justify-end mb-4">
        <button @click="openCreatePlan"
          class="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700">
          + Nuevo plan
        </button>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        <div v-if="store.plans.length === 0" class="col-span-full text-center py-10 text-gray-400">
          No hay planes configurados.
        </div>
        <div v-for="p in store.plans" :key="p.id"
          class="bg-white rounded-xl shadow-sm border p-5 flex flex-col">
          <div class="flex items-start justify-between mb-3">
            <p class="font-bold text-lg">{{ p.name }}</p>
            <span class="px-2 py-1 rounded-full text-xs font-medium"
              :class="p.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'">
              {{ p.is_active ? 'Activo' : 'Inactivo' }}
            </span>
          </div>
          <p class="text-gray-600 text-sm mb-3 flex-1">{{ p.description || '—' }}</p>
          <p class="text-2xl font-bold text-blue-600 mb-4">{{ formatCOP(p.price_cop) }}</p>
          <div class="flex gap-2 mt-auto">
            <button @click="openEditPlan(p)" class="flex-1 py-1.5 border rounded-lg text-sm hover:bg-gray-50">
              Editar
            </button>
            <button @click="togglePlan(p)"
              class="px-3 py-1.5 border rounded-lg text-sm hover:bg-gray-50 text-gray-500">
              {{ p.is_active ? 'Desactivar' : 'Activar' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- MÉTODOS -->
    <div v-else-if="activeTab === 'methods'">
      <div class="bg-white rounded-xl shadow-sm border overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 border-b">
            <tr>
              <th class="px-4 py-3 text-left font-medium text-gray-600">Método</th>
              <th class="px-4 py-3 text-left font-medium text-gray-600">Código</th>
              <th class="px-4 py-3 text-left font-medium text-gray-600">Recargo %</th>
              <th class="px-4 py-3 text-left font-medium text-gray-600">Estado</th>
            </tr>
          </thead>
          <tbody class="divide-y">
            <tr v-if="store.methods.length === 0">
              <td colspan="4" class="px-4 py-10 text-center text-gray-400">No hay métodos configurados.</td>
            </tr>
            <tr v-for="m in store.methods" :key="m.id" class="hover:bg-gray-50">
              <td class="px-4 py-3 font-medium">{{ m.label }}</td>
              <td class="px-4 py-3 font-mono text-xs text-gray-500">{{ m.code }}</td>
              <td class="px-4 py-3 text-gray-600">{{ m.surcharge_pct }}%</td>
              <td class="px-4 py-3">
                <button
                  @click="store.toggleMethod(m.id, !m.is_active)"
                  class="px-2 py-1 rounded-full text-xs font-medium transition-colors"
                  :class="m.is_active ? 'bg-green-100 text-green-700 hover:bg-green-200' : 'bg-gray-100 text-gray-500 hover:bg-gray-200'"
                >
                  {{ m.is_active ? 'Activo' : 'Inactivo' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="text-xs text-gray-400 mt-3">Los métodos de pago se configuran inicialmente vía seed de BD. Aquí solo se activan/desactivan.</p>
    </div>

    <!-- Modal fecha -->
    <div v-if="showDateModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl">
        <h3 class="font-semibold text-lg mb-4">{{ editingDate ? 'Editar fecha' : 'Nueva fecha' }}</h3>
        <div class="space-y-3">
          <div>
            <label class="block text-sm font-medium mb-1">Fecha de inicio <span class="text-red-500">*</span></label>
            <input v-model="dateForm.date" type="date"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">Capacidad</label>
            <input v-model.number="dateForm.capacity" type="number" min="0"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none" />
          </div>
          <label class="flex items-center gap-2 cursor-pointer">
            <input v-model="dateForm.is_active" type="checkbox" class="rounded" />
            <span class="text-sm">Fecha activa (visible en el formulario)</span>
          </label>
        </div>
        <div class="flex gap-3 mt-5">
          <button @click="saveDate" :disabled="store.saving"
            class="flex-1 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 disabled:opacity-50">
            {{ store.saving ? 'Guardando...' : 'Guardar' }}
          </button>
          <button @click="showDateModal = false"
            class="px-4 py-2 border rounded-lg text-sm hover:bg-gray-50">
            Cancelar
          </button>
        </div>
      </div>
    </div>

    <!-- Modal plan -->
    <div v-if="showPlanModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="bg-white rounded-xl p-6 w-full max-w-lg shadow-xl max-h-[90vh] overflow-y-auto">
        <h3 class="font-semibold text-lg mb-4">{{ editingPlan ? 'Editar plan' : 'Nuevo plan' }}</h3>
        <div class="space-y-3">
          <div>
            <label class="block text-sm font-medium mb-1">Nombre <span class="text-red-500">*</span></label>
            <input v-model="planForm.name" type="text" placeholder="ej. Plan Básico"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">Descripción</label>
            <textarea v-model="planForm.description" rows="2"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">Precio (COP) <span class="text-red-500">*</span></label>
            <input v-model.number="planForm.price_cop" type="number" min="0"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none" />
          </div>
          <label class="flex items-center gap-2 cursor-pointer">
            <input v-model="planForm.is_active" type="checkbox" class="rounded" />
            <span class="text-sm">Plan activo (visible en el formulario)</span>
          </label>
        </div>
        <div class="flex gap-3 mt-5">
          <button @click="savePlan" :disabled="store.saving"
            class="flex-1 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 disabled:opacity-50">
            {{ store.saving ? 'Guardando...' : 'Guardar' }}
          </button>
          <button @click="showPlanModal = false"
            class="px-4 py-2 border rounded-lg text-sm hover:bg-gray-50">
            Cancelar
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
