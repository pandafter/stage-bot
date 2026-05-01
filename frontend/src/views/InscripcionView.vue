<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useInscripcionStore } from '@/stores/inscripcion'
import { fetchConfig, createInscripcion, uploadComprobante } from '@/services/inscripciones'
import type { ConfigResponse, MetodoPagoId, ModalidadId } from '@/types/api'
import DiscountBanner from '@/components/ui/DiscountBanner.vue'
import { useDiscountTimer } from '@/composables/useDiscountTimer'
import { useFormPersistence } from '@/composables/useFormPersistence'

const router = useRouter()
const store = useInscripcionStore()
const config = ref<ConfigResponse | null>(null)
const loading = ref(false)
const comprobanteFile = ref<File | null>(null)
const serverError = ref('')
const termsAccepted = ref(false)

// Discount timer — starts when user enters inscription flow
const { startTimer, expired: discountExpired, effectivePrice, priceDiscount, priceFull } = useDiscountTimer()

// Form persistence — saves personal data to localStorage, restores on mount
const { clear: clearSavedForm } = useFormPersistence()

onMounted(async () => {
  // Start the 10-minute discount countdown
  startTimer()
  try {
    config.value = await fetchConfig()
  } catch (e) {
    console.error('Failed to load config', e)
  }
})

const step = computed(() => store.step)
const form = computed(() => store.form)
const errors = computed(() => store.errors)

/** Modalidades list from config — when discount expires, "completo" shows full price */
const effectiveModalidades = computed(() => {
  const mods = config.value?.modalidades ?? []
  if (!discountExpired.value) return mods
  // Swap the "completo" price to full when discount has expired
  return mods.map(m =>
    m.id === 'completo' ? { ...m, price_cop: priceFull } : m
  )
})

const selectedModalidad = computed(() =>
  effectiveModalidades.value.find(m => m.id === form.value.modalidad)
)

const selectedMetodo = computed(() =>
  config.value?.metodos.find(m => m.id === form.value.metodo_pago)
)

/** Whether to show the discount visual (crossed out full price + badge) */
const showDiscount = computed(() => !discountExpired.value)

const finalMonto = computed(() => {
  if (!selectedModalidad.value) return 0
  let monto = selectedModalidad.value.price_cop
  if (form.value.metodo_pago === 'tarjeta' && config.value) {
    monto += Math.round((monto * config.value.card_surcharge_pct) / 100)
  }
  return monto
})

function formatCOP(n: number): string {
  return n.toLocaleString('es-CO')
}

function validateStep1(): boolean {
  const e: Record<string, string> = {}
  if (!form.value.modalidad) e.modalidad = 'Selecciona la modalidad'
  if (!form.value.fecha_curso) e.fecha_curso = 'Selecciona la fecha'
  store.setErrors(e)
  return Object.keys(e).length === 0
}

function validateStep2(): boolean {
  const e: Record<string, string> = {}
  const f = form.value
  if (!f.nombre_piloto?.trim()) e.nombre_piloto = 'Nombre obligatorio'
  if (!f.edad || f.edad < 8 || f.edad > 90) e.edad = 'Edad entre 8 y 90'
  if (!f.tipo_documento) e.tipo_documento = 'Selecciona tipo de documento'
  if (!f.numero_documento?.trim()) e.numero_documento = 'Número de documento obligatorio'
  if (!f.telefono?.trim()) e.telefono = 'Teléfono obligatorio'
  const emailRe = /^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$/i
  if (!f.email || !emailRe.test(f.email)) e.email = 'Email inválido'
  if (!f.eps?.trim()) e.eps = 'EPS obligatoria'
  if (!f.grupo_sanguineo) e.grupo_sanguineo = 'Grupo sanguíneo obligatorio'
  if (!f.familiar_nombre?.trim()) e.familiar_nombre = 'Contacto de emergencia obligatorio'
  if (!f.familiar_telefono?.trim()) e.familiar_telefono = 'Teléfono de emergencia obligatorio'
  store.setErrors(e)
  return Object.keys(e).length === 0
}

function validateStep3(): boolean {
  const e: Record<string, string> = {}
  if (!form.value.metodo_pago) e.metodo_pago = 'Selecciona un método de pago'
  if (form.value.metodo_pago === 'transferencia' && !comprobanteFile.value) {
    e.comprobante = 'Sube el comprobante de la transferencia'
  }
  store.setErrors(e)
  return Object.keys(e).length === 0
}

