<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute, RouterLink, RouterView } from 'vue-router'
import { Menu, ArrowLeft, Sun, Moon, Server, ChevronDown } from 'lucide-vue-next'
import { consoleNav, consoleLeaves, type NavNode } from '@/config/nav'
import { useThemeStore } from '@/stores/theme'
import { cn } from '@/lib/utils'
import UserMenu from '@/components/UserMenu.vue'

const theme = useThemeStore()
const route = useRoute()
const mobileOpen = ref(false)

const currentTitle = computed(
  () => consoleLeaves.find((i) => i.to === route.path)?.title ?? '概况',
)

// —— 两级折叠：分组是否含当前路由 / 展开态 ——
function nodeActive(node: NavNode): boolean {
  return node.children?.some((c) => c.to === route.path) ?? false
}
const openKeys = ref<Set<string>>(new Set())
function syncOpen() {
  consoleNav.forEach((n) => {
    if (n.children && nodeActive(n)) openKeys.value.add(n.title)
  })
}
syncOpen()
watch(() => route.path, syncOpen)

function toggle(node: NavNode) {
  if (!node.children) return
  if (openKeys.value.has(node.title)) openKeys.value.delete(node.title)
  else openKeys.value.add(node.title)
}
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-content">
    <!-- ===== 侧栏 ===== -->
    <aside
      :class="
        cn(
          'z-40 flex w-[13.5rem] shrink-0 flex-col border-r border-sidebar-border bg-sidebar transition-transform duration-300',
          'max-lg:fixed max-lg:h-full',
          mobileOpen ? 'max-lg:translate-x-0' : 'max-lg:-translate-x-full',
        )
      "
    >
      <!-- 品牌 -->
      <div class="flex h-16 items-center gap-2.5 px-5">
        <div class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <Server class="size-[18px]" />
        </div>
        <div class="leading-tight">
          <div class="text-[15px] font-bold tracking-tight">代理控制台</div>
          <div class="text-[11px] text-muted-foreground">进件代理 · 佣金管理</div>
        </div>
      </div>

      <div class="px-3 pb-1">
        <div class="border-t border-sidebar-border" />
      </div>

      <!-- 菜单（两级折叠） -->
      <nav class="flex-1 overflow-y-auto px-3 py-2">
        <ul class="space-y-1">
          <li v-for="node in consoleNav" :key="node.title">
            <!-- 单项（无子菜单） -->
            <RouterLink
              v-if="node.to"
              :to="node.to"
              class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-sidebar-foreground transition-colors hover:bg-sidebar-accent"
              active-class="!bg-sidebar-accent !text-sidebar-accent-foreground font-semibold"
              exact-active-class="!bg-sidebar-accent !text-sidebar-accent-foreground font-semibold"
            >
              <component :is="node.icon" class="size-[18px] shrink-0" />
              <span class="flex-1">{{ node.title }}</span>
              <span
                v-if="node.badge"
                class="rounded bg-primary/12 px-1.5 py-0.5 text-[10px] font-medium text-primary"
                >{{ node.badge }}</span
              >
            </RouterLink>

            <!-- 有子菜单 -->
            <template v-else>
              <button
                :class="
                  cn(
                    'flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                    nodeActive(node)
                      ? 'text-sidebar-accent-foreground'
                      : 'text-sidebar-foreground hover:bg-sidebar-accent',
                  )
                "
                @click="toggle(node)"
              >
                <component :is="node.icon" class="size-[18px] shrink-0" />
                <span class="flex-1 text-left">{{ node.title }}</span>
                <ChevronDown
                  :class="[
                    'size-4 shrink-0 text-muted-foreground transition-transform',
                    openKeys.has(node.title) && 'rotate-180',
                  ]"
                />
              </button>

              <!-- 子项 -->
              <transition
                enter-active-class="transition-all duration-200 ease-out"
                leave-active-class="transition-all duration-150 ease-in"
                enter-from-class="opacity-0 max-h-0"
                enter-to-class="opacity-100 max-h-96"
                leave-from-class="opacity-100 max-h-96"
                leave-to-class="opacity-0 max-h-0"
              >
                <ul v-show="openKeys.has(node.title)" class="mt-0.5 space-y-0.5 overflow-hidden pl-3.5">
                  <li v-for="leaf in node.children" :key="leaf.to">
                    <RouterLink
                      :to="leaf.to"
                      class="flex items-center gap-2.5 rounded-lg py-2 pl-3 pr-2 text-[13px] text-sidebar-foreground/80 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
                      active-class="!bg-sidebar-accent !text-sidebar-accent-foreground font-medium"
                      exact-active-class="!bg-sidebar-accent !text-sidebar-accent-foreground font-medium"
                    >
                      <component :is="leaf.icon" v-if="leaf.icon" class="size-4 shrink-0" />
                      <span class="flex-1 truncate">{{ leaf.title }}</span>
                      <span
                        v-if="leaf.badge"
                        class="rounded bg-primary/15 px-1.5 py-0.5 text-[10px] font-semibold text-primary"
                        >{{ leaf.badge }}</span
                      >
                    </RouterLink>
                  </li>
                </ul>
              </transition>
            </template>
          </li>
        </ul>
      </nav>

      <!-- 底部说明 -->
      <div class="border-t border-sidebar-border px-4 py-3 text-[11px] leading-relaxed text-muted-foreground">
        控制台为平台方视角，管理进件代理与特约商户进件。
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
          <span class="text-muted-foreground">代理控制台</span>
          <span class="text-muted-foreground/50">/</span>
          <span class="font-medium">{{ currentTitle }}</span>
        </div>

        <div class="flex-1" />

        <!-- 返回客户端 -->
        <RouterLink
          to="/admin"
          class="group flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:border-primary/40 hover:bg-primary/[0.06] hover:text-primary"
        >
          <ArrowLeft class="size-4 shrink-0 transition-transform group-hover:-translate-x-0.5" />
          <span>返回客户端</span>
        </RouterLink>

        <!-- 分隔线 -->
        <div class="mx-1 h-5 w-px bg-border" />

        <!-- 主题切换 -->
        <button
          class="flex size-9 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          @click="theme.toggle()"
        >
          <Moon v-if="!theme.isDark" class="size-[18px]" />
          <Sun v-else class="size-[18px]" />
        </button>

        <!-- 用户菜单 -->
        <UserMenu />
      </header>

      <!-- 内容 -->
      <main class="flex-1 overflow-y-auto p-2.5">
        <RouterView />
      </main>
    </div>
  </div>
</template>
