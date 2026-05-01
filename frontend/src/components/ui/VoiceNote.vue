<script setup lang="ts">
import { ref, onUnmounted, computed } from 'vue'

const props = defineProps<{ src: string; label?: string }>()

const audio = ref<HTMLAudioElement | null>(null)
const playing = ref(false)
const currentTime = ref(0)
const duration = ref(0)
const loaded = ref(false)
const error = ref(false)

function init() {
  if (audio.value) return
  const a = new Audio(props.src)
  a.preload = 'metadata'
  a.onloadedmetadata = () => { duration.value = a.duration; loaded.value = true }
  a.ontimeupdate = () => { currentTime.value = a.currentTime }
  a.onended = () => { playing.value = false; currentTime.value = 0 }
  a.onerror = () => { error.value = true; loaded.value = true }
  audio.value = a
}

function toggle() {
  init()
  if (!audio.value) return
  if (playing.value) {
    audio.value.pause()
    playing.value = false
  } else {
    audio.value.play().catch(() => { error.value = true })
    playing.value = true
  }
}

function seek(e: MouseEvent) {
  if (!audio.value || !duration.value) return
  const bar = e.currentTarget as HTMLElement
  const ratio = e.offsetX / bar.offsetWidth
  audio.value.currentTime = ratio * duration.value
  currentTime.value = audio.value.currentTime
}

const progress = computed(() =>
  duration.value ? (currentTime.value / duration.value) * 100 : 0
)

function fmt(s: number): string {
  if (!isFinite(s)) return '0:00'
  const m = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return `${m}:${sec.toString().padStart(2, '0')}`
}

const displayTime = computed(() =>
  playing.value || currentTime.value > 0
    ? fmt(currentTime.value)
    : fmt(duration.value)
)

onUnmounted(() => {
  audio.value?.pause()
  audio.value = null
})
</script>

<template>
  <div class="vn" :class="{ playing, error }" @click.prevent>
    <!-- Play / Pause button -->
    <button class="vn-btn" @click="toggle" :aria-label="playing ? 'Pausar' : 'Reproducir'">
      <svg v-if="!playing" viewBox="0 0 24 24" fill="currentColor">
        <path d="M8 5v14l11-7z"/>
      </svg>
      <svg v-else viewBox="0 0 24 24" fill="currentColor">
        <path d="M6 19h4V5H6zm8-14v14h4V5z"/>
      </svg>
    </button>

    <!-- Waveform + progress bar -->
    <div class="vn-middle">
      <div class="vn-bar" @click="seek">
        <div class="vn-track">
          <!-- Static waveform bars -->
          <span v-for="h in bars" :key="h.i" class="vn-wave" :style="{ height: h.h + '%' }" />
        </div>
        <!-- Fill overlay -->
        <div class="vn-fill" :style="{ width: progress + '%' }" />
      </div>
      <div class="vn-meta">
        <span class="vn-label">{{ label || 'Mensaje del director' }}</span>
        <span class="vn-time">{{ displayTime }}</span>
      </div>
    </div>

    <!-- Avatar -->
    <img class="vn-avatar" src="/assets/AndresMelo.png" alt="Andrés Melo" />
  </div>
</template>

<script lang="ts">
// Pseudo-random waveform heights (seeded, so they're stable)
const seed = [60, 80, 45, 95, 55, 70, 40, 85, 65, 50, 75, 90, 48, 78, 58, 88, 42, 68, 52, 82, 46, 72, 92, 38, 66]
const bars = seed.map((h, i) => ({ i, h }))
</script>

<style scoped>
.vn {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(245,193,0,0.06);
  border: 1px solid rgba(245,193,0,0.2);
  padding: 12px 14px;
  cursor: default;
  transition: border-color .2s;
}
.vn:hover { border-color: rgba(245,193,0,0.4); }
.vn.error { border-color: rgba(230,50,0,0.3); opacity: 0.6; pointer-events: none; }

/* Play button */
.vn-btn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--gold);
  color: #000;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: transform .15s, background .15s;
}
.vn-btn:hover { transform: scale(1.08); background: #ffd700; }
.vn-btn svg { width: 20px; height: 20px; }

/* Middle */
.vn-middle { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 6px; }

/* Bar + waveform */
.vn-bar {
  position: relative;
  height: 32px;
  cursor: pointer;
}
.vn-track {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 0 2px;
}
.vn-wave {
  flex: 1;
  min-width: 2px;
  max-width: 3px;
  background: var(--border2);
  border-radius: 1px;
  transition: background .2s;
}
.playing .vn-wave { background: rgba(245,193,0,0.35); }

/* Progress fill (clip mask over waveform) */
.vn-fill {
  position: absolute;
  top: 0;
  left: 0;
  height: 100%;
  background: linear-gradient(90deg, rgba(245,193,0,0.6), rgba(245,193,0,0.2));
  pointer-events: none;
  transition: width .1s linear;
  mix-blend-mode: lighten;
}

/* Meta row */
.vn-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.vn-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: .12em;
  color: var(--text-dimmer);
}
.vn-time {
  font-family: var(--font-display);
  font-size: 12px;
  font-weight: 700;
  color: var(--gold);
  font-variant-numeric: tabular-nums;
}

/* Avatar */
.vn-avatar {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--gold);
  flex-shrink: 0;
}
</style>
