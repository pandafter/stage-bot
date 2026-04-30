<script setup lang="ts">
import { useInstagramBrowser } from '@/composables/useInstagramBrowser'
import { ref } from 'vue'

const { isInstagram } = useInstagramBrowser()
const dismissed = ref(false)

function openExternal() {
  // Most IG webviews honor a tel:/ intent... not for URLs. Best we can do is
  // tell the user how to open externally.
  alert('Toca el menú "..." en la esquina superior y selecciona "Abrir en navegador" para una mejor experiencia de pago.')
}
</script>

<template>
  <div v-if="isInstagram && !dismissed" class="ig-banner">
    <span>Estás dentro de Instagram. Para mejores resultados, abre en tu navegador.</span>
    <button @click="openExternal" class="ig-cta">Cómo abrir</button>
    <button @click="dismissed = true" class="ig-close" aria-label="Cerrar">×</button>
  </div>
</template>

<style scoped>
.ig-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 18px;
  background: var(--gold);
  color: #000;
  font-size: 13px;
  font-weight: 600;
  position: sticky;
  top: 0;
  z-index: 100;
}
.ig-banner span { flex: 1; }
.ig-cta {
  font-family: var(--font-display);
  font-weight: 800;
  text-transform: uppercase;
  font-size: 12px;
  letter-spacing: 0.06em;
  padding: 6px 12px;
  background: #000;
  color: var(--gold);
}
.ig-close {
  font-size: 20px;
  line-height: 1;
  padding: 4px 8px;
}
</style>
