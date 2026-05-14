import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'home', component: () => import('@/views/HomeView.vue') },
  { path: '/inscripcion', name: 'inscripcion', component: () => import('@/views/InscripcionView.vue') },
  { path: '/success', name: 'success', component: () => import('@/views/SuccessView.vue') },
  { path: '/callback', name: 'callback', component: () => import('@/views/CallbackView.vue') },

  // Admin routes
  {
    path: '/admin',
    redirect: '/admin/dashboard',
  },
  {
    path: '/admin/login',
    name: 'admin-login',
    component: () => import('@/views/admin/LoginView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/admin/:pathMatch(.*)*',
    component: () => import('@/views/admin/AdminLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: 'dashboard',
        name: 'admin-dashboard',
        component: () => import('@/views/admin/DashboardView.vue'),
      },
      {
        path: 'landing',
        name: 'admin-landing',
        component: () => import('@/views/admin/CmsLandingView.vue'),
      },
      {
        path: 'media',
        name: 'admin-media',
        component: () => import('@/views/admin/CmsMediaView.vue'),
      },
      {
        path: 'form',
        name: 'admin-form',
        component: () => import('@/views/admin/FormConfigView.vue'),
      },
      {
        path: 'inscripciones',
        name: 'admin-inscripciones',
        component: () => import('@/views/admin/InscripcionesView.vue'),
      },
      {
        path: 'inscripciones/:id',
        name: 'admin-inscripcion-detail',
        component: () => import('@/views/admin/InscripcionDetailView.vue'),
      },
      {
        path: 'theme',
        name: 'admin-theme',
        component: () => import('@/views/admin/ThemeView.vue'),
      },
    ],
  },

  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 }
  },
})

router.beforeEach((to, _from, next) => {
  const requiresAuth = to.matched.some(r => r.meta?.requiresAuth)
  if (requiresAuth) {
    const token = localStorage.getItem('admin_token')
    if (!token) {
      next('/admin/login')
      return
    }
  }
  next()
})

export default router
