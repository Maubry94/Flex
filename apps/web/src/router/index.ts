import { createRouter, createWebHistory } from 'vue-router'

import LibrariesView from '@/views/LibrariesView.vue'
import LibraryView from '@/views/LibraryView.vue'
import VideoView from '@/views/VideoView.vue'
import HomeView from '@/views/HomeView.vue'
import LibrarySettingsView from '@/views/LibrarySettingsView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView,
    },
    {
      path: '/libraries',
      name: 'libraries',
      component: LibrariesView,
    },
    {
      path: '/libraries/:libraryId',
      name: 'library',
      component: LibraryView,
    },
    {
      path: '/libraries/:libraryId/settings',
      name: 'library-settings',
      component: LibrarySettingsView,
    },
    {
      path: '/videos/:mediaId',
      name: 'video',
      component: VideoView,
    },
  ],
})
