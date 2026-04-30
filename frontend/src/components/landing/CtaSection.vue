<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <section class="section cta">
    <div class="container clip-card big">
      <span class="eyebrow">Cupos limitados</span>
      <h2>El kartódromo<br /><span class="gold">te espera</span></h2>
      <p>
        Asegura tu cupo con la reserva de $150.000. Te confirmamos por WhatsApp y
        te enviamos toda la info logística del fin de semana.
      </p>
      <RouterLink to="/inscripcion" class="btn-gold">Inscribirme ahora →</RouterLink>
    </div>
  </section>
</template>

<style scoped>
.cta { padding: 60px 0 100px; }
.big { padding: 56px 40px; text-align: center; }
.gold { color: var(--gold); }
h2 { font-size: clamp(36px, 6vw, 64px); margin: 12px 0 18px; }
p { color: var(--text-dim); max-width: 560px; margin: 0 auto 28px; line-height: 1.55; }
</style>
