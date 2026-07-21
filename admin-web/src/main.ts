import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import App from './App.vue'
import router from './router'
import { reveal } from './directives/reveal'
import { onUnauthorized, onMerchantUnauthorized } from '@/lib/api/client'
import { useAuthStore } from '@/stores/auth'
import { useMerchantAuthStore } from '@/stores/merchantAuth'

const app = createApp(App)
app.use(createPinia()).use(router).directive('reveal', reveal)

// 后台 token 失效（401）：清登录态并跳后台登录页，回跳当前路由
onUnauthorized(() => {
  const auth = useAuthStore()
  auth.logout()
  const current = router.currentRoute.value
  if (current.name !== 'login') {
    router.replace({ name: 'login', query: { redirect: current.fullPath } })
  }
})

// 商户端 token 失效（401）：清商户登录态并跳商户登录页
onMerchantUnauthorized(() => {
  const m = useMerchantAuthStore()
  m.logout()
  const current = router.currentRoute.value
  if (current.name !== 'm-login') {
    router.replace({ name: 'm-login', query: { redirect: current.fullPath } })
  }
})

app.mount('#app')
