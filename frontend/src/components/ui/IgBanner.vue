<script setup lang="ts">
import { useInstagramBrowser } from '@/composables/useInstagramBrowser'
import { ref, computed } from 'vue'

const { isInstagram } = useInstagramBrowser()
const dismissed = ref(false)
const showHelp = ref(false)

// Try sessionStorage so banner doesn't reappear during same session
try {
  if (sessionStorage.getItem('ig-banner-dismissed') === '1') dismissed.value = true
} catch { /* ignore */ }

function dismiss() {
  dismissed.value = true
  try { sessionStorage.setItem('ig-banner-dismissed', '1') } catch { /* ignore */ }
}

const currentUrl = computed(() => {
  if (typeof window === 'undefined') return ''
  return window.location.href
})

/**
 * Attempt to force-open the current URL in the device's default browser.
 * - Android: intent:// scheme works in most IG webviews
 * - iOS: No reliable programmatic way, so we show instructions
 * - Fallback: window.open with _system target (works in some webviews)
 */
function openInBrowser() {
  const url = currentUrl.value
  if (!url) return

  const ua = navigator.userAgent || ''
  const isAndroid = /android/i.test(ua)

  if (isAndroid) {
    // Android intent scheme — opens Chrome/default browser
    const intentUrl = `intent://${url.replace(/^https?:\/\//, '')}#Intent;scheme=https;package=com.android.chrome;end`
    window.location.href = intentUrl
    return
  }

  // iOS: no reliable intent. Try window.open, then show instructions as fallback.
  const win = window.open(url, '_blank')
  if (!win) {
    showHelp.value = true
  }
}
</script>

<template>
  <div v-if="isInstagram && !dismissed" class="ig-banner">
    <div class="ig-main">
      <div class="ig-icon">📱</div>
      <div class="ig-body">
        <p class="ig-title">Estás en el navegador de Instagram</p>
        <p class="ig-sub">Para una mejor experiencia de pago, abre en tu navegador.</p>
      </div>
      <button @click="openInBrowser" class="ig-open-btn">Abrir en navegador →</button>
      <button @click="dismiss" class="ig-close" aria-label="Cerrar">×</button>
    </div>

    <!-- Help overlay for iOS -->
    <div v-if="showHelp" class="ig-help">
      <div class="ig-help-card">
        <h4>Cómo abrir en tu navegador</h4>
        <ol>
          <li>Toca los <strong>tres puntos ⋯</strong> en la esquina superior derecha</li>
          <li>Selecciona <strong>"Abrir en navegador"</strong> o <strong>"Open in browser"</strong></li>
        </ol>
        <div class="ig-help-visual">
          <span class="dots-icon">⋯</span>
          <span class="arrow-down">↓</span>
          <span class="option-text">"Abrir en navegador"</span>
        </div>
        <div class="ig-help-actions">
          <button @click="showHelp = false" class="ig-help-btn">Entendido</button>
          <button @click="dismiss" class="ig-help-dismiss">Continuar aquí</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ig-banner {
  position: sticky;
  top: 0;
  z-index: 200;
}
.ig-main {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  background: linear-gradient(135deg, #1a1a1a 0%, #0d0d0d 100%);
  border-bottom: 2px solid var(--gold);
}
.ig-icon { font-size: 22px; flex-shrink: 0; }
.ig-body { flex: 1; min-width: 0; }
.ig-title {
  font-family: var(--font-display);
  font-weight: 800;
  font-size: 13px;
  text-transform: uppercase;
  letter-spacing: .06em;
  color: var(--gold);
}
.ig-sub {
  font-size: 12px;
  color: var(--text-dim);
  margin-top: 2px;
}
.ig-open-btn {
  font-family: var(--font-display);
  font-weight: 800;
  text-transform: uppercase;
  font-size: 12px;
  letter-spacing: 0.06em;
  padding: 8px 16px;
  background: var(--gold);
  color: #000;
  white-space: nowrap;
  clip-path: polygon(6px 0%, 100% 0%, calc(100% - 6px) 100%, 0% 100%);
  transition: transform .15s;
  flex-shrink: 0;
}
.ig-open-btn:hover { transform: translateY(-1px); }
.ig-close {
  font-size: 20px;
  line-height: 1;
  padding: 4px 6px;
  color: var(--text-dimmer);
  flex-shrink: 0;
}
.ig-close:hover { color: var(--text); }

/* Help overlay */
.ig-help {
  position: fixed;
  inset: 0;
  z-index: 300;
  background: rgba(0,0,0,0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.ig-help-card {
  background: var(--surface);
  border: 1px solid var(--border2);
  padding: 28px 24px;
  max-width: 380px;
  width: 100%;
  text-align: center;
}
.ig-help-card h4 {
  font-size: 20px;
  color: var(--gold);
  margin-bottom: 16px;
}
.ig-help-card ol {
  text-align: left;
  padding-left: 20px;
  font-size: 14px;
  color: var(--text-dim);
  line-height: 1.8;
  margin-bottom: 20px;
}
.ig-help-card strong { color: var(--text); }
.ig-help-visual {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 16px;
  background: rgba(245,193,0,0.06);
  border: 1px dashed var(--gold);
  margin-bottom: 20px;
  font-size: 14px;
  color: var(--text-dim);
}
.dots-icon {
  font-size: 28px;
  color: var(--text);
  background: var(--surface2);
  padding: 4px 10px;
  border: 1px solid var(--border2);
  border-radius: 4px;
}
.arrow-down { font-size: 20px; color: var(--gold); }
.option-text { color: var(--gold); font-weight: 600; }
.ig-help-actions { display: flex; gap: 10px; }
.ig-help-btn {
  flex: 1;
  font-family: var(--font-display);
  font-weight: 800;
  text-transform: uppercase;
  font-size: 14px;
  letter-spacing: .06em;
  padding: 12px 20px;
  background: var(--gold);
  color: #000;
  clip-path: polygon(8px 0%, 100% 0%, calc(100% - 8px) 100%, 0% 100%);
}
.ig-help-dismiss {
  flex: 1;
  font-size: 13px;
  padding: 12px;
  color: var(--text-dim);
  border: 1px solid var(--border2);
}
.ig-help-dismiss:hover { border-color: var(--gold); color: var(--gold); }

@media (max-width: 600px) {
  .ig-main { flex-wrap: wrap; gap: 8px; }
  .ig-open-btn { width: 100%; text-align: center; order: 10; }
}
</style>
