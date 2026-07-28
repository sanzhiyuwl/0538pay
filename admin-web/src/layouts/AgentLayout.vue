<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter, RouterLink, RouterView } from 'vue-router'
import { Menu, LogOut, Sun, Moon, Handshake } from 'lucide-vue-next'
import { agentNav } from '@/config/nav'
import { useThemeStore } from '@/stores/theme'
import { useAgentAuthStore } from '@/stores/agentAuth'
import { cn } from '@/lib/utils'

const theme = useThemeStore()
const route = useRoute()
const router = useRouter()
const agentAuth = useAgentAuthStore()
const mobileOpen = ref(false)

// 进入时刷新一次代理资料，保证权限（可能被平台调整过）与菜单准确。
onMounted(() => {
  agentAuth.refresh().catch(() => {
    // 刷新失败（token 失效等）交给 request 的 401 处理跳登录
  })
})

// 按权限过滤菜单：概览（无 perm）恒显示，其余按 permissions 门控。
const visibleNav = computed(() => agentNav.filter((n) => !n.perm || agentAuth.has(n.perm)))
const currentTitle = computed(() => agentNav.find((i) => i.to === route.path)?.title ?? '工作台')

function logout() {
  agentAuth.logout()
  router.push('/agent/login')
}
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-content">
    <!-- ===== 侧栏 ===== -->
    <aside
      :class="
        cn(
          'z-40 flex w-[11.75rem] shrink-0 flex-col border-r border-sidebar-border bg-sidebar transition-transform duration-300',
          'max-lg:fixed max-lg:h-full',
          mobileOpen ? 'max-lg:translate-x-0' : 'max-lg:-translate-x-full',
        )
      "
    >
      <!-- 品牌 -->
      <div class="flex h-16 items-center gap-2.5 px-5">
        <div class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <Handshake class="size-[18px]" />
        </div>
        <div class="leading-tight">
          <div class="text-[15px] font-bold tracking-tight">代理工作台</div>
          <div class="text-[11px] text-muted-foreground">进件 · 名额 · 结算</div>
        </div>
      </div>

      <div class="px-3 pb-1">
        <div class="border-t border-sidebar-border" />
      </div>

      <!-- 菜单（一级平铺，按权限过滤） -->
      <nav class="flex-1 overflow-y-auto px-3 py-2">
        <ul class="space-y-1">
          <li v-for="node in visibleNav" :key="node.title">
            <RouterLink
              :to="node.to!"
              class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-sidebar-foreground transition-colors hover:bg-sidebar-accent"
              active-class="!bg-sidebar-accent !text-sidebar-accent-foreground font-semibold"
              exact-active-class="!bg-sidebar-accent !text-sidebar-accent-foreground font-semibold"
            >
              <component :is="node.icon" class="size-[18px] shrink-0" />
              <span class="flex-1">{{ node.title }}</span>
            </RouterLink>
          </li>
        </ul>
      </nav>

      <!-- 底部说明 -->
      <div class="border-t border-sidebar-border px-4 py-3 text-[11px] leading-relaxed text-muted-foreground">
        代理端仅展示你名下数据，功能按平台开通的权限显示。
      </div>
    </aside>

    <!-- 移动端遮罩 -->
    <div v-if="mobileOpen" class="fixed inset-0 z-30 bg-black/40 lg:hidden" @click="mobileOpen = false" />

    <!-- ===== 主区 ===== -->
    <div class="flex min-w-0 flex-1 flex-col">
      <!-- 顶栏 -->
      <header class="flex h-16 shrink-0 items-center gap-3 border-b border-border bg-background px-4 lg:px-6">
        <button
          class="flex size-9 items-center justify-center rounded-lg text-muted-foreground hover:bg-accent lg:hidden"
          @click="mobileOpen = true"
        >
          <Menu class="size-5" />
        </button>

        <!-- 面包屑 -->
        <div class="flex items-center gap-1.5 text-sm">
          <span class="text-muted-foreground">代理工作台</span>
          <span class="text-muted-foreground/50">/</span>
          <span class="font-medium">{{ currentTitle }}</span>
        </div>

        <div class="flex-1" />

        <!-- 当前代理 -->
        <span class="text-sm text-muted-foreground">{{ agentAuth.name || '代理' }}</span>

        <!-- 主题切换 -->
        <button
          class="flex size-9 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          @click="theme.toggle()"
        >
          <Moon v-if="!theme.isDark" class="size-[18px]" />
          <Sun v-else class="size-[18px]" />
        </button>

        <!-- 退出 -->
        <button
          class="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:border-destructive/40 hover:bg-destructive/[0.06] hover:text-destructive"
          @click="logout"
        >
          <LogOut class="size-4 shrink-0" />
          <span>退出</span>
        </button>
      </header>

      <!-- 内容 -->
      <main class="flex-1 overflow-y-auto p-2.5">
        <RouterView />
      </main>
    </div>
  </div>
</template>
