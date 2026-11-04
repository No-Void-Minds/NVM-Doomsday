import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

import InvitePage from '@/pages/InvitePage.vue'

declare module 'vue-router' {
  interface RouteMeta {
    title: string
  }
}

// Hanya ada saat development. import.meta.env.DEV bernilai konstan false saat build, jadi Vite membuang
// route ini beserta file halamannya dari hasil produksi.
const devRoutes: RouteRecordRaw[] = import.meta.env.DEV
  ? [
      {
        path: '/_komponen',
        name: 'dev-components',
        component: () => import('@/pages/DevComponentsPage.vue'),
        meta: { title: 'Contoh komponen' },
      },
    ]
  : []

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'invite',
      component: InvitePage,
      meta: { title: 'No Void Minds Doomsday' },
    },
    {
      path: '/admin',
      name: 'admin',
      // Dimuat terpisah supaya pengunjung undangan tidak ikut mengunduh kode halaman admin.
      component: () => import('@/pages/AdminPage.vue'),
      meta: { title: 'Admin Doomsday' },
    },
    ...devRoutes,
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/pages/NotFoundPage.vue'),
      meta: { title: 'Halaman tidak ditemukan' },
    },
  ],
})

router.afterEach((to) => {
  document.title = to.meta.title
})

export default router
