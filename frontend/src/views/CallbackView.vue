<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { getInscripcion } from '@/services/inscripciones'
import type { InscripcionStatusResponse } from '@/types/api'

const route = useRoute()
const data = ref<InscripcionStatusResponse | null>(null)
const error = ref('')
const loading = ref(true)
const polling = ref(false)
let pollTimer: ReturnType<typeof setTimeout> | null = null
let pollAttempts = 0

const refId = computed(() =>
  (route.query.ref as string) || (route.query['bold-order-id'] as string) || ''
)
const txStatus = computed(() => (route.query['bold-tx-status'] as string) || '')

const isConfirmed = computed(() =>
  data.value?.status === 'pagado' || txStatus.value?.toLowerCase() === 'approved'
)
const isRejected = computed(() =>
  data.value?.status === 'pago rechazado' || txStatus.value?.toLowerCase() === 'rejected'
)
const isPending = computed(() => !isConfirmed.value && !isRejected.value)

onMounted(async () => {
  if (!refId.value) {
    error.value = 'No se encontró la referencia de pago.'
    loading.value = false
    return
  }
  await loadStatus()
  if (isPending.value && !error.value) {
    startPolling()
  }
})

onUnmounted(() => {
  if (pollTimer) clearTimeout(pollTimer)
})

async function loadStatus() {
  try {
    data.value = await getInscripcion(refId.value)
    loading.value = false
  } catch (e) {
    error.value = 'No pudimos consultar tu inscripción. Si pagaste, no te preocupes: te contactaremos.'
    loading.value = false
  }
}

function startPolling() {
  polling.value = true
  pollAttempts = 0
  poll()
}

function poll() {
  if (pollAttempts >= 30) {
    polling.value = false
    return
  }
  pollAttempts++
  pollTimer = setTimeout(async () => {
    try {
      data.value = await getInscripcion(refId.value)
      if (data.value?.status === 'pagado' || data.value?.status === 'pago rechazado') {
        polling.value = false
        return
      }
    } catch (_) { /* ignore */ }
    poll()
  }, 2000)
}

function formatCOP(n: number): string {
  return n.toLocaleString('es-CO')
}
</script>

<template>
  <header class="cb-nav">
    <div class="container nav-row">
      <router-link to="/" class="nav-back">← Inicio</router-link>
      <router-link to="/" class="nav-logo"><img src="/assets/logo.jpg" alt="Scudería St4ge" /></router-link>
      <span class="nav-label">Pago</span>
    </div>
  </header>

  <main class="cb-page">
    <!-- Loading -->
    <div v-if="loading" class="center">
      <div class="spinner" />
      <p>Consultando tu pago...</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="center">
      <div class="icon">⚠️</div>
      <h1>Error</h1>
      <p class="sub">{{ error }}</p>
      <router-link to="/" class="btn-gold">Volver al inicio</router-link>
    </div>

    <!-- Confirmed -->
    <div v-else-if="isConfirmed && data" class="center">
      <div class="icon">✅</div>
      <h1 class="confirmed">¡Pago<br>Confirmado!</h1>
      <p class="sub">
        Tu pago fue procesado exitosamente. Tu cupo está reservado y te contactaremos por WhatsApp con los detalles del curso.
      </p>
      <div class="status-badge confirmed">Pago confirmado</div>

      <div class="result-card confirmed-card">
        <div class="card-title">🏁 Cupo asegurado</div>
        <p>
          Recibimos la confirmación oficial de Bold. Te enviamos un correo con el resumen y nuestro equipo te contactará por WhatsApp.
        </p>
        <p class="ref">Conserva este ID: <strong>{{ data.id }}</strong></p>
      </div>

      <div class="summary-card">
        <div class="row"><span class="label">ID</span><span class="mono">{{ data.id }}</span></div>
        <div class="row"><span class="label">Piloto</span><span>{{ data.nombre_piloto }}</span></div>
        <div class="row"><span class="label">Fecha</span><span>{{ data.fecha_curso }}</span></div>
        <div class="row"><span class="label">Monto</span><span>${{ formatCOP(data.monto_cop) }} COP</span></div>
      </div>

      <router-link to="/" class="btn-outline">Volver al inicio</router-link>
    </div>

    <!-- Rejected -->
    <div v-else-if="isRejected && data" class="center">
      <div class="icon">⚠️</div>
      <h1 class="rejected">Pago<br>Rechazado</h1>
      <p class="sub">
        Tu pago no fue aprobado por el banco. No te preocupes — puedes intentarlo nuevamente o usar otro método.
      </p>
      <div class="status-badge rejected">Pago no procesado</div>

      <div class="result-card">
        <div class="card-title">↻ Intenta nuevamente</div>
        <p>Vuelve al formulario y elige otro método de pago, o escríbenos por WhatsApp para ayudarte.</p>
        <router-link to="/inscripcion" class="btn-gold full">↻ Volver al formulario</router-link>
      </div>
    </div>

    <!-- Pending / Verifying -->
    <div v-else-if="data" class="center">
      <div class="icon">⏳</div>
      <h1>Verificando<br>Pago</h1>
      <p class="sub">
        Estamos confirmando tu pago con Bold. Esto suele tardar solo unos segundos.
      </p>
      <div class="status-badge pending">Procesando</div>

      <div class="result-card verifying-card">
        <div class="card-title">🔄 Confirmando con Bold</div>
        <div class="verify-row">
          <div class="spinner-sm" />
          <span>Confirmación en curso...</span>
        </div>
        <p>Esta página se actualizará automáticamente al recibir la confirmación. No cierres esta ventana.</p>
      </div>

      <div class="summary-card">
        <div class="row"><span class="label">ID</span><span class="mono">{{ data.id }}</span></div>
        <div class="row"><span class="label">Piloto</span><span>{{ data.nombre_piloto }}</span></div>
        <div class="row"><span class="label">Monto</span><span>${{ formatCOP(data.monto_cop) }} COP</span></div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.cb-nav {
  position: sticky;
  top: 0;
  z-index: 50;
  background: rgba(9,9,9,0.95);
  backdrop-filter: blur(20px);
  border-bottom: 1px solid var(--border);
}
.nav-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
}
.nav-back { color: var(--text-dim); font-size: 13px; letter-spacing: .08em; }
.nav-back:hover { color: var(--gold); }
.nav-logo img { height: 40px; object-fit: contain; }
.nav-label {
  font-family: var(--font-display);
  font-size: 13px;
  letter-spacing: .15em;
  text-transform: uppercase;
  color: var(--text-dimmer);
}

