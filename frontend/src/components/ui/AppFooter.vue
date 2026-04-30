<script setup lang="ts"></script>

<template>
  <footer class="foot">
    <div class="container foot-row">
      <div class="brand">
        <img src="/assets/logo.jpg" alt="Scuderia St4ge" />
        <span>Scudería St4ge</span>
      </div>
      <p>Tocancipá, Cundinamarca · Academia de kartismo</p>
      <p class="copy">© {{ new Date().getFullYear() }} Scudería St4ge. Todos los derechos reservados.</p>
    </div>
  </footer>
</template>

<style scoped>
.foot {
  border-top: 1px solid var(--border);
  padding: 48px 0 32px;
  background: #050505;
}
.foot-row { display: flex; flex-direction: column; gap: 8px; align-items: center; text-align: center; }
.brand { display: flex; align-items: center; gap: 10px; font-family: var(--font-display); font-weight: 800; text-transform: uppercase; }
.brand img { width: 32px; height: 32px; object-fit: cover; border-radius: 4px; }
.copy { color: var(--text-dimmer); font-size: 13px; margin-top: 8px; }
p { color: var(--text-dim); font-size: 14px; }
</style>