function nextStep() {
  if (step.value === 1 && validateStep1()) store.next()
  else if (step.value === 2 && validateStep2()) store.next()
  else if (step.value === 3 && validateStep3()) store.next()
}

function prevStep() {
  serverError.value = ''
  store.prev()
}

function goToStep(s: 1 | 2 | 3 | 4) {
  // Only allow going back, not forward (must pass validation)
  if (s < step.value) store.goTo(s)
}

async function submit() {
  if (!termsAccepted.value) return
  loading.value = true
  serverError.value = ''
  try {
    const result = await createInscripcion({
      email: form.value.email!,
      nombre_piloto: form.value.nombre_piloto!,
      edad: form.value.edad!,
      tipo_documento: form.value.tipo_documento!,
      numero_documento: form.value.numero_documento!,
      telefono: form.value.telefono!,
      ciudad: form.value.ciudad || '',
      eps: form.value.eps!,
      grupo_sanguineo: form.value.grupo_sanguineo!,
      familiar_nombre: form.value.familiar_nombre!,
      familiar_telefono: form.value.familiar_telefono!,
      instagram_user: form.value.instagram_user || '',
      modalidad: form.value.modalidad as ModalidadId,
      metodo_pago: form.value.metodo_pago as MetodoPagoId,
      fecha_curso: form.value.fecha_curso!,
    })

    store.result = result

    // Clear saved form data on successful submission
    clearSavedForm()

    // Upload comprobante if transfer
    if (result.requires_comprobante && comprobanteFile.value) {
      try {
        await uploadComprobante(result.id, comprobanteFile.value)
      } catch (e) {
        console.error('Comprobante upload failed', e)
      }
    }

    // If Bold checkout, redirect
    if (result.checkout_url) {
      window.location.href = result.checkout_url
      return
    }

    // Otherwise go to success
    router.push({ name: 'success', query: { id: result.id } })
  } catch (e: any) {
    serverError.value = e.response?.data?.error || 'Error al crear la inscripción. Intenta de nuevo.'
  } finally {
    loading.value = false
  }
}

function handleFile(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files?.[0]) {
    if (input.files[0].size > 10 * 1024 * 1024) {
      store.setErrors({ ...store.errors, comprobante: 'Archivo supera 10 MB' })
      return
    }
    comprobanteFile.value = input.files[0]
    const newErrors = { ...store.errors }
    delete newErrors.comprobante
    store.setErrors(newErrors)
  }
}

// Clear errors when values change
watch(() => form.value.modalidad, () => {
  if (errors.value.modalidad) {
    const e = { ...errors.value }; delete e.modalidad; store.setErrors(e)
  }
})
watch(() => form.value.fecha_curso, () => {
  if (errors.value.fecha_curso) {
    const e = { ...errors.value }; delete e.fecha_curso; store.setErrors(e)
  }
})
watch(() => form.value.metodo_pago, () => {
  if (errors.value.metodo_pago) {
    const e = { ...errors.value }; delete e.metodo_pago; store.setErrors(e)
  }
})

const docTypes = ['Cédula de ciudadanía', 'Tarjeta de identidad', 'Cédula de extranjería', 'Pasaporte']
const gruposSanguineos = ['O+', 'O-', 'A+', 'A-', 'B+', 'B-', 'AB+', 'AB-']

const payIcons: Record<string, string> = {
  tarjeta: '💳',
  nequi: '',
  bancolombia: '',
  pse: '🏦',
  transferencia: '📎',
}
</script>

