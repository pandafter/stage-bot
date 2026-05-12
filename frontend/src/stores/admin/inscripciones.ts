import { defineStore } from 'pinia'
import { ref } from 'vue'
import { inscripcionesAdminService } from '../../services/admin'
import type { AdminInscripcion, InscripcionFilter } from '../../types/admin'

export const useInscripcionesAdminStore = defineStore('adminInscripciones', () => {
  const items = ref<AdminInscripcion[]>([])
  const total = ref(0)
  const loading = ref(false)
  const currentPage = ref(1)
  const filters = ref<InscripcionFilter>({ limit: 20 })

  async function load() {
    loading.value = true
    try {
      const res = await inscripcionesAdminService.list({
        ...filters.value,
        page: currentPage.value,
      })
      items.value = res.items ?? []
      total.value = res.total ?? 0
    } finally {
      loading.value = false
    }
  }

  async function updateStatus(id: string, status: string, note?: string) {
    await inscripcionesAdminService.updateStatus(id, status, note)
    const item = items.value.find(i => i.id === id)
    if (item) item.status = status
  }

  function setFilter(key: keyof InscripcionFilter, value: unknown) {
    (filters.value as Record<string, unknown>)[key] = value
    currentPage.value = 1
  }

  function resetFilters() {
    filters.value = { limit: 20 }
    currentPage.value = 1
  }

  return { items, total, loading, currentPage, filters, load, updateStatus, setFilter, resetFilters }
})
