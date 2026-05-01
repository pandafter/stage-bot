<script setup lang="ts">
import AppNav from '@/components/ui/AppNav.vue'
import AppFooter from '@/components/ui/AppFooter.vue'
import LandingDiscountBar from '@/components/ui/LandingDiscountBar.vue'
import HeroSection from '@/components/landing/HeroSection.vue'
import StatsSection from '@/components/landing/StatsSection.vue'
import ProgramSection from '@/components/landing/ProgramSection.vue'
import InstructoresSection from '@/components/landing/InstructoresSection.vue'
import GallerySection from '@/components/landing/GallerySection.vue'
import FaqSection from '@/components/landing/FaqSection.vue'
import CtaSection from '@/components/landing/CtaSection.vue'
import InstagramPostsSection from '@/components/landing/InstagramPostsSection.vue'
</script>

<template>
  <AppNav />
  <main>
    <HeroSection />
    <StatsSection />
    <ProgramSection />
    <InstagramPostsSection />
    <InstructoresSection />
    <GallerySection />
    <FaqSection />
    <CtaSection />
  </main>
  <AppFooter />

  <!-- Barra de descuento fija en el bottom — aparece 1.8s después de cargar -->
  <LandingDiscountBar />
</template>
