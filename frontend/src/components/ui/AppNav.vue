<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <header class="nav">
    <div class="container nav-row">
      <RouterLink to="/" class="brand">
        <img src="/assets/logo.jpg" alt="Scuderia St4ge" />
        <span>Scudería St4ge</span>
      </RouterLink>
      <nav class="links">
        <a href="#programa">Programa</a>
        <a href="#instructores">Instructores</a>
        <a href="#faq">FAQ</a>
      </nav>
      <RouterLink to="/inscripcion" class="btn-gold cta">Inscríbete</RouterLink>
    </div>
  </header>
</template>

<style scoped>
.nav {
  position: sticky;
  top: 0;
  z-index: 50;
  background: rgba(9, 9, 9, 0.85);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--border);
}
.nav-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  gap: 18px;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  font-family: var(--font-display);
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  font-size: 18px;
}
.brand img { width: 38px; height: 38px; object-fit: cover; border-radius: 4px; }
.links {
  display: flex;
  gap: 24px;
  font-size: 14px;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  font-weight: 600;
}
.links a:hover { color: var(--gold); }
.cta { padding: 11px 22px; font-size: 14px; }
@media (max-width: 720px) {
  .links { display: none; }
}
</style>
