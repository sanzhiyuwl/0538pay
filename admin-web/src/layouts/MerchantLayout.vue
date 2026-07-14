<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Menu, Bell, Sun, Moon, ChevronDown, Store } from 'lucide-vue-next'
import { merchantNav, merchantLeaves, type NavNode } from '@/config/nav'
import { useThemeStore } from '@/stores/theme'
import { cn } from '@/lib/utils'
import MerchantUserMenu from '@/components/MerchantUserMenu.vue'

const theme = useThemeStore()
const route = useRoute()
const mobileOpen = ref(false)

// 当前路由属于哪个一级菜单
function nodeActive(node: NavNode) {
  if (node.to) return route.path === node.to
  return node.children?.some((c) => c.to === route.path) ?? false
}

// 展开状态：默认展开当前所在的一级菜单
const openKeys = ref<Set<string>>(new Set())
function syncOpen() {
  merchantNav.forEach((n) => {
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

const currentTitle = computed(
  () => merchantLeaves.find((i) => i.to === route.path)?.title ?? '工作台',
)
const currentParent = computed(() => {
  const p = merchantNav.find((n) => n.children && nodeActive(n))
  return p?.title ?? ''
})
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-content">
    <!-- ===== 侧栏 ===== -->
    <aside
      :class="
        cn(
          'z-40 flex w-[11.25rem] shrink-0 flex-col border-r border-sidebar-border bg-sidebar transition-transform duration-300',
          'max-lg:fixed max-lg:h-full',
          mobileOpen ? 'max-lg:translate-x-0' : 'max-lg:-translate-x-full',
        )
      "
    >
      <!-- 品牌 -->
      <div class="flex h-16 items-center gap-2.5 px-5">
        <div class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <Store class="size-[18px]" />
        </div>
        <div class="leading-tight">
          <div class="text-[15px] font-bold tracking-tight">商户中心</div>
          <div class="text-[11px] text-muted-foreground">0538<span class="text-primary">Pay</span> 商户端</div>
        </div>
      </div>

      <!-- 菜单（两级折叠） -->
      <nav class="flex-1 overflow-y-auto px-3 py-3">
        <ul class="space-y-1">
          <li v-for="node in merchantNav" :key="node.title">
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
                <ul
                  v-show="openKeys.has(node.title)"
                  class="mt-0.5 space-y-0.5 overflow-hidden pl-3.5"
                >
                  <li v-for="leaf in node.children" :key="leaf.to">
                    <RouterLink
                      :to="leaf.to"
                      class="flex items-center gap-2 rounded-lg py-2 pl-3 pr-2 text-[13px] text-sidebar-foreground/80 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
                      active-class="!bg-sidebar-accent !text-sidebar-accent-foreground font-medium"
                      exact-active-class="!bg-sidebar-accent !text-sidebar-accent-foreground font-medium"
                    >
                      <span class="flex-1 truncate">{{ leaf.title }}</span>
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
        商户自助端，管理你的收款、结算与对接。
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
          <span v-if="currentParent" class="text-muted-foreground">{{ currentParent }}</span>
          <span v-if="currentParent" class="text-muted-foreground/50">/</span>
          <span class="font-medium">{{ currentTitle }}</span>
        </div>

        <div class="flex-1" />

        <!-- 通知 -->
        <button class="relative flex size-9 items-center justify-center rounded-lg text-muted-foreground hover:bg-accent">
          <Bell class="size-[18px]" />
          <span class="absolute right-2 top-2 size-1.5 rounded-full bg-destructive" />
        </button>

        <!-- 主题切换 -->
        <button
          class="flex size-9 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          @click="theme.toggle()"
        >
          <Moon v-if="!theme.isDark" class="size-[18px]" />
          <Sun v-else class="size-[18px]" />
        </button>

        <!-- 商户用户菜单 -->
        <MerchantUserMenu class="pl-1" />
      </header>

      <!-- 内容 -->
      <main class="flex-1 overflow-y-auto p-2.5">
        <RouterView />
      </main>
    </div>
  </div>
</template>
