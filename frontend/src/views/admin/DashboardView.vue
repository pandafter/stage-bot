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
  { key: 'total', label: 'Total inscripciones', color: 'bg-blue-500' },
  { key: 'pendientes', label: 'Pendientes de pago', color: 'bg-yellow-500' },
  { key: 'pagadas', label: 'Pagadas / Confirmadas', color: 'bg-green-500' },
  { key: 'rechazadas', label: 'Rechazadas', color: 'bg-red-500' },
]
</script>

<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold mb-6">Dashboard</h1>
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      <div v-for="card in cards" :key="card.key"
        class="bg-white rounded-xl shadow-sm p-5 border">
        <p class="text-sm text-gray-500 mb-1">{{ card.label }}</p>
        <p v-if="loading" class="text-3xl font-bold text-gray-300">—</p>
        <p v-else class="text-3xl font-bold">{{ stats[card.key as keyof typeof stats] }}</p>
        <div class="mt-3 h-1 rounded-full" :class="card.color" />
      </div>
    </div>
    <div class="bg-white rounded-xl shadow-sm p-5 border">
      <h2 class="font-semibold mb-3">Acciones rápidas</h2>
      <div class="flex flex-wrap gap-3">
        <router-link to="/admin/inscripciones" class="px-4 py-2 bg-blue-50 text-blue-700 rounded-lg text-sm font-medium hover:bg-blue-100">
          Ver inscripciones
        </router-link>
        <router-link to="/admin/landing" class="px-4 py-2 bg-purple-50 text-purple-700 rounded-lg text-sm font-medium hover:bg-purple-100">
          Editar landing
        </router-link>
        <router-link to="/admin/form" class="px-4 py-2 bg-green-50 text-green-700 rounded-lg text-sm font-medium hover:bg-green-100">
          Gestionar fechas
        </router-link>
        <router-link to="/admin/media" class="px-4 py-2 bg-orange-50 text-orange-700 rounded-lg text-sm font-medium hover:bg-orange-100">
          Biblioteca de imágenes
        </router-link>
      </div>
    </div>
  </div>
</template>
