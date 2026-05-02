<script setup lang="ts">
const items = [
  { t: 'Carrera Profesional', d: 'Charla con nuestros pilotos profesionales para que proyectes tu carrera como piloto profesional.' },
  { t: 'Teoría aplicada', d: 'Trazadas, frenado, peso, neumáticos. Lo que cambia tu vuelta.' },
  { t: 'Mecánica básica', d: 'Cómo está hecho tu kart. Para sentir la conducción al máximo.' },
  { t: 'Sesiones en pista', d: 'Múltiples tandas con datos y feedback uno a uno.' },
  { t: 'Equipo completo', d: 'Casco, mono, cuello cervical y guantes incluidos.' },
  { t: 'Diploma de piloto', d: 'Certificación de la academia. Foto y video del fin de semana.' },
]
</script>

<template>
  <section class="section" id="programa">
    <div class="container">
      <span class="eyebrow">Programa</span>
      <h2>Un fin de semana<br /><span class="gold">como piloto profesional</span></h2>
      <div class="grid">
        <div v-for="i in items" :key="i.t" class="clip-card item">
          <h3>{{ i.t }}</h3>
          <p>{{ i.d }}</p>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
h2 { font-size: clamp(36px, 5vw, 56px); margin: 12px 0 36px; }
.gold { color: var(--gold); }
.grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; }
.item h3 { font-size: 22px; color: var(--gold); margin-bottom: 10px; }
.item p { color: var(--text-dim); line-height: 1.55; font-size: 15px; }
@media (max-width: 900px) { .grid { grid-template-columns: 1fr 1fr; } }
@media (max-width: 600px) { .grid { grid-template-columns: 1fr; } }
</style>
