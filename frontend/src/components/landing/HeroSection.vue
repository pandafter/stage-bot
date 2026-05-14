<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { useLandingStore } from '@/stores/landing'

const store = useLandingStore()
onMounted(() => store.load())

const data = computed(() => store.hero())

const title = computed(() => data.value.title || 'Conviértete en Piloto de Karts')
const subtitle = computed(() => data.value.subtitle || 'Curso intensivo de fin de semana en el kartódromo de Tocancipá. Mecánica, técnica de manejo y carrera real con instructores activos en competencia.')
const ctaText = computed(() => data.value.cta_text || 'Reserva tu cupo →')
const ctaHref = computed(() => data.value.cta_href || '/inscripcion')
const bgUrl = computed(() => data.value.bg_url || '')
</script>

<template>
  <section class="hero">
    <img v-if="bgUrl" :src="bgUrl" class="bg-img" alt="" />
    <video v-else class="bg" autoplay muted loop playsinline>
      <source src="/assets/video1.mp4" type="video/mp4" />
    </video>
    <div class="overlay" />
    <div class="container hero-content">
      <span class="eyebrow">Tocancipá · Mayo 2026</span>
      <h1 v-html="title.replace('Piloto de Karts', '<span class=\'gold\'>Piloto de Karts</span>')" />
      <p class="lead">{{ subtitle }}</p>
      <div class="cta-row">
        <RouterLink :to="ctaHref" class="btn-gold">{{ ctaText }}</RouterLink>
        <a href="#programa" class="btn-outline">Ver programa</a>
      </div>
    </div>
  </section>
</template>

<style scoped>
.hero {
  position: relative;
  min-height: 88vh;
  display: flex;
  align-items: center;
  overflow: hidden;
}
.bg, .bg-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  z-index: 0;
  opacity: 0.45;
  display: block;
  max-width: none;
}
.overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(9,9,9,0.5) 0%, rgba(9,9,9,0.85) 100%);
  z-index: 1;
}
.hero-content {
  position: relative;
  z-index: 2;
  padding: 120px 24px 100px;
  max-width: 820px;
}
h1 {
  font-size: clamp(48px, 9vw, 104px);
  margin: 18px 0 24px;
}
.gold { color: var(--gold); }
.lead { font-size: 18px; line-height: 1.55; color: var(--text-dim); max-width: 560px; }
.cta-row { display: flex; gap: 14px; margin-top: 36px; flex-wrap: wrap; }
</style>
