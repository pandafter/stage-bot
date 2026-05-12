<script setup lang="ts">
import { ref } from 'vue'
import { useAdminCmsStore } from '../../stores/admin/cms'
import type { MediaItem } from '../../types/cms'

const props = defineProps<{
  modelValue?: number | null
  currentUrl?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [id: number | null]
  'update:currentUrl': [url: string]
}>()

const store = useAdminCmsStore()
const showGallery = ref(false)
const uploading = ref(false)
const fileInput = ref<HTMLInputElement>()

async function onFileChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  uploading.value = true
  try {
    await store.loadMedia()
    const item = await store.uploadMedia(file, '')
    emit('update:modelValue', item.id)
    emit('update:currentUrl', item.url)
    showGallery.value = false
  } finally {
    uploading.value = false
  }
}

function selectFromGallery(item: MediaItem) {
  emit('update:modelValue', item.id)
  emit('update:currentUrl', item.url)
  showGallery.value = false
}

async function openGallery() {
  await store.loadMedia()
  showGallery.value = true
}
</script>

<template>
  <div class="space-y-2">
    <div v-if="currentUrl" class="relative w-full max-w-xs">
      <img :src="currentUrl" class="rounded-lg object-cover w-full h-40 border" alt="imagen actual" />
      <button @click="emit('update:modelValue', null); emit('update:currentUrl', '')"
        class="absolute top-1 right-1 bg-red-500 text-white rounded-full w-6 h-6 flex items-center justify-center text-xs">
        ×
      </button>
    </div>
    <div class="flex gap-2">
      <button type="button" @click="fileInput?.click()"
        class="px-3 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700">
        {{ uploading ? 'Subiendo...' : 'Subir imagen' }}
      </button>
      <button type="button" @click="openGallery"
        class="px-3 py-1.5 text-sm border rounded-lg hover:bg-gray-50">
        Elegir de galería
      </button>
    </div>
    <input ref="fileInput" type="file" accept="image/*" class="hidden" @change="onFileChange" />

    <!-- Modal galería -->
    <div v-if="showGallery" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="bg-white rounded-xl p-6 w-full max-w-3xl max-h-[80vh] overflow-y-auto">
        <div class="flex justify-between mb-4">
          <h3 class="font-semibold text-lg">Seleccionar imagen</h3>
          <button @click="showGallery = false" class="text-gray-400 hover:text-gray-600 text-xl">×</button>
        </div>
        <div class="grid grid-cols-4 gap-3">
          <button v-for="item in store.media" :key="item.id" type="button"
            @click="selectFromGallery(item)"
            class="relative group rounded-lg overflow-hidden border-2 hover:border-blue-500 transition-colors"
            :class="{ 'border-blue-500': modelValue === item.id }">
            <img :src="item.url" :alt="item.alt_text" class="w-full h-24 object-cover" />
          </button>
        </div>
        <div v-if="store.media.length === 0" class="text-center py-12 text-gray-400">
          No hay imágenes. Sube una primero.
        </div>
      </div>
    </div>
  </div>
</template>
