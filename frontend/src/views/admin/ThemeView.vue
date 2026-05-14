<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { themeService } from '../../services/admin'

const theme = ref({
  primary_color: '#0066cc',
  secondary_color: '#1a1a2e',
  accent_color: '#ff6b35',
  font_heading: 'Inter',
  font_body: 'Inter',
  logo_url: '',
  favicon_url: '',
})

const loading = ref(true)
const saving = ref(false)
const saved = ref(false)

onMounted(async () => {
  try {
    const data = await themeService.get()
    if (data && typeof data === 'object') {
      theme.value = { ...theme.value, ...data }
    }
  } catch {
    // usar defaults si no hay tema guardado
  } finally {
    loading.value = false
  }
})

async function save() {
  saving.value = true
  try {
    await themeService.update(theme.value as Record<string, unknown>)
    saved.value = true
    setTimeout(() => { saved.value = false }, 2500)
  } finally {
    saving.value = false
  }
}

const fontOptions = ['Inter', 'Roboto', 'Montserrat', 'Poppins', 'Raleway', 'Open Sans', 'Lato']

const colorFields = [
  { key: 'primary_color', label: 'Color primario', desc: 'Botones principales, links activos' },
  { key: 'secondary_color', label: 'Color secundario', desc: 'Fondos oscuros, hero' },
  { key: 'accent_color', label: 'Color acento', desc: 'Highlights, detalles decorativos' },
] as const
</script>

<template>
  <div class="p-6 lg:p-8">
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900">Tema visual</h1>
      <p class="text-sm text-gray-500 mt-1">Los colores y fuentes se aplican al recargar el landing.</p>
    </div>

    <div v-if="loading" class="space-y-4">
      <div v-for="i in 3" :key="i" class="bg-white rounded-xl border border-gray-200 h-32 animate-pulse" />
    </div>

    <div v-else class="max-w-2xl space-y-5">
      <!-- Colores -->
      <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
        <h2 class="text-sm font-semibold text-gray-700 mb-4">Colores</h2>
        <div class="space-y-5">
          <div v-for="field in colorFields" :key="field.key" class="flex items-center gap-4">
            <input
              type="color"
              :value="theme[field.key]"
              @input="theme[field.key] = ($event.target as HTMLInputElement).value"
              class="w-10 h-10 rounded-lg border border-gray-200 cursor-pointer p-0.5 shrink-0"
            />
            <div class="flex-1">
              <p class="text-sm font-medium text-gray-800">{{ field.label }}</p>
              <p class="text-xs text-gray-400">{{ field.desc }}</p>
            </div>
            <input
              type="text"
              :value="theme[field.key]"
              @input="theme[field.key] = ($event.target as HTMLInputElement).value"
              maxlength="7"
              class="w-28 border border-gray-300 rounded-lg px-3 py-1.5 text-sm font-mono focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent"
            />
            <div class="w-8 h-8 rounded-lg border border-gray-200 shrink-0" :style="{ backgroundColor: theme[field.key] }" />
          </div>
        </div>
      </div>

      <!-- Preview -->
      <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
        <h2 class="text-sm font-semibold text-gray-700 mb-4">Vista previa</h2>
        <div class="rounded-xl overflow-hidden border border-gray-100">
          <div class="p-5" :style="{ backgroundColor: theme.secondary_color }">
            <p class="text-white text-xs mb-3 opacity-60">Landing preview</p>
            <button
              class="px-4 py-2 rounded-lg text-white text-sm font-semibold"
              :style="{ backgroundColor: theme.primary_color }"
            >
              Inscríbete ahora
            </button>
            <p class="mt-3 text-sm font-medium" :style="{ color: theme.accent_color }">
              Scudería St4ge — Academia de Kartismo
            </p>
          </div>
          <div class="flex gap-3 p-4 bg-gray-50">
            <div class="text-center flex-1">
              <div class="w-8 h-8 rounded-lg mx-auto mb-1" :style="{ backgroundColor: theme.primary_color }" />
              <p class="text-xs text-gray-500">Primario</p>
              <code class="text-xs text-gray-400">{{ theme.primary_color }}</code>
            </div>
            <div class="text-center flex-1">
              <div class="w-8 h-8 rounded-lg mx-auto mb-1" :style="{ backgroundColor: theme.secondary_color }" />
              <p class="text-xs text-gray-500">Secundario</p>
              <code class="text-xs text-gray-400">{{ theme.secondary_color }}</code>
            </div>
            <div class="text-center flex-1">
              <div class="w-8 h-8 rounded-lg mx-auto mb-1" :style="{ backgroundColor: theme.accent_color }" />
              <p class="text-xs text-gray-500">Acento</p>
              <code class="text-xs text-gray-400">{{ theme.accent_color }}</code>
            </div>
          </div>
        </div>
      </div>

      <!-- Tipografía -->
      <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
        <h2 class="text-sm font-semibold text-gray-700 mb-4">Tipografía</h2>
        <div class="grid grid-cols-2 gap-5">
          <div>
            <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">Fuente de títulos</label>
            <select
              v-model="theme.font_heading"
              class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent"
            >
              <option v-for="font in fontOptions" :key="font" :value="font">{{ font }}</option>
            </select>
            <p class="mt-3 text-xl font-bold text-gray-800" :style="{ fontFamily: theme.font_heading }">
              Scudería St4ge
            </p>
          </div>
          <div>
            <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">Fuente de cuerpo</label>
            <select
              v-model="theme.font_body"
              class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent"
            >
              <option v-for="font in fontOptions" :key="font" :value="font">{{ font }}</option>
            </select>
            <p class="mt-3 text-sm text-gray-600 leading-relaxed" :style="{ fontFamily: theme.font_body }">
              Aprende karting de la mano de los mejores pilotos colombianos.
            </p>
          </div>
        </div>
      </div>

      <!-- Logo / Favicon -->
      <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
        <h2 class="text-sm font-semibold text-gray-700 mb-4">Logo y Favicon</h2>
        <div class="space-y-4">
          <div>
            <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">URL del logo</label>
            <input
              v-model="theme.logo_url"
              type="url"
              placeholder="https://..."
              class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent"
            />
            <div v-if="theme.logo_url" class="mt-3 p-3 bg-gray-50 rounded-lg inline-block">
              <img :src="theme.logo_url" alt="Logo preview" class="h-10 object-contain" />
            </div>
          </div>
          <div>
            <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1.5">URL del favicon</label>
            <input
              v-model="theme.favicon_url"
              type="url"
              placeholder="https://..."
              class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none focus:border-transparent"
            />
          </div>
        </div>
      </div>

      <!-- Guardar -->
      <div class="flex items-center gap-4 pb-4">
        <button
          @click="save"
          :disabled="saving"
          class="flex items-center gap-2 px-8 py-2.5 rounded-lg text-sm font-semibold transition-colors disabled:opacity-50"
          :class="saved ? 'bg-green-500 text-white' : 'bg-blue-600 text-white hover:bg-blue-700'"
        >
          <svg v-if="saving" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
          </svg>
          <svg v-else-if="saved" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
          </svg>
          {{ saving ? 'Guardando...' : saved ? 'Tema guardado' : 'Guardar tema' }}
        </button>
      </div>
    </div>
  </div>
</template>