<template>
  <header class="form-nav">
    <div class="form-nav-row container">
      <router-link to="/" class="nav-back">← Volver</router-link>
      <router-link to="/" class="nav-logo"><img src="/assets/logo.jpg" alt="Scudería St4ge" /></router-link>
      <span class="nav-label">Inscripción</span>
    </div>
  </header>

  <div class="page-wrap" v-if="config">
    <div class="form-col">
      <div class="form-eyebrow">Inscripción · Scudería St4ge</div>
      <h1 class="form-title">Reserva<br>tu Cupo</h1>
      <p class="form-sub">Completa el formulario, realiza tu pago de reserva y asegura tu lugar en pista.</p>

      <!-- Discount countdown banner -->
      <DiscountBanner />

      <!-- Progress -->
      <div class="progress">
        <div
          v-for="s in 4" :key="s"
          class="prog-step"
          :class="{ done: s < step, active: s === step }"
          @click="goToStep(s as 1|2|3|4)"
        />
      </div>

      <!-- Step 1: Modalidad + Fecha -->
      <div v-if="step === 1" class="step">
        <div class="form-section">
          <div class="section-title"><span class="num">1</span>¿Cómo quieres pagar?</div>
          <div class="options-grid">
            <label
              v-for="m in effectiveModalidades" :key="m.id"
              class="option-card" :class="{ selected: form.modalidad === m.id }"
            >
              <input type="radio" :value="m.id" @change="store.update('modalidad', m.id)" :checked="form.modalidad === m.id" />
              <div v-if="m.id === 'completo' && showDiscount" class="option-discount">
                <span class="old-price">${{ formatCOP(priceFull) }}</span>
              </div>
              <div class="option-value">${{ formatCOP(m.price_cop) }}</div>
              <div class="option-label">{{ m.label }}</div>
              <div class="option-detail" v-if="m.id === 'reserva'">Asegura tu cupo · saldo el día del curso</div>
              <div class="option-detail" v-else-if="showDiscount">🔥 Precio preventa · descuento activo</div>
              <div class="option-detail" v-else>Preventa · sin pendientes</div>
            </label>
          </div>
          <p v-if="errors.modalidad" class="field-error">{{ errors.modalidad }}</p>
        </div>

        <div class="form-section">
          <div class="section-title"><span class="num">2</span>Fecha del curso</div>
          <div class="options-grid">
            <label
              v-for="f in config.fechas" :key="f"
              class="option-card" :class="{ selected: form.fecha_curso === f }"
            >
              <input type="radio" :value="f" @change="store.update('fecha_curso', f)" :checked="form.fecha_curso === f" />
              <div class="option-value">{{ f.split(' ')[1] }}</div>
              <div class="option-label">{{ f.split(' ')[0] }}</div>
              <div class="option-detail">Fin de semana completo</div>
            </label>
          </div>
          <p v-if="errors.fecha_curso" class="field-error">{{ errors.fecha_curso }}</p>
        </div>

        <button class="btn-gold full" @click="nextStep">Continuar →</button>
      </div>

      <!-- Step 2: Personal Data -->
      <div v-if="step === 2" class="step">
        <div class="form-section">
          <div class="section-title"><span class="num">3</span>Datos del piloto</div>
          <div class="form-grid">
            <div class="field span2">
              <label>Nombre completo <span class="req">*</span></label>
              <input type="text" :value="form.nombre_piloto" @input="store.update('nombre_piloto', ($event.target as HTMLInputElement).value)" :class="{ error: errors.nombre_piloto }" placeholder="Juan Pérez" />
              <p v-if="errors.nombre_piloto" class="field-error">{{ errors.nombre_piloto }}</p>
            </div>
            <div class="field">
              <label>Edad <span class="req">*</span></label>
              <input type="number" :value="form.edad" @input="store.update('edad', parseInt(($event.target as HTMLInputElement).value) || 0)" :class="{ error: errors.edad }" placeholder="25" min="8" max="90" />
              <p v-if="errors.edad" class="field-error">{{ errors.edad }}</p>
            </div>
            <div class="field">
              <label>Tipo de documento <span class="req">*</span></label>
              <select :value="form.tipo_documento" @change="store.update('tipo_documento', ($event.target as HTMLSelectElement).value)" :class="{ error: errors.tipo_documento }">
                <option value="" disabled selected>Seleccionar...</option>
                <option v-for="d in docTypes" :key="d" :value="d">{{ d }}</option>
              </select>
              <p v-if="errors.tipo_documento" class="field-error">{{ errors.tipo_documento }}</p>
            </div>
            <div class="field">
              <label>Número de documento <span class="req">*</span></label>
              <input type="text" :value="form.numero_documento" @input="store.update('numero_documento', ($event.target as HTMLInputElement).value)" :class="{ error: errors.numero_documento }" placeholder="1.234.567.890" />
              <p v-if="errors.numero_documento" class="field-error">{{ errors.numero_documento }}</p>
            </div>
            <div class="field">
              <label>Teléfono / WhatsApp <span class="req">*</span></label>
              <input type="tel" :value="form.telefono" @input="store.update('telefono', ($event.target as HTMLInputElement).value)" :class="{ error: errors.telefono }" placeholder="301 234 5678" />
              <p v-if="errors.telefono" class="field-error">{{ errors.telefono }}</p>
            </div>
            <div class="field">
              <label>Email <span class="req">*</span></label>
              <input type="email" :value="form.email" @input="store.update('email', ($event.target as HTMLInputElement).value)" :class="{ error: errors.email }" placeholder="correo@ejemplo.com" />
              <p v-if="errors.email" class="field-error">{{ errors.email }}</p>
            </div>
            <div class="field">
              <label>Ciudad</label>
              <input type="text" :value="form.ciudad" @input="store.update('ciudad', ($event.target as HTMLInputElement).value)" placeholder="Bogotá" />
            </div>
            <div class="field">
              <label>Instagram</label>
              <input type="text" :value="form.instagram_user" @input="store.update('instagram_user', ($event.target as HTMLInputElement).value)" placeholder="@tu_usuario" />
            </div>
          </div>
        </div>

        <div class="form-section">
          <div class="section-title"><span class="num">4</span>Datos médicos y emergencia</div>
          <div class="form-grid">
            <div class="field">
              <label>EPS <span class="req">*</span></label>
              <input type="text" :value="form.eps" @input="store.update('eps', ($event.target as HTMLInputElement).value)" :class="{ error: errors.eps }" placeholder="Sura, Sanitas..." />
              <p v-if="errors.eps" class="field-error">{{ errors.eps }}</p>
            </div>
            <div class="field">
              <label>Grupo sanguíneo <span class="req">*</span></label>
              <select :value="form.grupo_sanguineo" @change="store.update('grupo_sanguineo', ($event.target as HTMLSelectElement).value)" :class="{ error: errors.grupo_sanguineo }">
                <option value="" disabled selected>Seleccionar...</option>
                <option v-for="g in gruposSanguineos" :key="g" :value="g">{{ g }}</option>
              </select>
              <p v-if="errors.grupo_sanguineo" class="field-error">{{ errors.grupo_sanguineo }}</p>
            </div>
            <div class="field">
              <label>Familiar de emergencia <span class="req">*</span></label>
              <input type="text" :value="form.familiar_nombre" @input="store.update('familiar_nombre', ($event.target as HTMLInputElement).value)" :class="{ error: errors.familiar_nombre }" placeholder="María Pérez (mamá)" />
              <p v-if="errors.familiar_nombre" class="field-error">{{ errors.familiar_nombre }}</p>
            </div>
            <div class="field">
              <label>Teléfono emergencia <span class="req">*</span></label>
              <input type="tel" :value="form.familiar_telefono" @input="store.update('familiar_telefono', ($event.target as HTMLInputElement).value)" :class="{ error: errors.familiar_telefono }" placeholder="300 123 4567" />
              <p v-if="errors.familiar_telefono" class="field-error">{{ errors.familiar_telefono }}</p>
            </div>
          </div>
        </div>

        <div class="btn-row">
          <button class="btn-outline" @click="prevStep">← Atrás</button>
          <button class="btn-gold" @click="nextStep">Continuar →</button>
        </div>
      </div>

      <!-- Step 3: Payment Method -->
      <div v-if="step === 3" class="step">
        <div class="form-section">
          <div class="section-title"><span class="num">5</span>Método de pago</div>
          <div class="pay-options">
            <label
              v-for="m in config.metodos" :key="m.id"
              class="pay-opt" :class="{ selected: form.metodo_pago === m.id }"
            >
              <input type="radio" :value="m.id" @change="store.update('metodo_pago', m.id)" :checked="form.metodo_pago === m.id" />
              <span class="pay-indicator" />
              <span v-if="m.id === 'nequi'" class="pay-icon-img"><img src="/assets/nequiLogo.webp" alt="Nequi" /></span>
              <span v-else-if="m.id === 'bancolombia'" class="pay-icon-img"><img src="/assets/bancolombiaLogo.png" alt="Bancolombia" /></span>
              <span v-else class="pay-icon">{{ payIcons[m.id] || '💰' }}</span>
              <span class="pay-info">
                <span class="pay-name">{{ m.label }}</span>
                <span v-if="m.id === 'tarjeta'" class="pay-detail">+{{ config.card_surcharge_pct }}% recargo</span>
                <span v-else-if="m.id === 'transferencia'" class="pay-detail">Sube tu comprobante</span>
                <span v-else class="pay-detail">Pago seguro vía Bold</span>
              </span>
            </label>
          </div>
          <p v-if="errors.metodo_pago" class="field-error">{{ errors.metodo_pago }}</p>

          <!-- File upload for transferencia -->
          <div v-if="form.metodo_pago === 'transferencia'" class="file-upload-section">
            <p class="transfer-info">Realiza la transferencia a la cuenta y sube el comprobante:</p>
            <div class="file-upload" :class="{ 'has-file': comprobanteFile }">
              <input type="file" accept="image/*,.pdf" @change="handleFile" />
              <span class="file-icon">{{ comprobanteFile ? '✅' : '📎' }}</span>
              <span v-if="comprobanteFile" class="file-name">{{ comprobanteFile.name }}</span>
              <span v-else class="file-text"><strong>Haz clic o arrastra</strong>JPG, PNG o PDF — máx 10 MB</span>
            </div>
            <p v-if="errors.comprobante" class="field-error">{{ errors.comprobante }}</p>
          </div>
        </div>

        <div class="price-summary">
          <div class="price-row">
            <span>{{ selectedModalidad?.label }}</span>
            <span>${{ formatCOP(selectedModalidad?.price_cop || 0) }}</span>
          </div>
          <div v-if="form.metodo_pago === 'tarjeta'" class="price-row surcharge">
            <span>Recargo tarjeta ({{ config.card_surcharge_pct }}%)</span>
            <span>${{ formatCOP(finalMonto - (selectedModalidad?.price_cop || 0)) }}</span>
          </div>
          <div class="price-row total">
            <span>Total</span>
            <span>${{ formatCOP(finalMonto) }} COP</span>
          </div>
        </div>

        <div class="btn-row">
          <button class="btn-outline" @click="prevStep">← Atrás</button>
          <button class="btn-gold" @click="nextStep">Revisar y confirmar →</button>
        </div>
      </div>

      <!-- Step 4: Confirmation -->
      <div v-if="step === 4" class="step">
        <div class="form-section">
          <div class="section-title"><span class="num">✓</span>Confirma tu inscripción</div>

          <div class="summary-card">
            <div class="summary-row">
              <span class="summary-label">Piloto</span>
              <span>{{ form.nombre_piloto }}</span>
            </div>
            <div class="summary-row">
              <span class="summary-label">Documento</span>
              <span>{{ form.tipo_documento }} {{ form.numero_documento }}</span>
            </div>
            <div class="summary-row">
              <span class="summary-label">Email</span>
              <span>{{ form.email }}</span>
            </div>
            <div class="summary-row">
              <span class="summary-label">Teléfono</span>
              <span>{{ form.telefono }}</span>
            </div>
            <div class="summary-row">
              <span class="summary-label">Fecha del curso</span>
              <span>{{ form.fecha_curso }}</span>
            </div>
            <div class="summary-row">
              <span class="summary-label">Modalidad</span>
              <span>{{ selectedModalidad?.label }}</span>
            </div>
            <div class="summary-row">
              <span class="summary-label">Pago</span>
              <span>{{ selectedMetodo?.label }}</span>
            </div>
            <div class="summary-row total">
              <span class="summary-label">Total</span>
              <span>${{ formatCOP(finalMonto) }} COP</span>
            </div>
          </div>
        </div>

        <div class="terms-box">
          <p>Al confirmar, acepto los términos y condiciones de Scudería St4ge:</p>
          <ul>
            <li>El cupo se confirma al recibir el pago completo.</li>
            <li>La reserva abona al monto total del curso.</li>
            <li>La academia no se responsabiliza por lesiones derivadas de la actividad deportiva.</li>
            <li>Declaro que la información proporcionada es verídica.</li>
          </ul>
        </div>

        <label class="terms-check">
          <input type="checkbox" v-model="termsAccepted" />
          <span class="check-box" />
          <span>Acepto los términos y condiciones</span>
        </label>

        <p v-if="serverError" class="server-error">{{ serverError }}</p>

        <div class="btn-row">
          <button class="btn-outline" @click="prevStep">← Atrás</button>
          <button class="btn-gold" @click="submit" :disabled="!termsAccepted || loading">
            {{ loading ? 'Procesando...' : (form.metodo_pago === 'transferencia' ? 'Enviar inscripción' : 'Continuar al pago →') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Sidebar -->
    <aside class="info-col">
      <img src="/assets/logo.jpg" alt="Scudería St4ge" class="sidebar-logo" />
      <div class="info-tag">Curso de piloto</div>
      <h3 class="info-title">Scudería St4ge — Tocancipá</h3>
      <div class="divider" />
      <div class="info-price">
        <div class="price-label">{{ selectedModalidad?.label || 'Selecciona modalidad' }}</div>
        <div v-if="form.modalidad === 'completo' && showDiscount" class="sidebar-old-price">
          ${{ formatCOP(priceFull) }}
        </div>
        <div class="price-value">${{ formatCOP(finalMonto) }}</div>
        <div class="price-note">COP · pago único</div>
        <div v-if="form.modalidad === 'completo' && showDiscount" class="sidebar-discount-tag">
          🔥 Descuento preventa activo
        </div>
      </div>
      <div class="divider" />
      <div class="info-includes">
        <div class="inc-item">✓ 2 días en pista — fin de semana completo</div>
        <div class="inc-item">✓ Equipo de seguridad incluido</div>
        <div class="inc-item">✓ Instrucción personalizada</div>
        <div class="inc-item">✓ Carrera final con podio y diploma</div>
        <div class="inc-item">✓ Kart y mecánico asignado</div>
      </div>
      <div class="info-warning">
        ⚠️ <strong>Cupos limitados.</strong> Tu lugar se asegura únicamente con el pago de la reserva o el pago completo.
      </div>
    </aside>
  </div>

  <!-- Loading state -->
  <div v-else class="loading-state">
    <div class="spinner" />
    <p>Cargando...</p>
  </div>
</template>

<style scoped>
.form-nav {
  position: sticky;
  top: 0;
  z-index: 50;
  background: rgba(9,9,9,0.95);
  backdrop-filter: blur(20px);
  border-bottom: 1px solid var(--border);
}
.form-nav-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
}
.nav-back {
  color: var(--text-dim);
  font-size: 13px;
  letter-spacing: .08em;
  transition: color .2s;
}
.nav-back:hover { color: var(--gold); }
.nav-logo img { height: 40px; object-fit: contain; }
.nav-label {
  font-family: var(--font-display);
  font-size: 13px;
  letter-spacing: .15em;
  text-transform: uppercase;
  color: var(--text-dimmer);
}

.page-wrap {
  display: grid;
  grid-template-columns: 1fr 380px;
  min-height: calc(100vh - 72px);
}
.form-col {
  padding: 56px 64px;
}
.form-eyebrow {
  font-family: var(--font-display);
  font-size: 12px;
  letter-spacing: .28em;
  text-transform: uppercase;
  color: var(--gold);
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  gap: 12px;
}
.form-eyebrow::before {
  content: '';
  display: block;
  width: 28px;
  height: 1px;
  background: var(--gold);
}
.form-title {
  font-size: clamp(40px, 5vw, 72px);
  line-height: .95;
  margin-bottom: 8px;
}
.form-sub {
  font-size: 15px;
  color: var(--text-dim);
  margin-bottom: 40px;
  line-height: 1.6;
}

.progress {
  display: flex;
  gap: 4px;
  margin-bottom: 40px;
}
.prog-step {
  flex: 1;
  height: 3px;
  background: var(--border2);
  transition: background .3s;
  cursor: pointer;
}
.prog-step.done { background: var(--gold); }
.prog-step.active { background: var(--gold); opacity: .5; }

.form-section { margin-bottom: 40px; }
.section-title {
  font-family: var(--font-display);
  font-size: 20px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: .05em;
  color: var(--gold);
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 20px;
}
.num {
  width: 24px;
  height: 24px;
  background: var(--gold);
  color: #000;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 900;
  flex-shrink: 0;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
.field { display: flex; flex-direction: column; gap: 6px; }
.field.span2 { grid-column: span 2; }
.req { color: var(--gold); }
input.error, select.error { border-color: var(--red) !important; }

.options-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-top: 12px;
}
.option-card {
  border: 1px solid var(--border2);
  background: var(--surface);
  padding: 22px 20px;
  cursor: pointer;
  transition: border-color .2s, background .2s;
  position: relative;
}
.option-card:hover { border-color: rgba(245,193,0,.4); }
.option-card.selected { border-color: var(--gold); background: var(--gold-dim); }
.option-card.selected::after {
  content: '✓';
  position: absolute;
  top: 12px;
  right: 14px;
  color: var(--gold);
  font-size: 16px;
  font-weight: 900;
}
.option-card input { display: none; }
.option-value {
  font-family: var(--font-display);
  font-size: 36px;
  font-weight: 900;
  color: var(--gold);
  line-height: 1;
}
.option-label {
  font-size: 12px;
  letter-spacing: .1em;
  text-transform: uppercase;
  color: var(--text-dim);
  margin-top: 4px;
}
.option-detail {
  font-size: 11px;
  color: var(--text-dimmer);
  margin-top: 4px;
}
.option-discount {
  margin-bottom: 2px;
}
.old-price {
  font-size: 14px;
  text-decoration: line-through;
  color: var(--text-dimmer);
}

/* Payment options */
.pay-options { display: flex; flex-direction: column; gap: 10px; margin-top: 12px; }
.pay-opt {
  border: 1px solid var(--border2);
  background: var(--surface);
  padding: 18px 20px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 16px;
  transition: border-color .2s, background .2s;
}
.pay-opt:hover { border-color: rgba(245,193,0,.4); }
.pay-opt.selected { border-color: var(--gold); background: var(--gold-dim); }
.pay-opt input { display: none; }
.pay-indicator {
  width: 16px;
  height: 16px;
  border: 1px solid var(--border2);
  background: var(--surface);
  flex-shrink: 0;
  border-radius: 50%;
  transition: border-color .2s;
}
.pay-opt.selected .pay-indicator { border-color: var(--gold); background: var(--gold); }
.pay-icon { font-size: 24px; }
.pay-icon-img img { width: 80px; height: 28px; object-fit: contain; }
.pay-info { flex: 1; }
.pay-name {
  font-family: var(--font-display);
  font-size: 18px;
  font-weight: 800;
  text-transform: uppercase;
  display: block;
}
.pay-detail { font-size: 12px; color: var(--text-dim); display: block; margin-top: 2px; }

/* File upload */
.file-upload-section { margin-top: 20px; }
.transfer-info { font-size: 13px; color: var(--text-dim); margin-bottom: 12px; }
.file-upload {
  border: 1px dashed var(--border2);
  background: var(--surface);
  padding: 32px 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  transition: border-color .2s, background .2s;
  position: relative;
}
.file-upload:hover { border-color: var(--gold); background: var(--surface2); }
.file-upload.has-file { border-color: #22c55e; border-style: solid; }
.file-upload input { position: absolute; inset: 0; opacity: 0; cursor: pointer; }
.file-icon { font-size: 32px; }
.file-text { font-size: 14px; color: var(--text-dim); text-align: center; }
.file-text strong { color: var(--text); display: block; font-size: 13px; }
.file-name { font-size: 13px; color: #22c55e; font-weight: 500; }

/* Price summary */
.price-summary {
  background: var(--surface);
  border: 1px solid var(--border);
  padding: 20px;
  margin-bottom: 28px;
}
.price-row {
  display: flex;
  justify-content: space-between;
  font-size: 14px;
  color: var(--text-dim);
  padding: 6px 0;
}
.price-row.surcharge { font-size: 13px; color: var(--text-dimmer); }
.price-row.total {
  border-top: 1px solid var(--border);
  margin-top: 8px;
  padding-top: 12px;
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 900;
  color: var(--gold);
}

/* Summary card */
.summary-card {
  background: var(--surface);
  border: 1px solid var(--border);
  padding: 24px;
}
.summary-row {
  display: flex;
  justify-content: space-between;
  padding: 10px 0;
  font-size: 14px;
  border-bottom: 1px solid var(--border);
}
.summary-row:last-child { border-bottom: none; }
.summary-row.total {
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 900;
  color: var(--gold);
  margin-top: 8px;
  padding-top: 16px;
  border-top: 1px solid var(--gold-dim);
  border-bottom: none;
}
.summary-label { color: var(--text-dim); }

/* Terms */
.terms-box {
  background: var(--surface);
  border: 1px solid var(--border);
  padding: 20px;
  font-size: 13px;
  color: var(--text-dim);
  line-height: 1.7;
  max-height: 160px;
  overflow-y: auto;
  margin-bottom: 16px;
}
.terms-box ul { padding-left: 18px; margin-top: 8px; }
.terms-check {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  margin-bottom: 24px;
  font-size: 14px;
}
.terms-check input { display: none; }
.check-box {
  width: 18px;
  height: 18px;
  border: 1px solid var(--border2);
  background: var(--surface);
  flex-shrink: 0;
  position: relative;
  transition: border-color .2s, background .2s;
}
.terms-check input:checked + .check-box {
  border-color: var(--gold);
  background: var(--gold);
}
.terms-check input:checked + .check-box::after {
  content: '✓';
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  color: #000;
  font-weight: 900;
}

.server-error {
  background: rgba(230,50,0,.1);
  border: 1px solid rgba(230,50,0,.3);
  padding: 12px 16px;
  font-size: 14px;
  color: #ff7a5a;
  margin-bottom: 16px;
}

/* Buttons */
.btn-row { display: flex; gap: 12px; }
.btn-row .btn-gold { flex: 1; }
.btn-row .btn-outline { flex: 0 0 auto; }
.full { width: 100%; }

/* Sidebar */
.info-col {
  background: var(--surface);
  border-left: 1px solid var(--border);
  padding: 40px 32px;
  position: sticky;
  top: 72px;
  align-self: start;
  height: calc(100vh - 72px);
  overflow-y: auto;
}
.sidebar-logo { height: 40px; object-fit: contain; margin-bottom: 20px; }
.info-tag {
  font-family: var(--font-display);
  font-size: 11px;
  letter-spacing: .28em;
  text-transform: uppercase;
  color: var(--gold);
  margin-bottom: 8px;
}
.info-title {
  font-size: 24px;
  line-height: 1.1;
  margin-bottom: 16px;
}
.info-price { margin: 20px 0; }
.price-label { font-size: 12px; color: var(--text-dimmer); text-transform: uppercase; letter-spacing: .1em; }
.price-value {
  font-family: var(--font-display);
  font-size: 48px;
  font-weight: 900;
  color: var(--gold);
  line-height: 1;
  margin-top: 4px;
}
.price-note { font-size: 12px; color: var(--text-dim); margin-top: 4px; }
.sidebar-old-price {
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 700;
  color: var(--text-dimmer);
  text-decoration: line-through;
  line-height: 1;
}
.sidebar-discount-tag {
  font-size: 12px;
  color: var(--gold);
  margin-top: 8px;
  padding: 4px 10px;
  background: var(--gold-dim);
  border: 1px solid rgba(245,193,0,0.25);
  display: inline-block;
}
.info-includes { display: flex; flex-direction: column; gap: 8px; margin: 20px 0; }
.inc-item { font-size: 13px; color: var(--text-dim); }
.info-warning {
  background: rgba(230,50,0,.08);
  border: 1px solid rgba(230,50,0,.2);
  padding: 14px 16px;
  font-size: 12px;
  color: #ff7a5a;
  line-height: 1.6;
  margin-top: 16px;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  gap: 16px;
  color: var(--text-dim);
}
.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border2);
  border-top-color: var(--gold);
  border-radius: 50%;
  animation: spin .8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

@media (max-width: 900px) {
  .page-wrap { grid-template-columns: 1fr; }
  .form-col { padding: 36px 20px; }
  .info-col { position: static; height: auto; border-left: none; border-top: 1px solid var(--border); }
  .form-grid { grid-template-columns: 1fr; }
  .field.span2 { grid-column: span 1; }
  .options-grid { grid-template-columns: 1fr; }
  .btn-row { flex-direction: column-reverse; }
  .form-nav-row { padding: 12px 16px; }
}
</style>
