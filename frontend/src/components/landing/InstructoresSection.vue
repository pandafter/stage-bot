<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useLandingStore } from '@/stores/landing'

const store = useLandingStore()
onMounted(() => store.load())

const defaultTeam = [
  { photo_url: '/assets/AndresMelo.png', name: 'Andrés Melo', role: 'Director técnico — Piloto activo', bio_html: '<p>Más de 15 años en pista. Coach principal de la academia.</p>' },
  { photo_url: '/assets/SantiGutierrez.png', name: 'Santi Gutiérrez', role: 'Instructor en pista', bio_html: '<p>Especialista en trazadas y frenado tardío.</p>' },
  { photo_url: '/assets/EquipoMecanico.png', name: 'Equipo mecánico', role: 'Soporte total durante el curso', bio_html: '<p>Karts revisados antes y entre cada tanda.</p>' },
]

const team = computed(() => {
  const items = store.instructores()
  return items.length > 0 ? items : defaultTeam
})
</script>

<template>
  <section class="section instructores" id="instructores">
    <div class="container">
      <span class="eyebrow">Instructores</span>
      <h2>El equipo que te<br /><span class="gold">forma como piloto</span></h2>
      <div class="grid">
        <article v-for="m in team" :key="m.name" class="clip-card card">
          <img :src="m.photo_url" :alt="m.name" />
          <h3>{{ m.name }}</h3>
          <span class="role">{{ m.role }}</span>
          <div class="bio" v-html="m.bio_html" />
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
.card img { width: calc(100% + 56px); height: 280px; object-fit: cover; margin: -28px -28px 18px; display: block; max-width: none; }
.card h3 { font-size: 22px; color: var(--gold); }
.role { display: block; color: var(--text-dim); text-transform: uppercase; letter-spacing: 0.1em; font-size: 12px; margin: 6px 0 14px; }
.bio { color: var(--text-dim); line-height: 1.55; font-size: 15px; }
@media (max-width: 900px) { .grid { grid-template-columns: 1fr 1fr; } }
@media (max-width: 600px) { .grid { grid-template-columns: 1fr; } }
</style>
