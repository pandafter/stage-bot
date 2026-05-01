import { ref, computed, onMounted, onUnmounted } from 'vue'

const STORAGE_KEY = 'st4ge_discount_expiry'
const DISCOUNT_DURATION_MS = 10 * 60 * 1000 // 10 minutos

// 💰 Puedes luego reemplazar esto con backend
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

  /** 🔥 Progress de 1 → 0 */
  const progress = computed(() => {
    const total = DISCOUNT_DURATION_MS / 1000
    return Math.max(0, secondsLeft.value / total)
  })

  /** 🔥 Estados de urgencia */
  const urgency = computed<'calm' | 'warning' | 'critical'>(() => {
    if (secondsLeft.value > 300) return 'calm'      // >5 min
    if (secondsLeft.value > 60) return 'warning'    // 1-5 min
    return 'critical'                               // <1 min
  })

  // =========================
  // 🧠 STORAGE (expiry)
  // =========================

  function getExpiryTime(): number {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (raw) {
        const t = parseInt(raw, 10)
        if (!isNaN(t) && t > 0) return t
      }
    } catch {}
    return 0
  }

  function setExpiryTime(t: number) {
    try {
      localStorage.setItem(STORAGE_KEY, String(t))
    } catch {}
  }

  function clearExpiry() {
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch {}
  }

  // =========================
  // ⏱️ CORE TIMER
  // =========================

  function tick() {
    const expiry = getExpiryTime()

    if (!expiry) {
      expired.value = true
      secondsLeft.value = 0
      return
    }

    const remaining = expiry - Date.now()

    if (remaining <= 0) {
      expired.value = true
      secondsLeft.value = 0

      if (interval) {
        clearInterval(interval)
        interval = null
      }
    } else {
      expired.value = false
      secondsLeft.value = Math.ceil(remaining / 1000)
    }
  }

  // =========================
  // 🚀 CONTROL
  // =========================

  /** Inicia o reanuda */
  function startTimer() {
    const existing = getExpiryTime()

    if (!existing) {
      const expiry = Date.now() + DISCOUNT_DURATION_MS
      setExpiryTime(expiry)
    }

    tick()

    if (!interval && !expired.value) {
      interval = setInterval(tick, 1000)
    }
  }

  /** Reset manual (testing o lógica de negocio) */
  function resetTimer() {
    clearExpiry()

    const expiry = Date.now() + DISCOUNT_DURATION_MS
    setExpiryTime(expiry)

    expired.value = false
    tick()

    if (!interval) {
      interval = setInterval(tick, 1000)
    }
  }

  // =========================
  // 🔄 LIFECYCLE
  // =========================

  onMounted(() => {
    const existing = getExpiryTime()

    if (existing) {
      tick()

      if (!expired.value && !interval) {
        interval = setInterval(tick, 1000)
      }
    }
  })

  onUnmounted(() => {
    if (interval) {
      clearInterval(interval)
      interval = null
    }
  })

  // =========================
  // 🎯 RETURN
  // =========================

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