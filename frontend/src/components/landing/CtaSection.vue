<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { useLandingStore } from '@/stores/landing'

const store = useLandingStore()
onMounted(() => store.load())

const data = computed(() => store.cta())

const title = computed(() => data.value.title || 'El kartódromo te espera')
const subtitle = computed(() => data.value.subtitle || 'Asegura tu cupo con la reserva de $150.000. Te confirmamos por WhatsApp y te enviamos toda la info logística del fin de semana.')
const ctaText = computed(() => data.value.cta_text || 'Inscribirme ahora →')
const ctaHref = computed(() => data.value.cta_href || '/inscripcion')
</script>

<template>
  <section class="section cta">
    <div class="container clip-card big">
      <span class="eyebrow">Cupos limitados</span>
      <h2 v-html="title.replace('te espera', '<span class=\'gold\'>te espera</span>')" />
      <p>{{ subtitle }}</p>
      <RouterLink :to="ctaHref" class="btn-gold">{{ ctaText }}</RouterLink>
    </div>
  </section>
</template>

<style scoped>
.cta { padding: 60px 0 100px; }
.big { padding: 56px 40px; text-align: center; }
.gold { color: var(--gold); }
h2 { font-size: clamp(36px, 6vw, 64px); margin: 12px 0 18px; }
p { color: var(--text-dim); max-width: 560px; margin: 0 auto 28px; line-height: 1.55; }
</style>
