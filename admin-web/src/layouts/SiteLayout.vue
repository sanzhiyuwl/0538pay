<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { RouterLink, RouterView, useRouter } from 'vue-router'
import { Zap, Menu, X, Sun, Moon } from 'lucide-vue-next'
import { useThemeStore } from '@/stores/theme'
import { Button } from '@/components/ui'

const theme = useThemeStore()
const router = useRouter()

// 吸顶导航：滚动后加背景/阴影
const scrolled = ref(false)
function onScroll() {
  scrolled.value = window.scrollY > 8
}
onMounted(() => window.addEventListener('scroll', onScroll, { passive: true }))
onUnmounted(() => window.removeEventListener('scroll', onScroll))

const mobileOpen = ref(false)

// 导航项：锚点跳首页板块 或 独立页
const navLinks = [
  { label: '首页', to: '/' },
  { label: '产品特性', to: '/#features' },
  { label: '产品矩阵', to: '/#products' },
  { label: '费率方案', to: '/#pricing' },
  { label: '开发文档', to: '/docs' },
  { label: '关于我们', to: '/about' },
]

const footerCols = [
  { title: '产品', links: [{ label: '聚合支付', to: '/#features' }, { label: '费率方案', to: '/#pricing' }, { label: '商户中心', to: '/m/login' }, { label: '控制台', to: '/console' }] },
  { title: '开发者', links: [{ label: '接入文档', to: '/docs' }, { label: '签名规则', to: '/docs' }, { label: '错误码', to: '/docs' }] },
  { title: '关于', links: [{ label: '关于我们', to: '/about' }, { label: '服务协议', to: '/agreement' }, { label: '联系客服', to: '/about' }] },
]

function goTo(to: string) {
  mobileOpen.value = false
  router.push(to)
}
const year = 2026
</script>

<template>
  <div class="flex min-h-screen flex-col bg-background">
    <!-- 吸顶导航 -->
    <header
      class="sticky top-0 z-40 transition-all"
      :class="scrolled ? 'border-b border-border bg-background/90 backdrop-blur' : 'bg-transparent'"
    >
      <div class="mx-auto flex h-16 max-w-7xl items-center gap-6 px-4 lg:px-6">
        <!-- Logo -->
        <RouterLink to="/" class="flex items-center gap-2">
          <div class="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Zap class="size-[18px]" />
          </div>
          <span class="text-lg font-bold tracking-tight">0538<span class="text-primary">Pay</span></span>
        </RouterLink>

        <!-- 桌面导航 -->
        <nav class="hidden items-center gap-1 lg:flex">
          <RouterLink
            v-for="l in navLinks"
            :key="l.to"
            :to="l.to"
            class="rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            {{ l.label }}
          </RouterLink>
        </nav>

        <div class="flex-1" />

        <!-- 主题 + CTA -->
        <button
          class="flex size-9 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          @click="theme.toggle()"
        >
          <Moon v-if="!theme.isDark" class="size-[18px]" />
          <Sun v-else class="size-[18px]" />
        </button>
        <div class="hidden items-center gap-2 sm:flex">
          <Button variant="outline" size="sm" @click="goTo('/m/login')">登录</Button>
          <Button size="sm" @click="goTo('/m/reg')">免费注册</Button>
        </div>

        <!-- 移动菜单按钮 -->
        <button class="flex size-9 items-center justify-center rounded-lg text-muted-foreground hover:bg-accent lg:hidden" @click="mobileOpen = !mobileOpen">
          <component :is="mobileOpen ? X : Menu" class="size-5" />
        </button>
      </div>

      <!-- 移动菜单 -->
      <transition
        enter-active-class="transition duration-150 ease-out"
        leave-active-class="transition duration-100 ease-in"
        enter-from-class="opacity-0 -translate-y-2"
        leave-to-class="opacity-0 -translate-y-2"
      >
        <div v-if="mobileOpen" class="border-t border-border bg-background px-4 py-3 lg:hidden">
          <nav class="flex flex-col gap-1">
            <button v-for="l in navLinks" :key="l.to" class="rounded-md px-3 py-2 text-left text-sm text-muted-foreground hover:bg-accent hover:text-foreground" @click="goTo(l.to)">
              {{ l.label }}
            </button>
            <div class="mt-2 flex gap-2 border-t border-border pt-3">
              <Button variant="outline" size="sm" class="flex-1" @click="goTo('/m/login')">登录</Button>
              <Button size="sm" class="flex-1" @click="goTo('/m/reg')">免费注册</Button>
            </div>
          </nav>
        </div>
      </transition>
    </header>

    <!-- 内容 -->
    <main class="flex-1">
      <RouterView />
    </main>

    <!-- 大页脚 -->
    <footer class="border-t border-border bg-content">
      <div class="mx-auto max-w-7xl px-4 py-12 lg:px-6">
        <div class="grid grid-cols-2 gap-8 md:grid-cols-4">
          <!-- 品牌列 -->
          <div class="col-span-2 md:col-span-1">
            <div class="flex items-center gap-2">
              <div class="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <Zap class="size-[18px]" />
              </div>
              <span class="text-lg font-bold tracking-tight">0538<span class="text-primary">Pay</span></span>
            </div>
            <p class="mt-3 text-sm leading-relaxed text-muted-foreground">
              专业的聚合支付服务平台，支持多渠道收款、实时到账、开放 API 对接。
            </p>
          </div>
          <!-- 链接列 -->
          <div v-for="col in footerCols" :key="col.title">
            <div class="text-sm font-semibold">{{ col.title }}</div>
            <ul class="mt-3 space-y-2">
              <li v-for="l in col.links" :key="l.label">
                <RouterLink :to="l.to" class="text-sm text-muted-foreground transition-colors hover:text-primary">{{ l.label }}</RouterLink>
              </li>
            </ul>
          </div>
        </div>

        <div class="mt-10 flex flex-col items-center justify-between gap-2 border-t border-border pt-6 text-xs text-muted-foreground sm:flex-row">
          <span>© 2016-{{ year }} 0538Pay 版权所有 · 鲁ICP备2026000538号-1</span>
          <span>鲁公网安备 37098202000538号</span>
        </div>
      </div>
    </footer>
  </div>
</template>
