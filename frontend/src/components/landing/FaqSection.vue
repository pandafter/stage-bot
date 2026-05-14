<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useLandingStore } from '@/stores/landing'

const store = useLandingStore()
onMounted(() => store.load())

const defaultFaqs = [
  { q: '¿Necesito experiencia previa?', a_html: '<p>No. El curso está diseñado para empezar desde cero, pero también reta a quienes ya han manejado.</p>' },
  { q: '¿Qué edades aplican?', a_html: '<p>Recibimos pilotos desde los 8 años hasta adultos. Cada grupo se ajusta por edad y experiencia.</p>' },
  { q: '¿Qué incluye la inscripción?', a_html: '<p>Equipo de seguridad completo, todas las tandas en pista, instrucción uno a uno y diploma final.</p>' },
  { q: '¿Cómo pago?', a_html: '<p>Tarjeta de crédito/débito, Nequi, Bancolombia, PSE o transferencia con comprobante. Procesamos pagos vía Bold.</p>' },
  { q: '¿Puedo solo reservar el cupo?', a_html: '<p>Sí. La reserva es de $150.000 y abona al pago completo. Te garantiza tu lugar mientras decides.</p>' },
]

const faqs = computed(() => {
  const items = store.faq()
  return items.length > 0 ? items : defaultFaqs
})

const open = ref<number | null>(0)
function toggle(i: number) { open.value = open.value === i ? null : i }
</script>

<template>
  <section class="section" id="faq">
    <div class="container narrow">
      <span class="eyebrow">FAQ</span>
      <h2>Preguntas<br /><span class="gold">frecuentes</span></h2>
      <div class="list">
        <div v-for="(f, i) in faqs" :key="f.q" class="item" :class="{ open: open === i }">
          <button class="q" @click="toggle(i)">
            <span>{{ f.q }}</span>
            <span class="caret">{{ open === i ? '−' : '+' }}</span>
          </button>
          <div v-if="open === i" class="a" v-html="f.a_html" />
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.narrow { max-width: 820px; }
h2 { font-size: clamp(36px, 5vw, 56px); margin: 12px 0 36px; }
.gold { color: var(--gold); }
.list { display: flex; flex-direction: column; gap: 10px; }
.item { background: var(--surface); border: 1px solid var(--border); }
.q {
  width: 100%;
  text-align: left;
  padding: 18px 20px;
  font-family: var(--font-display);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  font-size: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: var(--text);
}
.item.open .q { color: var(--gold); }
.caret { color: var(--gold); font-size: 22px; }
.a { padding: 0 20px 18px; color: var(--text-dim); line-height: 1.6; }
</style>
