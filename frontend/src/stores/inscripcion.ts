import { defineStore } from 'pinia'
import type { CreateInscripcionRequest } from '@/types/api'

interface Result {
  id: string
  checkout_url?: string
  requires_comprobante: boolean
}

export const useInscripcionStore = defineStore('inscripcion', {
  state: () => ({
    step: 1 as 1 | 2 | 3 | 4,
    form: {} as Partial<CreateInscripcionRequest>,
    errors: {} as Record<string, string>,
    submitting: false,
    result: null as Result | null,
  }),
  actions: {
    next() {
      if (this.step < 4) this.step = (this.step + 1) as 1 | 2 | 3 | 4
    },
    prev() {
      if (this.step > 1) this.step = (this.step - 1) as 1 | 2 | 3 | 4
    },
    goTo(s: 1 | 2 | 3 | 4) {
      this.step = s
    },
    update<K extends keyof CreateInscripcionRequest>(k: K, v: CreateInscripcionRequest[K]) {
      this.form[k] = v
    },
    setErrors(e: Record<string, string>) {
      this.errors = e
    },
    reset() {
      this.step = 1
      this.form = {}
      this.errors = {}
      this.result = null
    },
  },
})
