<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { useDiscountTimer } from '@/composables/useDiscountTimer'

const {
  minutes, seconds, expired, phase, urgency, progress,
  priceFull, priceDiscount, discountPct, secondChanceFlash, startTimer,
} = useDiscountTimer()

const visible = ref(false)
const dismissed = ref(false)

function fmt(n: number): string { return n.toString().padStart(2, '0') }
function formatCOP(n: number): string { return n.toLocaleString('es-CO') }

onMounted(() => {
  startTimer()
  setTimeout(() => { visible.value = true }, 1800)
})

function dismiss() { dismissed.value = true }
</script>

<template>
  <Transition name="slide-up">
    <div
      v-if="visible && !dismissed && !expired"
      class="ldb"
      :class="urgency"
    >
      <!-- Progress bar line at the very top of the strip -->
      <div class="ldb-progress" :style="{ width: (progress * 100) + '%' }" />

      <div class="ldb-inner">

        <!-- Badge + precio -->
        <div class="ldb-prices">
          <span v-if="phase === 1" class="ldb-badge">-{{ discountPct }}%</span>
          <span v-else class="ldb-badge flash">¡Última oportunidad!</span>
          <span class="ldb-old">${{ formatCOP(priceFull) }}</span>
          <span class="ldb-arrow">→</span>
          <span class="ldb-new">${{ formatCOP(priceDiscount) }}</span>
          <span class="ldb-currency">COP</span>
        </div>

        <!-- Timer -->
        <div class="ldb-timer">
          <span class="ldb-timer-label">
            {{ phase === 1 ? 'Termina en' : '¡Quedan solo!' }}
          </span>
          <div class="ldb-digits">
            <span class="ldb-digit">{{ fmt(minutes) }}</span>
            <span class="ldb-sep">:</span>
            <span class="ldb-digit">{{ fmt(seconds) }}</span>
          </div>
        </div>

        <!-- CTA -->
        <RouterLink to="/inscripcion" class="ldb-cta">
          Reservar cupo →
        </RouterLink>

        <!-- Dismiss -->
        <button class="ldb-close" @click="dismiss" aria-label="Cerrar">×</button>
      </div>

      <!-- Flash overlay de segunda oportunidad -->
      <Transition name="flash-in">
        <div v-if="secondChanceFlash" class="ldb-flash-overlay">
          <span>🔥 ¡Te damos 10 minutos más!</span>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<style scoped>
.ldb {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 150;
  overflow: hidden;
  box-shadow: 0 -4px 32px rgba(0,0,0,0.6);
}

