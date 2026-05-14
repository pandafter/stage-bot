<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import axios from 'axios'
import IgBanner from '@/components/ui/IgBanner.vue'

onMounted(async () => {
  try {
    const { data } = await axios.get('/api/config')
    if (data.theme) {
      const root = document.documentElement
      if (data.theme.primary_color) root.style.setProperty('--color-primary', data.theme.primary_color)
      if (data.theme.secondary_color) root.style.setProperty('--color-secondary', data.theme.secondary_color)
      if (data.theme.accent_color) root.style.setProperty('--color-accent', data.theme.accent_color)
      if (data.theme.font_heading) root.style.setProperty('--font-heading', data.theme.font_heading)
      if (data.theme.font_body) root.style.setProperty('--font-body', data.theme.font_body)
    }
  } catch {
    // Silently fail — defaults from CSS apply
  }
})
</script>

<template>
  <!-- Sticky IG-browser alert — sigue al usuario en todas las vistas -->
  <IgBanner />
  <RouterView />
</template>
