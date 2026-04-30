<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { getInscripcion } from '@/services/inscripciones'
import type { InscripcionStatusResponse } from '@/types/api'

const route = useRoute()
const data = ref<InscripcionStatusResponse | null>(null)
const error = ref('')
const loading = ref(true)

const id = computed(() => (route.query.id as string) || '')

onMounted(async () => {
  if (!id.value) {
    error.value = 'No se encontró el ID de inscripción.'
    loading.value = false
    return
  }
  try {
    data.value = await getInscripcion(id.value)
  } catch (e) {
    error.value = 'No pudimos cargar tu inscripción. Si ya pagaste, no te preocupes: te contactaremos.'
  } finally {
    loading.value = false
  }
})

function formatCOP(n: number): string {
  return n.toLocaleString('es-CO')
}
</script>

<template>
  <header class="success-nav">
    <div class="container nav-row">
      <router-link to="/" class="nav-back">← Inicio</router-link>
      <router-link to="/" class="nav-logo"><img src="/assets/logo.jpg" alt="Scudería St4ge" /></router-link>
      <span class="nav-label">Confirmación</span>
    </div>
  </header>

  <main class="success-page">
    <!-- Loading -->
    <div v-if="loading" class="center">
      <div class="spinner" />
      <p>Cargando...</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="center">
      <div class="icon">⚠️</div>
      <h1>Error</h1>
      <p class="sub">{{ error }}</p>
      <router-link to="/" class="btn-gold">Volver al inicio</router-link>
    </div>

    <!-- Success -->
    <div v-else-if="data" class="center">
      <div class="icon">🏁</div>
      <h1>¡Cupo<br>Reservado!</h1>
      <p class="sub">
        Tu inscripción quedó registrada. Te contactaremos por WhatsApp para confirmar los detalles.
      </p>

      <div class="status-badge" :class="data.status === 'pagado' ? 'confirmed' : 'pending'">
        {{ data.status }}
      </div>

      <div class="summary-card">
        <div class="row"><span class="label">ID</span><span class="mono">{{ data.id }}</span></div>
        <div class="row"><span class="label">Piloto</span><span>{{ data.nombre_piloto }}</span></div>
        <div class="row"><span class="label">Fecha curso</span><span>{{ data.fecha_curso }}</span></div>
        <div class="row"><span class="label">Monto</span><span>${{ formatCOP(data.monto_cop) }} COP</span></div>
        <div class="row"><span class="label">Estado</span><span>{{ data.status }}</span></div>
      </div>

      <div class="next-steps">
        <h3>Próximos pasos</h3>
        <ul>
          <li>Conserva tu ID: <strong>{{ data.id }}</strong></li>
          <li>Te contactaremos por WhatsApp para enviarte la info logística</li>
          <li>Si tienes dudas, escríbenos a nuestro Instagram</li>
        </ul>
      </div>

      <router-link to="/" class="btn-outline">Volver al inicio</router-link>
    </div>
  </main>
</template>

<style scoped>
.success-nav {
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

.success-page {
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
  color: var(--gold);
  margin-bottom: 16px;
}
.sub {
  font-size: 16px;
  color: var(--text-dim);
  line-height: 1.6;
  margin-bottom: 28px;
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
.status-badge.pending { background: var(--gold-dim); color: var(--gold); border: 1px solid rgba(245,193,0,.3); }

.summary-card {
  background: var(--surface);
  border: 1px solid var(--border);
  padding: 20px 24px;
  text-align: left;
  margin-bottom: 28px;
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

.next-steps {
  background: var(--surface);
  border: 1px solid var(--border);
  padding: 20px 24px;
  text-align: left;
  margin-bottom: 28px;
}
.next-steps h3 {
  font-size: 16px;
  color: var(--gold);
  margin-bottom: 12px;
}
.next-steps ul {
  padding-left: 18px;
  font-size: 14px;
  color: var(--text-dim);
  line-height: 1.8;
}
.next-steps strong { color: var(--gold); font-family: monospace; font-size: 13px; }

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border2);
  border-top-color: var(--gold);
  border-radius: 50%;
  animation: spin .8s linear infinite;
  margin: 0 auto 16px;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
