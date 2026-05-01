<script setup lang="ts">
const stats = [
  { value: '+1000', label: 'Pilotos formados' },
  { value: '10 hrs', label: 'En pista por estudiante' },
  { value: '2:1', label: 'Ratio piloto/instructor' },
  { value: '100%', label: 'Satisfacción del cliente' },
]
</script>

<template>
  <section class="section stats">
    <div class="container grid">
      <div v-for="s in stats" :key="s.label" class="cell">
        <div class="value">{{ s.value }}</div>
        <div class="label">{{ s.label }}</div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.stats { background: #050505; border-top: 1px solid var(--border); border-bottom: 1px solid var(--border); }
.grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
}
.cell { text-align: center; padding: 16px 8px; }
.value {
  font-family: var(--font-display);
  font-weight: 900;
  font-size: clamp(36px, 5vw, 60px);
  color: var(--gold);
  line-height: 1;
}
.label {
  margin-top: 8px;
  font-size: 13px;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--text-dim);
}
@media (max-width: 720px) { .grid { grid-template-columns: repeat(2, 1fr); } }
</style>
