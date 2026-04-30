<script setup lang="ts">
const videos = ['/assets/video1.mp4', '/assets/video2.mp4', '/assets/video3.mp4', '/assets/video4.mp4']
</script>

<template>
  <section class="section gallery">
    <div class="container">
      <span class="eyebrow">En pista</span>
      <h2>Esto pasa<br /><span class="gold">en el kartódromo</span></h2>
      <div class="grid">
        <video v-for="v in videos" :key="v" autoplay muted loop playsinline>
          <source :src="v" type="video/mp4" />
        </video>
      </div>
    </div>
  </section>
</template>

<style scoped>
h2 { font-size: clamp(36px, 5vw, 56px); margin: 12px 0 36px; }
.gold { color: var(--gold); }
.grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px; }
video { width: 100%; aspect-ratio: 16/9; object-fit: cover; border: 1px solid var(--border); }
@media (max-width: 720px) { .grid { grid-template-columns: 1fr; } }
</style>
