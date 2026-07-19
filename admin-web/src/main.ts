import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import App from './App.vue'
import router from './router'
import { reveal } from './directives/reveal'
import { onUnauthorized } from '@/lib/api/client'
import { useAuthStore } from '@/stores/auth'

const app = createApp(App)
app.use(createPinia()).use(router).directive('reveal', reveal)

// token 失效（后端返回 401）时清登录态并跳登录页，回跳当前路由
onUnauthorized(() => {
  const auth = useAuthStore()
  auth.logout()
  const current = router.currentRoute.value
  if (current.name !== 'login') {
    router.replace({ name: 'login', query: { redirect: current.fullPath } })
  }
})

app.mount('#app')
