import { watch, onMounted } from 'vue'
import { useInscripcionStore } from '@/stores/inscripcion'
import type { CreateInscripcionRequest } from '@/types/api'

const STORAGE_KEY = 'st4ge_form_draft'
const VERSION = 1

/**
 * Fields that are safe to persist locally (personal info only).
 * Explicitly excludes: modalidad, metodo_pago, fecha_curso (financial/selection data).
 */
const SAFE_FIELDS: (keyof CreateInscripcionRequest)[] = [
  'nombre_piloto',
  'edad',
  'tipo_documento',
  'numero_documento',
  'telefono',
  'email',
  'ciudad',
  'eps',
  'grupo_sanguineo',
  'familiar_nombre',
  'familiar_telefono',
  'instagram_user',
]

interface StoredDraft {
  v: number
  ts: number
  data: Partial<CreateInscripcionRequest>
}

/** Simple obfuscation — not encryption, but prevents casual reading of personal data.
 *  Uses base64 + char shift. Real security comes from HTTPS + short TTL. */
function encode(plain: string): string {
  try {
    const shifted = plain.split('').map(c => String.fromCharCode(c.charCodeAt(0) + 3)).join('')
    return btoa(unescape(encodeURIComponent(shifted)))
  } catch {
    return ''
  }
}

function decode(encoded: string): string {
  try {
    const shifted = decodeURIComponent(escape(atob(encoded)))
    return shifted.split('').map(c => String.fromCharCode(c.charCodeAt(0) - 3)).join('')
  } catch {
    return ''
  }
}

function saveDraft(form: Partial<CreateInscripcionRequest>) {
  try {
    const safe: Partial<CreateInscripcionRequest> = {}
    for (const key of SAFE_FIELDS) {
      const val = form[key]
      if (val !== undefined && val !== null && val !== '') {
        ;(safe as any)[key] = val
      }
    }
    // Don't save if nothing meaningful
    if (Object.keys(safe).length === 0) return

    const draft: StoredDraft = { v: VERSION, ts: Date.now(), data: safe }
    const json = JSON.stringify(draft)
    const encoded = encode(json)
    localStorage.setItem(STORAGE_KEY, encoded)
  } catch {
    // localStorage full or unavailable — silently fail
  }
}

function loadDraft(): Partial<CreateInscripcionRequest> | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null

    const json = decode(raw)
    if (!json) { clearDraft(); return null }

    const draft: StoredDraft = JSON.parse(json)

    // Version check
    if (draft.v !== VERSION) { clearDraft(); return null }

    // Expire after 24 hours for privacy
    const age = Date.now() - draft.ts
    if (age > 24 * 60 * 60 * 1000) { clearDraft(); return null }

    // Only return safe fields
    const result: Partial<CreateInscripcionRequest> = {}
    for (const key of SAFE_FIELDS) {
      const val = (draft.data as any)[key]
      if (val !== undefined && val !== null) {
        ;(result as any)[key] = val
      }
    }
    return Object.keys(result).length > 0 ? result : null
  } catch {
    clearDraft()
    return null
  }
}

function clearDraft() {
  try { localStorage.removeItem(STORAGE_KEY) } catch { /* ignore */ }
}

/**
 * Composable that auto-saves personal form data to localStorage and restores
 * on mount. Only saves SAFE_FIELDS (personal, non-financial). Data is obfuscated
 * and expires after 24h.
 */
export function useFormPersistence() {
  const store = useInscripcionStore()

  /** Restore saved draft if available. Returns true if data was restored. */
  function restore(): boolean {
    const draft = loadDraft()
    if (!draft) return false

    for (const [key, val] of Object.entries(draft)) {
      if (val !== undefined && val !== null) {
        store.update(key as keyof CreateInscripcionRequest, val as any)
      }
    }
    return true
  }

  onMounted(() => {
    // Only restore if the form is empty (user hasn't started filling)
    const hasData = store.form.nombre_piloto || store.form.email
    if (!hasData) {
      restore()
    }
  })

  // Auto-save on form changes (debounced via watch flush)
  watch(
    () => ({ ...store.form }),
    (newForm) => {
      saveDraft(newForm)
    },
    { deep: true }
  )

  /** Clear saved data (call on successful submission). */
  function clear() {
    clearDraft()
  }

  return { restore, clear }
}
