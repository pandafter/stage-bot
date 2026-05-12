<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAdminCmsStore } from '../../stores/admin/cms'

const store = useAdminCmsStore()
const fileInput = ref<HTMLInputElement>()
const uploading = ref(false)
const altTextInput = ref('')
const copiedId = ref<number | null>(null)

onMounted(() => store.loadMedia())

async function onFileChange(e: Event) {
  const files = (e.target as HTMLInputElement).files
  if (!files?.length) return
  uploading.value = true
  for (const file of files) {
    await store.uploadMedia(file, altTextInput.value)
  }
  uploading.value = false
  altTextInput.value = ''
  if (fileInput.value) fileInput.value.value = ''
}

async function remove(id: number) {
  if (!confirm('¿Eliminar esta imagen?')) return
  await store.deleteMedia(id)
}

function copyUrl(url: string, id: number) {
  navigator.clipboard.writeText(url)
  copiedId.value = id
  setTimeout(() => { copiedId.value = null }, 1500)
}

function formatSize(bytes: number) {
  return bytes < 1024 * 1024
    ? `${(bytes / 1024).toFixed(0)} KB`
    : `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
</script>

<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold">Biblioteca de Imágenes</h1>
        <p class="text-gray-500 text-sm mt-1">{{ store.media.length }} imágenes · máx. 5 MB por archivo</p>
      </div>
      <div class="flex items-center gap-3">
        <input v-model="altTextInput" type="text" placeholder="Texto alternativo (opcional)"
          class="border rounded-lg px-3 py-2 text-sm w-56" />
        <button @click="fileInput?.click()"
          class="px-4 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 text-sm">
          {{ uploading ? 'Subiendo...' : '+ Subir imágenes' }}
        </button>
        <input ref="fileInput" type="file" accept="image/*" multiple class="hidden" @change="onFileChange" />
      </div>
    </div>

    <div v-if="store.media.length === 0" class="flex flex-col items-center justify-center h-64 text-gray-400 border-2 border-dashed rounded-xl">
      <p class="text-lg mb-2">No hay imágenes aún</p>
      <p class="text-sm">Sube tu primera imagen para empezar</p>
    </div>

    <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
      <div v-for="item in store.media" :key="item.id"
        class="group relative border rounded-xl overflow-hidden bg-white shadow-sm hover:shadow-md transition-shadow">
        <img :src="item.url" :alt="item.alt_text" class="w-full h-36 object-cover" />
        <div class="p-2">
          <p class="text-xs text-gray-600 truncate" :title="item.filename">{{ item.filename }}</p>
          <p class="text-xs text-gray-400">{{ formatSize(item.size_bytes) }}</p>
        </div>
        <div class="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
          <button @click="copyUrl(item.url, item.id)"
            class="px-3 py-1.5 bg-white text-gray-800 rounded-lg text-xs font-medium hover:bg-gray-100">
            {{ copiedId === item.id ? '✓ Copiado' : 'Copiar URL' }}
          </button>
          <button @click="remove(item.id)"
            class="px-3 py-1.5 bg-red-500 text-white rounded-lg text-xs font-medium hover:bg-red-600">
            Eliminar
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
