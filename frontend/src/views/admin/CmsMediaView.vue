<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAdminCmsStore } from '../../stores/admin/cms'

const store = useAdminCmsStore()
const fileInput = ref<HTMLInputElement>()
const uploading = ref(false)
const altTextInput = ref('')
const copiedId = ref<number | null>(null)
const dragover = ref(false)

onMounted(() => store.loadMedia())

async function uploadFiles(files: FileList) {
  if (!files.length) return
  uploading.value = true
  for (const file of files) {
    await store.uploadMedia(file, altTextInput.value)
  }
  uploading.value = false
  altTextInput.value = ''
  if (fileInput.value) fileInput.value.value = ''
}

async function onFileChange(e: Event) {
  const files = (e.target as HTMLInputElement).files
  if (files) await uploadFiles(files)
}

async function onDrop(e: DragEvent) {
  dragover.value = false
  const files = e.dataTransfer?.files
  if (files) await uploadFiles(files)
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
  <div class="p-6 lg:p-8">
    <!-- Header -->
    <div class="flex items-start justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Biblioteca de imágenes</h1>
        <p class="text-sm text-gray-500 mt-1">{{ store.media.length }} archivos · máx. 5 MB por archivo</p>
      </div>
      <div class="flex items-center gap-3">
        <input
          v-model="altTextInput"
          type="text"
          placeholder="Texto alternativo (opcional)"
          class="border border-gray-300 rounded-lg px-3 py-2 text-sm w-52 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
        <button
          @click="fileInput?.click()"
          :disabled="uploading"
          class="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 disabled:opacity-60 transition-colors"
        >
          <svg v-if="uploading" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
          </svg>
          <svg v-else class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
          </svg>
          {{ uploading ? 'Subiendo...' : 'Subir imágenes' }}
        </button>
        <input ref="fileInput" type="file" accept="image/*" multiple class="hidden" @change="onFileChange" />
      </div>
    </div>

    <!-- Drop zone / empty state -->
    <div
      v-if="store.media.length === 0 && !store.loading"
      class="border-2 border-dashed rounded-xl flex flex-col items-center justify-center py-20 cursor-pointer transition-colors"
      :class="dragover ? 'border-blue-400 bg-blue-50' : 'border-gray-200 hover:border-gray-300'"
      @click="fileInput?.click()"
      @dragover.prevent="dragover = true"
      @dragleave="dragover = false"
      @drop.prevent="onDrop"
    >
      <svg class="w-12 h-12 text-gray-300 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
          d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
      <p class="text-base font-medium text-gray-600 mb-1">Arrastra imágenes aquí</p>
      <p class="text-sm text-gray-400">o haz clic para seleccionar archivos</p>
    </div>

    <!-- Loading -->
    <div v-else-if="store.loading" class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
      <div v-for="i in 10" :key="i" class="rounded-xl bg-gray-200 animate-pulse aspect-square" />
    </div>

    <!-- Grid -->
    <div
      v-else
      class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4"
      @dragover.prevent="dragover = true"
      @dragleave="dragover = false"
      @drop.prevent="onDrop"
    >
      <!-- Upload card -->
      <div
        class="border-2 border-dashed rounded-xl flex flex-col items-center justify-center aspect-square cursor-pointer transition-colors text-gray-400 hover:border-blue-300 hover:text-blue-400"
        :class="dragover ? 'border-blue-400 bg-blue-50' : 'border-gray-200'"
        @click="fileInput?.click()"
      >
        <svg class="w-7 h-7 mb-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4v16m8-8H4" />
        </svg>
        <span class="text-xs font-medium">Subir</span>
      </div>

      <div
        v-for="item in store.media"
        :key="item.id"
        class="group relative border border-gray-200 rounded-xl overflow-hidden bg-white shadow-sm hover:shadow-md transition-shadow"
      >
        <div class="aspect-square overflow-hidden bg-gray-100">
          <img :src="item.url" :alt="item.alt_text" class="w-full h-full object-cover" />
        </div>
        <div class="px-2.5 py-2">
          <p class="text-xs text-gray-600 truncate font-medium" :title="item.filename">{{ item.filename }}</p>
          <p class="text-xs text-gray-400">{{ formatSize(item.size_bytes) }}</p>
        </div>
        <!-- Hover overlay -->
        <div class="absolute inset-0 bg-black/55 opacity-0 group-hover:opacity-100 transition-opacity flex flex-col items-center justify-center gap-2 p-2">
          <button
            @click.stop="copyUrl(item.url, item.id)"
            class="w-full px-3 py-1.5 bg-white text-gray-800 rounded-lg text-xs font-medium hover:bg-gray-100 transition-colors"
          >
            {{ copiedId === item.id ? '✓ Copiado' : 'Copiar URL' }}
          </button>
          <button
            @click.stop="remove(item.id)"
            class="w-full px-3 py-1.5 bg-red-500 text-white rounded-lg text-xs font-medium hover:bg-red-600 transition-colors"
          >
            Eliminar
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
