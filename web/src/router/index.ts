import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import DevicesView from '../views/DevicesView.vue'
import DeviceDetailView from '../views/DeviceDetailView.vue'
import LoginView from '../views/LoginView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView },
    { path: '/', name: 'devices', component: DevicesView },
    { path: '/devices/:id', name: 'device-detail', component: DeviceDetailView },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!auth.authenticated && to.name !== 'login') return { name: 'login' }
  if (auth.authenticated && to.name === 'login') return { name: 'devices' }
})

export default router
