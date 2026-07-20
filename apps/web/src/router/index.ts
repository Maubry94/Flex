import { createRouter, createWebHistory } from 'vue-router'
import { getAuthStatus } from '@/lib/api/auth'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/HomeView.vue'),
    },
    {
      path: '/libraries',
      name: 'libraries',
      component: () => import('@/views/LibrariesView.vue'),
    },
    {
      path: '/favorites',
      name: 'favorites',
      component: () => import('@/views/FavoritesView.vue'),
    },
    { path: '/collections', name: 'collections', component: () => import('@/views/CollectionsView.vue') },
    { path: '/collections/:collectionId', name: 'collection', component: () => import('@/views/CollectionView.vue') },
    {
      path: '/libraries/:libraryId',
      name: 'library',
      component: () => import('@/views/LibraryView.vue'),
    },
    {
      path: '/libraries/:libraryId/settings',
      name: 'library-settings',
      component: () => import('@/views/LibrarySettingsView.vue'),
      meta: { requiresAdmin: true },
    },
    {
      path: '/videos/:mediaId',
      name: 'video',
      component: () => import('@/views/VideoView.vue'),
    },
    {
      path: '/users',
      name: 'users',
      component: () => import('@/views/UsersView.vue'),
      meta: { requiresAdmin: true },
    },
    { path: '/account', name: 'account', component: () => import('@/views/AccountView.vue') },
  ],
})

router.beforeEach(async (to) => {
  if (!to.meta.requiresAdmin) return true
  try {
    const status = await getAuthStatus()
    return status.authenticated && status.user?.role === 'admin' ? true : { name: 'home' }
  } catch {
    return { name: 'home' }
  }
})