.cb-page {
  min-height: calc(100vh - 72px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
}
.center {
  text-align: center;
  max-width: 520px;
  width: 100%;
}
.icon { font-size: 56px; margin-bottom: 16px; }
h1 {
  font-size: clamp(36px, 6vw, 64px);
  color: var(--text);
  margin-bottom: 16px;
}
h1.confirmed { color: #22c55e; }
h1.rejected { color: var(--red); }
.sub {
  font-size: 16px;
  color: var(--text-dim);
  line-height: 1.6;
  margin-bottom: 24px;
}

.status-badge {
  display: inline-block;
  font-family: var(--font-display);
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: .1em;
  font-size: 13px;
  padding: 8px 20px;
  margin-bottom: 28px;
}
.status-badge.confirmed { background: rgba(34,197,94,.12); color: #22c55e; border: 1px solid rgba(34,197,94,.3); }
.status-badge.rejected { background: rgba(230,50,0,.1); color: #ff7a5a; border: 1px solid rgba(230,50,0,.3); }
.status-badge.pending { background: var(--gold-dim); color: var(--gold); border: 1px solid rgba(245,193,0,.3); }

.result-card {
  background: var(--surface);
  border: 1px solid var(--border);
  padding: 24px;
  text-align: left;
  margin-bottom: 24px;
}
.result-card.confirmed-card { border-color: rgba(34,197,94,.3); }
.result-card.verifying-card { border-color: rgba(245,193,0,.3); }
.card-title {
  font-family: var(--font-display);
  font-size: 20px;
  font-weight: 800;
  text-transform: uppercase;
  margin-bottom: 12px;
}
.result-card p { font-size: 14px; color: var(--text-dim); line-height: 1.6; margin-top: 8px; }
.ref { margin-top: 12px; }
.ref strong { font-family: monospace; color: var(--gold); }
.full { width: 100%; margin-top: 16px; }

.verify-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
  font-size: 14px;
  color: var(--gold);
}

.summary-card {
  background: var(--surface);
  border: 1px solid var(--border);
  padding: 20px 24px;
  text-align: left;
  margin-bottom: 24px;
}
.row {
  display: flex;
  justify-content: space-between;
  padding: 10px 0;
  font-size: 14px;
  border-bottom: 1px solid var(--border);
}
.row:last-child { border-bottom: none; }
.label { color: var(--text-dim); }
.mono { font-family: monospace; color: var(--gold); font-size: 13px; }

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border2);
  border-top-color: var(--gold);
  border-radius: 50%;
  animation: spin .8s linear infinite;
  margin: 0 auto 16px;
}
.spinner-sm {
  width: 18px;
  height: 18px;
  border: 2px solid var(--border2);
  border-top-color: var(--gold);
  border-radius: 50%;
  animation: spin .8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
