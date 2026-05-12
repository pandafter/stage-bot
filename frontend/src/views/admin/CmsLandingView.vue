<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useAdminCmsStore } from '../../stores/admin/cms'
import ImageUpload from '../../components/admin/ImageUpload.vue'
import RichTextEditor from '../../components/admin/RichTextEditor.vue'
import SortableList from '../../components/admin/SortableList.vue'

const store = useAdminCmsStore()
const activeTab = ref('hero')
const saved = ref('')

onMounted(() => store.loadSections())

// ── Hero ──────────────────────────────────────────────────────────
const hero = computed(() => ({
  title: '', subtitle: '', cta_text: 'Inscríbete', cta_href: '/inscripcion',
  bg_url: '', bg_media_id: null as number | null,
  ...(store.sections['hero']?.data ?? {}),
}))

const heroForm = ref({ ...hero.value })
function resetHero() { heroForm.value = { ...hero.value } }

async function saveHero() {
  const ok = await store.saveSection('hero', heroForm.value)
  if (ok) showSaved('hero')
}

// ── Stats ─────────────────────────────────────────────────────────
const statsItems = ref<Array<{ icon: string; value: string; label: string }>>([])

function loadStats() {
  const data = store.sections['stats']?.data as { items?: typeof statsItems.value } | undefined
  statsItems.value = data?.items ? [...data.items] : []
}

function addStat() { statsItems.value.push({ icon: 'trophy', value: '+0', label: 'nuevo stat' }) }
function removeStat(i: number) { statsItems.value.splice(i, 1) }

async function saveStats() {
  const ok = await store.saveSection('stats', { items: statsItems.value })
  if (ok) showSaved('stats')
}

// ── FAQ ───────────────────────────────────────────────────────────
const faqItems = ref<Array<{ q: string; a_html: string }>>([])

function loadFaq() {
  const data = store.sections['faq']?.data as { items?: typeof faqItems.value } | undefined
  faqItems.value = data?.items ? [...data.items] : []
}

function addFaq() { faqItems.value.push({ q: 'Nueva pregunta', a_html: '<p>Respuesta</p>' }) }
function removeFaq(i: number) { faqItems.value.splice(i, 1) }

async function saveFaq() {
  const ok = await store.saveSection('faq', { items: faqItems.value })
  if (ok) showSaved('faq')
}

// ── Instructores ──────────────────────────────────────────────────
const instructores = ref<Array<{ name: string; role: string; bio_html: string; photo_url?: string; photo_media_id?: number | null }>>([])

function loadInstructores() {
  const data = store.sections['instructores']?.data as { items?: typeof instructores.value } | undefined
  instructores.value = data?.items ? [...data.items] : []
}

function addInstructor() { instructores.value.push({ name: 'Nuevo instructor', role: 'Instructor', bio_html: '<p>Bio</p>' }) }
function removeInstructor(i: number) { instructores.value.splice(i, 1) }

async function saveInstructores() {
  const ok = await store.saveSection('instructores', { items: instructores.value })
  if (ok) showSaved('instructores')
}

// ── CTA ───────────────────────────────────────────────────────────
const ctaForm = ref({ title: '', subtitle: '', cta_text: '', cta_href: '/inscripcion', bg_url: '', bg_media_id: null as number | null })

function loadCta() {
  const data = store.sections['cta']?.data as typeof ctaForm.value | undefined
  if (data) ctaForm.value = { ...ctaForm.value, ...data }
}

async function saveCta() {
  const ok = await store.saveSection('cta', ctaForm.value)
  if (ok) showSaved('cta')
}

// ── Tab switching ─────────────────────────────────────────────────
function switchTab(tab: string) {
  activeTab.value = tab
  if (tab === 'hero') heroForm.value = { ...hero.value }
  if (tab === 'stats') loadStats()
  if (tab === 'faq') loadFaq()
  if (tab === 'instructores') loadInstructores()
  if (tab === 'cta') loadCta()
}

function showSaved(section: string) {
  saved.value = section
  setTimeout(() => { saved.value = '' }, 2000)
}

