<script setup lang="ts">
import IgBanner from '@/components/ui/IgBanner.vue'
import AppNav from '@/components/ui/AppNav.vue'
import AppFooter from '@/components/ui/AppFooter.vue'
import HeroSection from '@/components/landing/HeroSection.vue'
import StatsSection from '@/components/landing/StatsSection.vue'
import ProgramSection from '@/components/landing/ProgramSection.vue'
import InstructoresSection from '@/components/landing/InstructoresSection.vue'
import GallerySection from '@/components/landing/GallerySection.vue'
import FaqSection from '@/components/landing/FaqSection.vue'
import CtaSection from '@/components/landing/CtaSection.vue'
</script>

<template>
  <IgBanner />
  <AppNav />
  <main>
    <HeroSection />
    <StatsSection />
    <ProgramSection />
    <InstructoresSection />
    <GallerySection />
    <FaqSection />
    <CtaSection />
  </main>
  <AppFooter />
</template>
