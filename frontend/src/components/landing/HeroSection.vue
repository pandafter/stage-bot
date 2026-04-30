<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <section class="hero">
    <video class="bg" autoplay muted loop playsinline>
      <source src="/assets/video1.mp4" type="video/mp4" />
    </video>
    <div class="overlay" />
    <div class="container hero-content">
      <span class="eyebrow">Tocancipá · Mayo 2026</span>
      <h1>
        Conviértete en<br /><span class="gold">Piloto de Karts</span>
      </h1>
      <p class="lead">
        Curso intensivo de fin de semana en el kartódromo de Tocancipá.
        Mecánica, técnica de manejo y carrera real con instructores activos en competencia.
      </p>
      <div class="cta-row">
        <RouterLink to="/inscripcion" class="btn-gold">Reserva tu cupo →</RouterLink>
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
.bg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  z-index: 0;
  opacity: 0.45;
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
