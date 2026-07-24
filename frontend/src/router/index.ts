import { createRouter, createWebHistory } from 'vue-router'
import { useSourcesStore } from '@/stores/sources'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'upload', component: () => import('@/screens/UploadScreen.vue') },
    { path: '/params', name: 'params', component: () => import('@/screens/ParamsScreen.vue') },
    {
      path: '/computing',
      name: 'computing',
      component: () => import('@/screens/ComputingScreen.vue'),
    },
    { path: '/results', name: 'results', component: () => import('@/screens/ResultsScreen.vue') },
    {
      path: '/consolidated',
      name: 'consolidated',
      component: () => import('@/screens/ConsolidatedScreen.vue'),
    },
  ],
})

// Гейт: результаты и сводный документ недоступны без загруженных данных.
router.beforeEach((to) => {
  const sources = useSourcesStore()
  if (
    (to.name === 'params' || to.name === 'results' || to.name === 'consolidated') &&
    !sources.loaded
  ) {
    return { name: 'upload' }
  }
  return true
})

export default router
