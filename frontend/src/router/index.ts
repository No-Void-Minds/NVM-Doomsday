import { createRouter, createWebHistory } from 'vue-router'

import InvitePage from '@/pages/InvitePage.vue'

declare module 'vue-router' {
  interface RouteMeta {
    title: string
  }
}

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
    {
      path: '/_komponen',
      name: 'dev-components',
      component: () => import('@/pages/DevComponentsPage.vue'),
      meta: { title: 'Contoh komponen' },
    },
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
