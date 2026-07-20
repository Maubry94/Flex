import { createRouter, createWebHistory } from 'vue-router'

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
    },
    {
      path: '/videos/:mediaId',
      name: 'video',
      component: () => import('@/views/VideoView.vue'),
    },
  ],
})
