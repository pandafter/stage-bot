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
    // Usar defaults si no hay tema guardado
  } finally {
    loading.value = false
  }
})

async function save() {
  saving.value = true
  try {
    await themeService.update(theme.value as Record<string, unknown>)
    saved.value = true
    setTimeout(() => { saved.value = false }, 2000)
  } finally {
    saving.value = false
  }
}

const fontOptions = ['Inter', 'Roboto', 'Montserrat', 'Poppins', 'Raleway', 'Open Sans', 'Lato']
</script>

<template>
  <div class="p-6">
    <div class="mb-6">
      <h1 class="text-2xl font-bold">Tema visual</h1>
      <p class="text-sm text-gray-500 mt-1">Los colores y fuentes se aplican en tiempo real al recargar el landing.</p>
    </div>

    <div v-if="loading" class="flex items-center justify-center h-40 text-gray-400">Cargando...</div>

    <div v-else class="max-w-2xl space-y-6">
      <!-- Colores -->
      <div class="bg-white rounded-xl shadow-sm border p-6">
        <h2 class="font-semibold mb-4">Colores</h2>
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div v-for="[key, label] in [
            ['primary_color', 'Color primario'],
            ['secondary_color', 'Color secundario'],
            ['accent_color', 'Color acento'],
          ]" :key="key">
            <label class="block text-sm font-medium mb-2">{{ label }}</label>
            <div class="flex items-center gap-3">
              <input type="color"
                :value="theme[key as keyof typeof theme]"
                @input="theme[key as keyof typeof theme] = ($event.target as HTMLInputElement).value"
                class="w-12 h-10 rounded-lg border cursor-pointer p-0.5" />
              <input type="text"
                :value="theme[key as keyof typeof theme]"
                @input="theme[key as keyof typeof theme] = ($event.target as HTMLInputElement).value"
                class="flex-1 border rounded-lg px-3 py-2 text-sm font-mono" />
            </div>
          </div>
        </div>
      </div>

      <!-- Preview de colores -->
      <div class="bg-white rounded-xl shadow-sm border p-6">
        <h2 class="font-semibold mb-4">Vista previa de colores</h2>
        <div class="flex gap-3 flex-wrap">
          <div class="flex flex-col items-center gap-1">
            <div class="w-16 h-16 rounded-xl shadow-sm" :style="{ backgroundColor: theme.primary_color }" />
            <span class="text-xs text-gray-500">Primario</span>
          </div>
          <div class="flex flex-col items-center gap-1">
            <div class="w-16 h-16 rounded-xl shadow-sm" :style="{ backgroundColor: theme.secondary_color }" />
            <span class="text-xs text-gray-500">Secundario</span>
          </div>
          <div class="flex flex-col items-center gap-1">
            <div class="w-16 h-16 rounded-xl shadow-sm" :style="{ backgroundColor: theme.accent_color }" />
            <span class="text-xs text-gray-500">Acento</span>
          </div>
          <div class="flex-1 rounded-xl p-4" :style="{ backgroundColor: theme.secondary_color }">
            <button class="px-4 py-2 rounded-lg text-white text-sm font-medium"
              :style="{ backgroundColor: theme.primary_color }">
              Botón de ejemplo
            </button>
            <p class="mt-2 text-sm" :style="{ color: theme.accent_color }">Texto acento</p>
          </div>
        </div>
      </div>

      <!-- Tipografía -->
      <div class="bg-white rounded-xl shadow-sm border p-6">
        <h2 class="font-semibold mb-4">Tipografía</h2>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium mb-1">Fuente de títulos</label>
            <select v-model="theme.font_heading"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none">
              <option v-for="font in fontOptions" :key="font" :value="font">{{ font }}</option>
            </select>
            <p class="mt-2 text-2xl font-bold" :style="{ fontFamily: theme.font_heading }">
              Scudería St4ge
            </p>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">Fuente de cuerpo</label>
            <select v-model="theme.font_body"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none">
              <option v-for="font in fontOptions" :key="font" :value="font">{{ font }}</option>
            </select>
            <p class="mt-2 text-sm" :style="{ fontFamily: theme.font_body }">
              Aprende karting de la mano de los mejores pilotos colombianos.
            </p>
          </div>
        </div>
      </div>

      <!-- URLs Logo / Favicon -->
      <div class="bg-white rounded-xl shadow-sm border p-6">
        <h2 class="font-semibold mb-4">Logo y Favicon</h2>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-1">URL del logo</label>
            <input v-model="theme.logo_url" type="url" placeholder="https://..."
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none" />
            <div v-if="theme.logo_url" class="mt-2">
              <img :src="theme.logo_url" alt="Logo preview" class="h-12 object-contain" />
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">URL del favicon</label>
            <input v-model="theme.favicon_url" type="url" placeholder="https://..."
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none" />
          </div>
        </div>
      </div>

      <!-- Guardar -->
      <div class="flex items-center gap-4">
        <button @click="save" :disabled="saving"
          class="px-8 py-2.5 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50">
          {{ saving ? 'Guardando...' : 'Guardar tema' }}
        </button>
        <span v-if="saved" class="text-green-600 text-sm font-medium">Tema guardado</span>
      </div>

      <!-- Nota multi-tenant -->
      <div class="bg-blue-50 rounded-xl p-4 text-sm text-blue-700">
        <p class="font-semibold mb-1">Multi-tenant listo</p>
        <p>Cada empresa que use esta plataforma tendrá su propio tema, secciones y formulario. El motor resuelve el tenant por dominio — hoy usa <code class="bg-blue-100 px-1 rounded">DEFAULT_TENANT=st4ge</code>.</p>
      </div>
    </div>
  </div>
</template>
