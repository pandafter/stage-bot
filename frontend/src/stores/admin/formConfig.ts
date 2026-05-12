import { defineStore } from 'pinia'
import { ref } from 'vue'
import { formConfigService } from '../../services/admin'
import type { FormDate, FormPlan, FormMethod } from '../../types/admin'

export const useFormConfigStore = defineStore('adminFormConfig', () => {
  const dates = ref<FormDate[]>([])
  const plans = ref<FormPlan[]>([])
  const methods = ref<FormMethod[]>([])
  const loading = ref(false)
  const saving = ref(false)

  async function loadAll() {
    loading.value = true
    try {
      const [d, p, m] = await Promise.all([
        formConfigService.listDates(),
        formConfigService.listPlans(),
        formConfigService.listMethods(),
      ])
      dates.value = d ?? []
      plans.value = p ?? []
      methods.value = m ?? []
    } finally {
      loading.value = false
    }
  }

  // Dates
  async function createDate(data: Omit<FormDate, 'id' | 'available_slots'>) {
    saving.value = true
    try {
      const created = await formConfigService.createDate(data as Record<string, unknown>)
      dates.value.push(created)
    } finally {
      saving.value = false
    }
  }

  async function updateDate(id: number, data: Partial<FormDate>) {
    saving.value = true
    try {
      await formConfigService.updateDate(id, data as Record<string, unknown>)
      const idx = dates.value.findIndex(d => d.id === id)
      if (idx !== -1) dates.value[idx] = { ...dates.value[idx], ...data }
    } finally {
      saving.value = false
    }
  }

  async function deleteDate(id: number) {
    await formConfigService.deleteDate(id)
    dates.value = dates.value.filter(d => d.id !== id)
  }

  async function reorderDates(ids: number[]) {
    await formConfigService.reorderDates(ids)
    const map = Object.fromEntries(dates.value.map(d => [d.id, d]))
    dates.value = ids.map((id, i) => ({ ...map[id], display_order: i })).filter(Boolean) as FormDate[]
  }

  // Plans
  async function createPlan(data: Omit<FormPlan, 'id'>) {
    saving.value = true
    try {
      const created = await formConfigService.createPlan(data as Record<string, unknown>)
      plans.value.push(created)
    } finally {
      saving.value = false
    }
  }

  async function updatePlan(id: number, data: Partial<FormPlan>) {
    saving.value = true
    try {
      await formConfigService.updatePlan(id, data as Record<string, unknown>)
      const idx = plans.value.findIndex(p => p.id === id)
      if (idx !== -1) plans.value[idx] = { ...plans.value[idx], ...data }
    } finally {
      saving.value = false
    }
  }

  async function deletePlan(id: number) {
    await formConfigService.deletePlan(id)
    plans.value = plans.value.filter(p => p.id !== id)
  }

  // Methods
  async function toggleMethod(id: number, isEnabled: boolean) {
    saving.value = true
    try {
      await formConfigService.updateMethod(id, { is_active: isEnabled })
      const m = methods.value.find(m => m.id === id)
      if (m) m.is_active = isEnabled
    } finally {
      saving.value = false
    }
  }

  async function updateMethod(id: number, data: Partial<FormMethod>) {
    saving.value = true
    try {
      await formConfigService.updateMethod(id, data as Record<string, unknown>)
      const idx = methods.value.findIndex(m => m.id === id)
      if (idx !== -1) methods.value[idx] = { ...methods.value[idx], ...data }
    } finally {
      saving.value = false
    }
  }

  return {
    dates, plans, methods, loading, saving,
    loadAll,
    createDate, updateDate, deleteDate, reorderDates,
    createPlan, updatePlan, deletePlan,
    toggleMethod, updateMethod,
  }
})
