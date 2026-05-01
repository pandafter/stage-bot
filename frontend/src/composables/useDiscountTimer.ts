import { ref, computed, onMounted, onUnmounted } from 'vue'

// ══════════════════════════════════════════════════════════════════
//  Precios
// ══════════════════════════════════════════════════════════════════
const PRICE_FULL = 890_000
const PRICE_DISCOUNT = 730_000

// ══════════════════════════════════════════════════════════════════
//  Duración de cada fase (10 minutos)
// ══════════════════════════════════════════════════════════════════
const PHASE_MS = 10 * 60 * 1000

// ══════════════════════════════════════════════════════════════════
//  Estado compartido a nivel de módulo.
//  → Persiste entre navegaciones SPA (landing ↔ form ↔ callback).
//  → Se RESETEA en hard refresh (F5 / cerrar pestaña) porque es RAM pura.
// ══════════════════════════════════════════════════════════════════
let _startTime: number | null = null

/**
 * Fases del descuento:
 *   1 → Primer countdown (10 min) — precio descuento
 *   2 → Segunda oportunidad (+10 min) — todavía precio descuento
 *   3 → Expirado — precio cambia a completo
 */
type Phase = 1 | 2 | 3

export function useDiscountTimer() {
  const secondsLeft = ref(0)
  const phase = ref<Phase>(1)
  const expired = ref(false)
  /** true durante los primeros 3 segundos de fase 2 (para animación flash) */
  const secondChanceFlash = ref(false)
  let interval: ReturnType<typeof setInterval> | null = null
  let flashTimeout: ReturnType<typeof setTimeout> | null = null

  // ── Computed ──────────────────────────────────────────────────
  const minutes = computed(() => Math.floor(secondsLeft.value / 60))
  const seconds = computed(() => secondsLeft.value % 60)

  const priceFull = PRICE_FULL
  const priceDiscount = PRICE_DISCOUNT

  /** Precio efectivo: descuento en fase 1-2, completo en fase 3 */
  const effectivePrice = computed(() =>
    expired.value ? PRICE_FULL : PRICE_DISCOUNT
  )

  const discountPct = computed(() =>
    Math.round(((PRICE_FULL - PRICE_DISCOUNT) / PRICE_FULL) * 100)
  )

  /** Barra de progreso relativa a la fase actual (1 → 0) */
  const progress = computed(() => {
    const phaseSecs = PHASE_MS / 1000
    return Math.max(0, secondsLeft.value / phaseSecs)
  })

  /** Urgencia visual */
  const urgency = computed<'calm' | 'warning' | 'critical'>(() => {
    if (phase.value === 2) {
      // Fase 2 arranca más urgente
      if (secondsLeft.value > 180) return 'warning'   // >3 min → naranja
      return 'critical'                                 // ≤3 min → rojo
    }
    // Fase 1 normal
    if (secondsLeft.value > 300) return 'calm'          // >5 min
    if (secondsLeft.value > 60) return 'warning'        // 1-5 min
    return 'critical'                                    // <1 min
  })

  // ── Tick ──────────────────────────────────────────────────────
  function tick() {
    if (!_startTime) {
      expired.value = true
      phase.value = 3
      secondsLeft.value = 0
      return
    }

    const elapsed = Date.now() - _startTime

    if (elapsed < PHASE_MS) {
      // ── Fase 1 ──
      phase.value = 1
      expired.value = false
      secondsLeft.value = Math.ceil((PHASE_MS - elapsed) / 1000)

    } else if (elapsed < PHASE_MS * 2) {
      // ── Fase 2: segunda oportunidad ──
      if (phase.value === 1) {
        // Transición 1→2: flash visual
        phase.value = 2
        secondChanceFlash.value = true
        flashTimeout = setTimeout(() => { secondChanceFlash.value = false }, 3000)
      }
      phase.value = 2
      expired.value = false
      secondsLeft.value = Math.ceil((PHASE_MS * 2 - elapsed) / 1000)

    } else {
      // ── Fase 3: expirado ──
      phase.value = 3
      expired.value = true
      secondsLeft.value = 0
      stopInterval()
    }
  }

  function stopInterval() {
    if (interval) { clearInterval(interval); interval = null }
  }

  // ── Control ───────────────────────────────────────────────────

  /** Inicia el countdown (idempotente: si ya está corriendo, no hace nada). */
  function startTimer() {
    if (!_startTime) _startTime = Date.now()
    tick()
    if (!interval && !expired.value) {
      interval = setInterval(tick, 1000)
    }
  }

  /** Reset total — vuelve a fase 1 con 10 min (solo para testing). */
  function resetTimer() {
    _startTime = Date.now()
    phase.value = 1
    expired.value = false
    secondChanceFlash.value = false
    tick()
    if (!interval) {
      interval = setInterval(tick, 1000)
    }
  }

  // ── Lifecycle ─────────────────────────────────────────────────

  onMounted(() => {
    // Si el timer ya está corriendo (navegación SPA), reanudar
    if (_startTime) {
      tick()
      if (!expired.value && !interval) {
        interval = setInterval(tick, 1000)
      }
    }
  })

  onUnmounted(() => {
    stopInterval()
    if (flashTimeout) { clearTimeout(flashTimeout); flashTimeout = null }
  })

  // ── Return ────────────────────────────────────────────────────
  return {
    secondsLeft,
    minutes,
    seconds,
    phase,
    expired,
    secondChanceFlash,
    progress,
    urgency,
    effectivePrice,
    priceFull,
    priceDiscount,
    discountPct,
    startTimer,
    resetTimer,
  }
}
