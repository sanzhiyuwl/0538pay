<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { RouterLink, RouterView, useRouter, useRoute } from 'vue-router'
import { Zap, Menu, X, Sun, Moon, Info } from 'lucide-vue-next'
import { useThemeStore } from '@/stores/theme'
import { Button } from '@/components/ui'
import { useSiteStore } from '@/stores/site'

const theme = useThemeStore()
// 站点配置来自后台「网站设置」，实时联动（config 为 reactive，模板直接读即响应）
const site = useSiteStore().config
const router = useRouter()
const route = useRoute()

// 吸顶导航：滚动后加背景/阴影
const scrolled = ref(false)
// 首页当前所在锚点板块（用于导航高亮）
const activeSection = ref('')
const sectionIds = ['features', 'pricing', 'products']
function onScroll() {
  scrolled.value = window.scrollY > 8
  // 仅在首页时检测板块：判定线穿过某板块（top ≤ 线 且 bottom > 线）才算激活
  if (route.path === '/') {
    const line = 140
    let cur = ''
    for (const id of sectionIds) {
      const el = document.getElementById(id)
      if (!el) continue
      const r = el.getBoundingClientRect()
      if (r.top <= line && r.bottom > line) { cur = id; break }
    }
    activeSection.value = cur
  }
}
onMounted(() => window.addEventListener('scroll', onScroll, { passive: true }))
onUnmounted(() => window.removeEventListener('scroll', onScroll))

// 判断导航项是否激活
function isActive(to: string): boolean {
  if (to.startsWith('/#')) {
    return route.path === '/' && activeSection.value === to.slice(2)
  }
  if (to === '/') {
    return route.path === '/' && activeSection.value === ''
  }
  return route.path === to || route.path.startsWith(to + '/')
}

const mobileOpen = ref(false)

// 合规声明告示条显隐（右侧关闭按钮控制）
const showDisclaimer = ref(true)

