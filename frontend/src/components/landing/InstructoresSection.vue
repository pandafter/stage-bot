<script setup lang="ts">
import VoiceNote from '@/components/ui/VoiceNote.vue'

const team = [
  { img: '/assets/AndresMelo.png', name: 'Andrés Melo', role: 'Director técnico — Piloto activo', bio: 'Más de 15 años en pista. Coach principal de la academia.' },
  { img: '/assets/SantiGutierrez.png', name: 'Santi Gutiérrez', role: 'Instructor en pista', bio: 'Especialista en trazadas y frenado tardío.' },
  { img: '/assets/EquipoMecanico.png', name: 'Equipo mecánico', role: 'Soporte total durante el curso', bio: 'Karts revisados antes y entre cada tanda.' },
]
</script>

<template>
  <section class="section instructores" id="instructores">
    <div class="container">
      <span class="eyebrow">Instructores</span>
      <h2>El equipo que te<br /><span class="gold">forma como piloto</span></h2>
      <div class="grid">
        <article v-for="m in team" :key="m.name" class="clip-card card">
          <img :src="m.img" :alt="m.name" />
          <h3>{{ m.name }}</h3>
          <span class="role">{{ m.role }}</span>
          <p>{{ m.bio }}</p>
          <!-- Nota de voz del director (solo en la tarjeta de Andrés) -->
          <VoiceNote
            v-if="m.name === 'Andrés Melo'"
            src="/assets/director.mp3"
            label="Mensaje del director"
            class="card-voice"
          />
        </article>
      </div>
    </div>
  </section>
</template>

<style scoped>
.instructores { background: #060606; }
h2 { font-size: clamp(36px, 5vw, 56px); margin: 12px 0 36px; }
.gold { color: var(--gold); }
.grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; }
.card img { width: 100%; height: 280px; object-fit: cover; margin: -28px -28px 18px; width: calc(100% + 56px); }
.card h3 { font-size: 22px; color: var(--gold); }
.role { display: block; color: var(--text-dim); text-transform: uppercase; letter-spacing: 0.1em; font-size: 12px; margin: 6px 0 14px; }
.card p { color: var(--text-dim); line-height: 1.55; font-size: 15px; }
@media (max-width: 900px) { .grid { grid-template-columns: 1fr 1fr; } }
@media (max-width: 600px) { .grid { grid-template-columns: 1fr; } }
.card-voice { margin-top: 16px; }
</style>