/* Urgency backgrounds */
.ldb.calm    { background: #0e0e0e; border-top: 2px solid rgba(245,193,0,0.4); }
.ldb.warning { background: #0e0a00; border-top: 2px solid rgba(255,165,0,0.5); }
.ldb.critical {
  background: #120500;
  border-top: 2px solid rgba(230,50,0,0.6);
  animation: pulse-bg 1s ease-in-out infinite;
}
@keyframes pulse-bg {
  0%,100% { border-top-color: rgba(230,50,0,0.6); }
  50%      { border-top-color: rgba(230,50,0,1); }
}

/* Progress line */
.ldb-progress {
  position: absolute;
  top: 0; left: 0;
  height: 3px;
  transition: width 1s linear;
}
.calm    .ldb-progress { background: var(--gold); }
.warning .ldb-progress { background: orange; }
.critical .ldb-progress { background: var(--red); }

/* Inner layout */
.ldb-inner {
  position: relative;
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 12px 20px 12px 24px;
  flex-wrap: wrap;
  max-width: 1100px;
  margin: 0 auto;
}

/* Prices */
.ldb-prices {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.ldb-badge {
  font-family: var(--font-display);
  font-weight: 900;
  font-size: 13px;
  letter-spacing: .06em;
  text-transform: uppercase;
  padding: 3px 10px;
  clip-path: polygon(4px 0%,100% 0%,calc(100% - 4px) 100%,0% 100%);
}
.calm    .ldb-badge { background: var(--gold); color: #000; }
.warning .ldb-badge { background: orange; color: #000; }
.critical .ldb-badge { background: var(--red); color: #fff; }

.ldb-old {
  font-size: 14px;
  text-decoration: line-through;
  color: var(--text-dimmer);
}
.ldb-arrow { color: var(--text-dimmer); font-size: 13px; }
.ldb-new {
  font-family: var(--font-display);
  font-weight: 900;
  font-size: 22px;
  line-height: 1;
}
.calm    .ldb-new { color: var(--gold); }
.warning .ldb-new { color: orange; }
.critical .ldb-new { color: #ff6b4a; }

.ldb-currency {
  font-size: 12px;
  color: var(--text-dimmer);
  align-self: flex-end;
  margin-bottom: 2px;
}

/* Timer */
.ldb-timer {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}
.ldb-timer-label {
  font-size: 9px;
  letter-spacing: .15em;
  text-transform: uppercase;
  color: var(--text-dimmer);
  line-height: 1;
}
.ldb-digits {
  display: flex;
  align-items: center;
  gap: 2px;
}
.ldb-digit {
  font-family: var(--font-display);
  font-weight: 900;
  font-size: 26px;
  line-height: 1;
  min-width: 34px;
  text-align: center;
  padding: 3px 5px;
  background: rgba(0,0,0,0.5);
  border: 1px solid var(--border2);
  font-variant-numeric: tabular-nums;
}
.calm    .ldb-digit { color: var(--gold); }
.warning .ldb-digit { color: orange; }
.critical .ldb-digit { color: var(--red); animation: blink 1s step-end infinite; }
.ldb-sep {
  font-family: var(--font-display);
  font-weight: 900;
  font-size: 22px;
  color: var(--text-dimmer);
  padding: 0 1px;
}

/* CTA */
.ldb-cta {
  font-family: var(--font-display);
  font-weight: 900;
  text-transform: uppercase;
  letter-spacing: .08em;
  font-size: 14px;
  padding: 12px 28px;
  clip-path: polygon(8px 0%,100% 0%,calc(100% - 8px) 100%,0% 100%);
  white-space: nowrap;
  transition: transform .15s, filter .15s;
  flex-shrink: 0;
  margin-left: auto;
}
.calm    .ldb-cta { background: var(--gold); color: #000; }
.warning .ldb-cta { background: orange; color: #000; }
.critical .ldb-cta { background: var(--red); color: #fff; }
.ldb-cta:hover { transform: translateY(-2px); filter: brightness(1.1); }

/* Dismiss */
.ldb-close {
  font-size: 20px;
  line-height: 1;
  padding: 4px 6px;
  color: var(--text-dimmer);
  flex-shrink: 0;
}
.ldb-close:hover { color: var(--text); }

/* Slide-up transition */
.slide-up-enter-active { transition: transform .4s cubic-bezier(0.22,1,0.36,1), opacity .3s; }
.slide-up-leave-active { transition: transform .3s ease-in, opacity .25s; }
.slide-up-enter-from  { transform: translateY(100%); opacity: 0; }
.slide-up-leave-to    { transform: translateY(100%); opacity: 0; }

/* Mobile */
@media (max-width: 640px) {
  .ldb-inner {
    padding: 10px 14px;
    gap: 10px;
  }
  .ldb-prices { gap: 6px; }
  .ldb-old, .ldb-arrow, .ldb-currency { display: none; }
  .ldb-new { font-size: 20px; }
  .ldb-digit { font-size: 22px; min-width: 28px; }
  .ldb-cta {
    font-size: 13px;
    padding: 10px 20px;
    margin-left: auto;
  }
}

@keyframes blink { 50% { opacity: 0.3; } }

/* Badge flash for second chance */
.ldb-badge.flash {
  animation: badge-flash .6s ease-in-out 3;
}
@keyframes badge-flash {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; transform: scale(1.05); }
}

/* Second chance flash overlay */
.ldb-flash-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(230,50,0,0.95);
  z-index: 5;
  font-family: var(--font-display);
  font-weight: 900;
  font-size: 18px;
  text-transform: uppercase;
  letter-spacing: .08em;
  color: #fff;
}
.flash-in-enter-active { transition: opacity .3s; }
.flash-in-leave-active { transition: opacity .8s ease-out; }
.flash-in-enter-from, .flash-in-leave-to { opacity: 0; }
</style>
