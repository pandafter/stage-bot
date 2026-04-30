import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: () => import('@/views/HomeView.vue') },
    { path: '/inscripcion', name: 'inscripcion', component: () => import('@/views/InscripcionView.vue') },
    { path: '/success', name: 'success', component: () => import('@/views/SuccessView.vue') },
    { path: '/callback', name: 'callback', component: () => import('@/views/CallbackView.vue') },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
  scrollBehavior() {
    return { top: 0 }
  },
})

export default router
