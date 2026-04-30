import { ref, onMounted } from 'vue'

export function useInstagramBrowser() {
  const isInstagram = ref(false)
  onMounted(() => {
    if (typeof navigator === 'undefined') return
    const ua = navigator.userAgent || ''
    isInstagram.value = /Instagram/i.test(ua) || /FBAN|FBAV/i.test(ua)
  })
  return { isInstagram }
}
