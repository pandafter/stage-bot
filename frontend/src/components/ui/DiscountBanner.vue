<script setup lang="ts">
import { useDiscountTimer } from '@/composables/useDiscountTimer'

const {
  minutes, seconds, expired, urgency, progress,
  priceFull, priceDiscount, discountPct,
} = useDiscountTimer()

function fmt(n: number): string { return n.toString().padStart(2, '0') }
function formatCOP(n: number): string { return n.toLocaleString('es-CO') }
</script>

<template>
  <!-- Active discount -->
  <div v-if="!expired" class="discount-bar" :class="urgency">
    <div class="discount-progress" :style="{ width: (progress * 100) + '%' }" />
    <div class="discount-content">
      <div class="discount-left">
        <span class="discount-badge">-{{ discountPct }}%</span>
        <span class="discount-text">
          <span class="discount-old">${{ formatCOP(priceFull) }}</span>
          <span class="discount-arrow">→</span>
          <span class="discount-new">${{ formatCOP(priceDiscount) }}</span>
        </span>
      </div>
      <div class="discount-timer">
        <span class="timer-label">Descuento termina en</span>
        <div class="timer-digits">
          <span class="digit">{{ fmt(minutes) }}</span>
          <span class="separator">:</span>
          <span class="digit">{{ fmt(seconds) }}</span>
        </div>
      </div>
    </div>
  </div>

  <!-- Expired -->
  <div v-else class="discount-bar expired">
    <div class="discount-content">
      <div class="discount-left">
        <span class="discount-badge off">Tiempo agotado</span>
        <span class="discount-text">
          El precio de preventa sigue vigente — <strong>¡completa tu inscripción!</strong>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.discount-bar {
  position: relative;
  overflow: hidden;
  border: 1px solid;
  margin-bottom: 28px;
}

.discount-bar.calm {
  border-color: rgba(245,193,0,0.3);
  background: rgba(245,193,0,0.06);
}
.discount-bar.warning {
  border-color: rgba(255,165,0,0.4);
  background: rgba(255,165,0,0.08);
}
.discount-bar.critical {
  border-color: rgba(230,50,0,0.4);
  background: rgba(230,50,0,0.08);
  animation: pulse-border 1s ease-in-out infinite;
}
.discount-bar.expired {
  border-color: var(--border2);
  background: var(--surface);
}

@keyframes pulse-border {
  0%, 100% { border-color: rgba(230,50,0,0.4); }
  50% { border-color: rgba(230,50,0,0.8); }
}

.discount-progress {
  position: absolute;
  top: 0;
  left: 0;
  height: 3px;
  transition: width 1s linear;
}
.calm .discount-progress { background: var(--gold); }
.warning .discount-progress { background: orange; }
.critical .discount-progress { background: var(--red); }

.discount-content {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  gap: 16px;
  flex-wrap: wrap;
}

.discount-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.discount-badge {
  font-family: var(--font-display);
  font-weight: 900;
  font-size: 14px;
  letter-spacing: .06em;
  text-transform: uppercase;
  padding: 4px 12px;
  clip-path: polygon(4px 0%, 100% 0%, calc(100% - 4px) 100%, 0% 100%);
}
.calm .discount-badge { background: var(--gold); color: #000; }
.warning .discount-badge { background: orange; color: #000; }
.critical .discount-badge { background: var(--red); color: #fff; }
.discount-badge.off { background: var(--surface2); color: var(--text-dim); clip-path: none; padding: 4px 10px; font-size: 12px; }

.discount-text {
  font-size: 14px;
  color: var(--text-dim);
  display: flex;
  align-items: center;
  gap: 8px;
}
.discount-old {
  text-decoration: line-through;
  color: var(--text-dimmer);
  font-size: 15px;
}
.discount-arrow { color: var(--text-dimmer); font-size: 13px; }
.discount-new {
  font-family: var(--font-display);
  font-weight: 900;
  font-size: 22px;
  color: var(--gold);
}
.critical .discount-new { color: var(--red); }

.discount-timer {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}
.timer-label {
  font-size: 10px;
  letter-spacing: .15em;
  text-transform: uppercase;
  color: var(--text-dimmer);
}
.timer-digits {
  display: flex;
  align-items: center;
  gap: 2px;
}
.digit {
  font-family: var(--font-display);
  font-weight: 900;
  font-size: 28px;
  line-height: 1;
  min-width: 38px;
  text-align: center;
  padding: 4px 6px;
  background: rgba(0,0,0,0.3);
  border: 1px solid var(--border2);
}
.calm .digit { color: var(--gold); }
.warning .digit { color: orange; }
.critical .digit { color: var(--red); animation: blink 1s step-end infinite; }
.separator {
  font-family: var(--font-display);
  font-weight: 900;
  font-size: 24px;
  color: var(--text-dimmer);
}
.critical .separator { animation: blink 1s step-end infinite; }

@keyframes blink {
  50% { opacity: 0.3; }
}

@media (max-width: 600px) {
  .discount-content { flex-direction: column; align-items: stretch; gap: 12px; }
  .discount-timer { align-items: center; }
  .discount-left { justify-content: center; }
}
</style>
