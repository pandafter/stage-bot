import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/services/api'

export interface HeroData {
  title?: string
  subtitle?: string
  cta_text?: string
  cta_href?: string
  bg_url?: string
}

export interface StatItem {
  value: string
  label: string
}

export interface InstructorItem {
  name: string
  role: string
  bio_html: string
  photo_url?: string
}

export interface FaqItem {
  q: string
  a_html: string
}

export interface CtaData {
  title?: string
  subtitle?: string
  cta_text?: string
  cta_href?: string
  bg_url?: string
}

export const useLandingStore = defineStore('landing', () => {
  const cms = ref<Record<string, any>>({})
  const loaded = ref(false)

  async function load() {
    if (loaded.value) return
    try {
      const res = await api.get('/config')
      cms.value = res.data?.cms ?? {}
    } catch {
      // silently fall back to hardcoded defaults in components
    } finally {
      loaded.value = true
    }
  }

  function hero(): HeroData {
    return cms.value['hero'] ?? {}
  }

  function stats(): StatItem[] {
    return cms.value['stats']?.items ?? []
  }

  function instructores(): InstructorItem[] {
    return cms.value['instructores']?.items ?? []
  }

  function faq(): FaqItem[] {
    return cms.value['faq']?.items ?? []
  }

  function cta(): CtaData {
    return cms.value['cta'] ?? {}
  }

  return { cms, loaded, load, hero, stats, instructores, faq, cta }
})
