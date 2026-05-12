import { defineStore } from 'pinia'
import { ref } from 'vue'
import { cmsService, mediaService } from '../../services/admin'
import type { CmsSection, MediaItem } from '../../types/cms'

export const useAdminCmsStore = defineStore('adminCms', () => {
  const sections = ref<Record<string, CmsSection>>({})
  const media = ref<MediaItem[]>([])
  const loading = ref(false)
  const saving = ref(false)
  const error = ref('')

  async function loadSections() {
    loading.value = true
    try {
      const data: CmsSection[] = await cmsService.getSections()
      sections.value = Object.fromEntries(data.map(s => [s.section_key, s]))
    } catch {
      error.value = 'Error cargando secciones'
    } finally {
      loading.value = false
    }
  }

  async function saveSection(key: string, data: Record<string, unknown>) {
    saving.value = true
    try {
      const updated = await cmsService.updateSection(key, data)
      sections.value[key] = updated
      return true
    } catch {
      error.value = 'Error guardando sección'
      return false
    } finally {
      saving.value = false
    }
  }

  async function loadMedia(tag?: string) {
    const data = await mediaService.list({ tag })
    media.value = data
  }

  async function uploadMedia(file: File, altText = '') {
    const item = await mediaService.upload(file, altText)
    media.value.unshift(item)
    return item
  }

  async function deleteMedia(id: number) {
    await mediaService.delete(id)
    media.value = media.value.filter(m => m.id !== id)
  }

  return { sections, media, loading, saving, error, loadSections, saveSection, loadMedia, uploadMedia, deleteMedia }
})
