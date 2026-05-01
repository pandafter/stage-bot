<script setup lang="ts">
import { RouterView } from 'vue-router'
import IgBanner from '@/components/ui/IgBanner.vue'
</script>

<template>
  <!-- Sticky IG-browser alert — sigue al usuario en todas las vistas -->
  <IgBanner />
  <RouterView />
</template>