// 导航项：锚点跳首页板块 或 独立页
const navLinks = [
  { label: '首页', to: '/' },
  { label: '产品特性', to: '/#features' },
  { label: '费率方案', to: '/#pricing' },
  { label: '产品矩阵', to: '/#products' },
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

// 导航点击：锚点平滑滚动、首页回顶、独立页跳转
async function navClick(to: string) {
  mobileOpen.value = false
  if (to === '/') {
    if (route.path !== '/') await router.push('/')
    activeSection.value = ''
    window.scrollTo({ top: 0, behavior: 'smooth' })
    return
  }
  if (to.startsWith('/#')) {
    const id = to.slice(2)
    if (route.path !== '/') {
      await router.push('/')
      // 等待首页渲染后再滚动
      requestAnimationFrame(() => document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' }))
    } else {
      document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
    }
    return
  }
  router.push(to)
}
</script>

<template>
  <div class="relative flex min-h-screen flex-col bg-background">
    <!-- 合规声明告示条（页面最顶部，导航上方，撑满宽度纯色，可关闭）-->
    <div
      v-if="site.disclaimer && showDisclaimer"
      class="flex items-center gap-2.5 border-b border-warning/30 bg-[#fdf6e3] px-4 py-2.5 text-sm text-amber-700 dark:bg-[#3a2e10] dark:text-amber-300 lg:px-6"
    >
      <Info class="size-4 shrink-0 text-warning" />
      <span class="leading-relaxed">{{ site.disclaimer }}</span>
      <button
        class="ml-auto flex size-7 shrink-0 items-center justify-center rounded-full text-amber-700/70 transition-colors hover:bg-warning/20 hover:text-amber-700 dark:text-amber-300/70 dark:hover:text-amber-300"
        aria-label="关闭提示"
        @click="showDisclaimer = false"
      >
        <X class="size-4" />
      </button>
    </div>

    <!-- 吸顶导航 -->
    <header
      class="sticky top-0 z-40 border-b transition-colors duration-300"
      :class="scrolled ? 'border-border bg-background shadow-sm' : 'border-transparent bg-background/30 backdrop-blur-xl'"
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
          <button
            v-for="l in navLinks"
            :key="l.to"
            type="button"
            class="group relative px-4 py-5 text-[15px] font-medium transition-colors"
            :class="isActive(l.to) ? 'text-primary' : 'text-foreground/80 hover:text-foreground'"
            @click="navClick(l.to)"
          >
            {{ l.label }}
            <!-- 选中态蓝色横杠（居中短横）-->
            <span
              class="absolute bottom-0 left-1/2 h-[3px] w-6 -translate-x-1/2 rounded-full bg-primary transition-all duration-300"
              :class="isActive(l.to) ? 'opacity-100' : 'opacity-0 group-hover:opacity-40'"
            />
          </button>
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
            <button
              v-for="l in navLinks"
              :key="l.to"
              class="rounded-md px-3 py-2 text-left text-sm transition-colors hover:bg-accent"
              :class="isActive(l.to) ? 'font-medium text-primary' : 'text-muted-foreground hover:text-foreground'"
              @click="navClick(l.to)"
            >
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
    <main class="flex-1 bg-content">
      <RouterView />
    </main>

    <!-- 大页脚 -->
    <footer class="relative bg-content">
      <!-- 顶部柔和渐变分隔线（替代生硬 border） -->
      <div class="h-px w-full bg-gradient-to-r from-transparent via-border to-transparent" />
      <div class="mx-auto max-w-7xl px-4 py-14 lg:px-6">
        <div class="grid grid-cols-2 gap-8 md:grid-cols-4 md:gap-12">
          <!-- 品牌列（右侧竖直渐变分隔线，仅桌面显示）-->
          <div class="relative col-span-2 md:col-span-1 md:pr-12 md:after:absolute md:after:right-0 md:after:top-1 md:after:h-[85%] md:after:w-px md:after:bg-gradient-to-b md:after:from-transparent md:after:via-border md:after:to-transparent">
            <div class="flex items-center gap-2">
              <div class="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <Zap class="size-[18px]" />
              </div>
              <span class="text-lg font-bold tracking-tight">0538<span class="text-primary">Pay</span></span>
            </div>
            <p class="mt-3 text-sm leading-relaxed text-muted-foreground">
              专业的聚合支付服务平台，支持多渠道收款、实时到账、开放 API 对接。
            </p>
            <div class="mt-4 text-xs text-muted-foreground">
              客服 QQ：{{ site.qq }} · {{ site.email }}
            </div>
          </div>
          <!-- 链接列 -->
          <div v-for="col in footerCols" :key="col.title">
            <div class="text-sm font-semibold">{{ col.title }}</div>
            <ul class="mt-4 space-y-2.5">
              <li v-for="l in col.links" :key="l.label">
                <RouterLink :to="l.to" class="text-sm text-muted-foreground transition-colors hover:text-primary">{{ l.label }}</RouterLink>
              </li>
            </ul>
          </div>
        </div>

        <!-- 合规声明（纯文字）-->
        <p v-if="site.disclaimer" class="mt-10 border-t border-border/50 pt-6 text-xs leading-relaxed text-muted-foreground">
          {{ site.disclaimer }}
        </p>

        <!-- 底部版权 + 备案（读取后台站点配置）-->
        <div class="mt-6 flex flex-col items-center justify-between gap-3 text-xs text-muted-foreground sm:flex-row">
          <span>{{ site.copyright }} · {{ site.company }}</span>
          <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
            <a :href="site.copyrightLink" target="_blank" rel="noopener" class="transition-colors hover:text-primary">{{ site.icp }}</a>
            <a :href="site.policeLink" target="_blank" rel="noopener" class="flex items-center gap-1 transition-colors hover:text-primary">{{ site.police }}</a>
          </div>
        </div>
      </div>
    </footer>
  </div>
</template>
