import { ref, computed, onMounted, onUnmounted } from 'vue'

const STORAGE_KEY = 'st4ge_discount_start'
const DISCOUNT_DURATION_MS = 10 * 60 * 1000 // 10 minutes

const PRICE_FULL = 890_000
const PRICE_DISCOUNT = 730_000

export function useDiscountTimer() {
  const secondsLeft = ref(0)
  const expired = ref(false)
  let interval: ReturnType<typeof setInterval> | null = null

  const minutes = computed(() => Math.floor(secondsLeft.value / 60))
  const seconds = computed(() => secondsLeft.value % 60)
  const priceFull = PRICE_FULL
  const priceDiscount = PRICE_DISCOUNT
  const discountPct = computed(() =>
    Math.round(((PRICE_FULL - PRICE_DISCOUNT) / PRICE_FULL) * 100)
  )

  /** Progress from 1 (full) → 0 (expired) */
  const progress = computed(() => {
    const total = DISCOUNT_DURATION_MS / 1000
    return Math.max(0, secondsLeft.value / total)
  })

  /** Urgency tiers for styling */
  const urgency = computed<'calm' | 'warning' | 'critical'>(() => {
    if (secondsLeft.value > 300) return 'calm'     // >5 min
    if (secondsLeft.value > 60) return 'warning'    // 1-5 min
    return 'critical'                                // <1 min
  })

  function getStartTime(): number {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (raw) {
        const t = parseInt(raw, 10)
        if (!isNaN(t) && t > 0) return t
      }
    } catch { /* localStorage unavailable */ }
    return 0
  }

  function setStartTime(t: number) {
    try {
      localStorage.setItem(STORAGE_KEY, String(t))
    } catch { /* ignore */ }
  }

  function tick() {
    const start = getStartTime()
    if (!start) {
      expired.value = true
      secondsLeft.value = 0
      return
    }
    const elapsed = Date.now() - start
    const remaining = DISCOUNT_DURATION_MS - elapsed
    if (remaining <= 0) {
      expired.value = true
      secondsLeft.value = 0
      if (interval) { clearInterval(interval); interval = null }
    } else {
      expired.value = false
      secondsLeft.value = Math.ceil(remaining / 1000)
    }
  }

  /** Start (or resume) the countdown. Called when user enters the inscripcion flow. */
  function startTimer() {
    const existing = getStartTime()
    if (!existing) {
      setStartTime(Date.now())
    }
    tick()
    if (!interval && !expired.value) {
      interval = setInterval(tick, 1000)
    }
  }

  /** Reset the timer (e.g. for testing or if user returns after expiry). */
  function resetTimer() {
    try { localStorage.removeItem(STORAGE_KEY) } catch { /* ignore */ }
    expired.value = false
    secondsLeft.value = DISCOUNT_DURATION_MS / 1000
    setStartTime(Date.now())
    tick()
    if (!interval) {
      interval = setInterval(tick, 1000)
    }
  }

  onMounted(() => {
    // Auto-resume if a timer was already running
    const existing = getStartTime()
    if (existing) {
      tick()
      if (!expired.value && !interval) {
        interval = setInterval(tick, 1000)
      }
    }
  })

  onUnmounted(() => {
    if (interval) { clearInterval(interval); interval = null }
  })

  return {
    secondsLeft,
    minutes,
    seconds,
    expired,
    progress,
    urgency,
    priceFull,
    priceDiscount,
    discountPct,
    startTimer,
    resetTimer,
  }
}