const tabs = [
  { key: 'hero', label: 'Hero' },
  { key: 'stats', label: 'Estadísticas' },
  { key: 'instructores', label: 'Instructores' },
  { key: 'faq', label: 'FAQ' },
  { key: 'cta', label: 'CTA' },
]
</script>

<template>
  <div class="p-6">
    <div class="mb-6">
      <h1 class="text-2xl font-bold">Editor de Landing</h1>
      <p class="text-gray-500 text-sm mt-1">Los cambios se publican inmediatamente al guardar.</p>
    </div>

    <div v-if="store.loading" class="flex items-center justify-center h-40 text-gray-400">
      Cargando secciones...
    </div>

    <template v-else>
      <!-- Tabs -->
      <div class="flex gap-1 border-b mb-6">
        <button v-for="tab in tabs" :key="tab.key" @click="switchTab(tab.key)"
          class="px-4 py-2 text-sm font-medium rounded-t-lg transition-colors"
          :class="activeTab === tab.key
            ? 'bg-white border border-b-white text-blue-600 -mb-px'
            : 'text-gray-500 hover:text-gray-700'">
          {{ tab.label }}
          <span v-if="saved === tab.key" class="ml-1 text-green-500 text-xs">✓ Guardado</span>
        </button>
      </div>

      <!-- Hero -->
      <div v-if="activeTab === 'hero'" class="space-y-4 max-w-2xl">
        <div>
          <label class="block text-sm font-medium mb-1">Título principal</label>
          <input v-model="heroForm.title" type="text" class="w-full border rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:outline-none" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Subtítulo</label>
          <textarea v-model="heroForm.subtitle" rows="3" class="w-full border rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:outline-none" />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium mb-1">Texto del botón CTA</label>
            <input v-model="heroForm.cta_text" type="text" class="w-full border rounded-lg px-3 py-2" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">Link del botón CTA</label>
            <input v-model="heroForm.cta_href" type="text" class="w-full border rounded-lg px-3 py-2" />
          </div>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Imagen de fondo</label>
          <ImageUpload v-model="heroForm.bg_media_id" v-model:current-url="heroForm.bg_url" />
        </div>
        <div class="flex gap-3 pt-2">
          <button @click="saveHero" :disabled="store.saving"
            class="px-6 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50">
            {{ store.saving ? 'Guardando...' : 'Guardar Hero' }}
          </button>
          <button @click="resetHero" class="px-4 py-2 border rounded-lg text-gray-600 hover:bg-gray-50">
            Descartar cambios
          </button>
        </div>
      </div>

      <!-- Stats -->
      <div v-if="activeTab === 'stats'" class="max-w-2xl">
        <SortableList v-model="statsItems" item-key="label">
          <template #default="{ element: item, index: i }">
            <div class="border rounded-lg p-4 mb-3 bg-white">
              <div class="grid grid-cols-3 gap-3">
                <div>
                  <label class="block text-xs font-medium mb-1 text-gray-500">Icono (nombre)</label>
                  <input v-model="item.icon" type="text" placeholder="trophy" class="w-full border rounded px-2 py-1 text-sm" />
                </div>
                <div>
                  <label class="block text-xs font-medium mb-1 text-gray-500">Valor</label>
                  <input v-model="item.value" type="text" placeholder="+200" class="w-full border rounded px-2 py-1 text-sm" />
                </div>
                <div>
                  <label class="block text-xs font-medium mb-1 text-gray-500">Etiqueta</label>
                  <input v-model="item.label" type="text" placeholder="pilotos formados" class="w-full border rounded px-2 py-1 text-sm" />
                </div>
              </div>
              <button @click="removeStat(i)" class="mt-2 text-red-500 text-xs hover:underline">Eliminar</button>
            </div>
          </template>
        </SortableList>
        <div class="flex gap-3 mt-4">
          <button @click="addStat" class="px-4 py-2 border-2 border-dashed rounded-lg text-gray-500 hover:border-blue-400 hover:text-blue-500 text-sm">
            + Agregar stat
          </button>
          <button @click="saveStats" :disabled="store.saving"
            class="px-6 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50">
            Guardar Estadísticas
          </button>
        </div>
      </div>

      <!-- Instructores -->
      <div v-if="activeTab === 'instructores'" class="max-w-3xl">
        <div v-for="(inst, i) in instructores" :key="i" class="border rounded-lg p-4 mb-4 bg-white">
          <div class="grid grid-cols-2 gap-4 mb-3">
            <div>
              <label class="block text-xs font-medium mb-1 text-gray-500">Nombre</label>
              <input v-model="inst.name" type="text" class="w-full border rounded px-2 py-1.5 text-sm" />
            </div>
            <div>
              <label class="block text-xs font-medium mb-1 text-gray-500">Cargo</label>
              <input v-model="inst.role" type="text" class="w-full border rounded px-2 py-1.5 text-sm" />
            </div>
          </div>
          <div class="mb-3">
            <label class="block text-xs font-medium mb-1 text-gray-500">Foto</label>
            <ImageUpload v-model="inst.photo_media_id" v-model:current-url="inst.photo_url" />
          </div>
          <div class="mb-2">
            <label class="block text-xs font-medium mb-1 text-gray-500">Bio</label>
            <RichTextEditor v-model="inst.bio_html" />
          </div>
          <button @click="removeInstructor(i)" class="text-red-500 text-xs hover:underline">Eliminar instructor</button>
        </div>
        <div class="flex gap-3 mt-2">
          <button @click="addInstructor" class="px-4 py-2 border-2 border-dashed rounded-lg text-gray-500 hover:border-blue-400 text-sm">
            + Agregar instructor
          </button>
          <button @click="saveInstructores" :disabled="store.saving"
            class="px-6 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50">
            Guardar Instructores
          </button>
        </div>
      </div>

      <!-- FAQ -->
      <div v-if="activeTab === 'faq'" class="max-w-2xl">
        <SortableList v-model="faqItems" item-key="q">
          <template #default="{ element: item, index: i }">
            <div class="border rounded-lg p-4 mb-3 bg-white">
              <div class="mb-2">
                <label class="block text-xs font-medium mb-1 text-gray-500">Pregunta</label>
                <input v-model="item.q" type="text" class="w-full border rounded px-2 py-1.5 text-sm" />
              </div>
              <div class="mb-2">
                <label class="block text-xs font-medium mb-1 text-gray-500">Respuesta</label>
                <RichTextEditor v-model="item.a_html" />
              </div>
              <button @click="removeFaq(i)" class="text-red-500 text-xs hover:underline">Eliminar</button>
            </div>
          </template>
        </SortableList>
        <div class="flex gap-3 mt-4">
          <button @click="addFaq" class="px-4 py-2 border-2 border-dashed rounded-lg text-gray-500 hover:border-blue-400 text-sm">
            + Agregar pregunta
          </button>
          <button @click="saveFaq" :disabled="store.saving"
            class="px-6 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50">
            Guardar FAQ
          </button>
        </div>
      </div>

      <!-- CTA -->
      <div v-if="activeTab === 'cta'" class="space-y-4 max-w-2xl">
        <div>
          <label class="block text-sm font-medium mb-1">Título</label>
          <input v-model="ctaForm.title" type="text" class="w-full border rounded-lg px-3 py-2" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Subtítulo</label>
          <textarea v-model="ctaForm.subtitle" rows="2" class="w-full border rounded-lg px-3 py-2" />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium mb-1">Texto botón</label>
            <input v-model="ctaForm.cta_text" type="text" class="w-full border rounded-lg px-3 py-2" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">Link botón</label>
            <input v-model="ctaForm.cta_href" type="text" class="w-full border rounded-lg px-3 py-2" />
          </div>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Imagen de fondo</label>
          <ImageUpload v-model="ctaForm.bg_media_id" v-model:current-url="ctaForm.bg_url" />
        </div>
        <button @click="saveCta" :disabled="store.saving"
          class="px-6 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50">
          Guardar CTA
        </button>
      </div>
    </template>
  </div>
</template>
