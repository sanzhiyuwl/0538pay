import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '@/layouts/AdminLayout.vue'
import Dashboard from '@/views/Dashboard.vue'
import Placeholder from '@/views/Placeholder.vue'
import StyleGuide from '@/views/StyleGuide.vue'
import Orders from '@/views/Orders.vue'
import Merchants from '@/views/Merchants.vue'
import { allLeaves } from '@/config/nav'

// 已实现的正式页面（其余菜单项暂用占位页）
const realPages: Record<string, any> = {
  '/orders': Orders,
  '/merchants': Merchants,
}

const placeholderRoutes = allLeaves
  .filter((i) => i.to !== '/')
  .map((i) => ({
    path: i.to,
    name: i.to,
    component: realPages[i.to] ?? Placeholder,
  }))

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: AdminLayout,
      children: [
        { path: '', name: 'dashboard', component: Dashboard },
        { path: 'style-guide', name: 'style-guide', component: StyleGuide },
        ...placeholderRoutes,
      ],
    },
  ],
})

export default router
