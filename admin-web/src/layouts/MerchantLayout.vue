<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Menu, Bell, Sun, Moon, ChevronDown, Store, ShieldAlert } from 'lucide-vue-next'
import { merchantNav as rawMerchantNav, type NavNode, type NavLeaf } from '@/config/nav'
import { useThemeStore } from '@/stores/theme'
import { cn } from '@/lib/utils'
import { useIdleLogout } from '@/composables/useIdleLogout'
import { useMerchantAuthStore } from '@/stores/merchantAuth'
import { useMerchantFeaturesStore } from '@/stores/merchantFeatures'
import MerchantUserMenu from '@/components/MerchantUserMenu.vue'
import MerchantNotificationDrawer from '@/components/MerchantNotificationDrawer.vue'
import { fetchMessages } from '@/lib/api/merchantCenter'
import { Button } from '@/components/ui'

const theme = useThemeStore()

// 商户端全局功能开关：挂载拉一次，用于过滤导航（保证金门槛关闭时隐藏保证金入口）。
const featuresStore = useMerchantFeaturesStore()
onMounted(() => featuresStore.load())

// 按开关过滤后的导航：保证金门槛（deposit）关闭时，从「账户中心」剔除「保证金」子项。
const merchantNav = computed<NavNode[]>(() =>
  rawMerchantNav
    .map((node) => {
      if (!node.children) return node
      const children = node.children.filter(
        (leaf) => leaf.to !== '/m/deposit' || featuresStore.features.deposit,
      )
      return { ...node, children }
    })
    // 剔除过滤后子项为空的分组（当前不会出现，防御性保留）
    .filter((node) => !node.children || node.children.length > 0),
)
// 可路由叶子（面包屑/当前标题用），随过滤后的导航同步。
const merchantLeaves = computed<NavLeaf[]>(() =>
  merchantNav.value.flatMap((n) =>
    n.children ? n.children : n.to ? [{ title: n.title, to: n.to }] : [],
  ),
)

// 闲置超时自动退出（安全增强）：与管理后台同源逻辑，跳商户登录页。
const merchantAuth = useMerchantAuthStore()
const idle = useIdleLogout({ logout: () => merchantAuth.logout(), loginRouteName: 'm-login' })
const route = useRoute()
const mobileOpen = ref(false)
const noticeOpen = ref(false)

// 站内信未读数（真接口）：挂载拉一次，抽屉关闭后同步刷新红点。
const unreadCount = ref(0)
const hasUnread = computed(() => unreadCount.value > 0)
async function refreshUnread() {
  try {
    const res = await fetchMessages({ page: 1, pageSize: 1 })
    unreadCount.value = res.unread || 0
  } catch {
    /* 未读数拉取失败不打扰 */
  }
}
onMounted(refreshUnread)
function onNoticeClose() {
  noticeOpen.value = false
  refreshUnread()
}

// 当前路由属于哪个一级菜单
function nodeActive(node: NavNode) {
  if (node.to) return route.path === node.to
  return node.children?.some((c) => c.to === route.path) ?? false
}

// 展开状态：默认展开当前所在的一级菜单
const openKeys = ref<Set<string>>(new Set())
function syncOpen() {
  merchantNav.value.forEach((n) => {
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
  () => merchantLeaves.value.find((i) => i.to === route.path)?.title ?? '工作台',
)
const currentParent = computed(() => {
  const p = merchantNav.value.find((n) => n.children && nodeActive(n))
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
        <button
          class="relative flex size-9 items-center justify-center rounded-lg text-muted-foreground hover:bg-accent"
          @click="noticeOpen = true"
        >
          <Bell class="size-[18px]" />
          <span v-if="hasUnread" class="absolute right-2 top-2 size-1.5 rounded-full bg-destructive" />
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

    <!-- 站内信抽屉 -->
    <MerchantNotificationDrawer :open="noticeOpen" @close="onNoticeClose" />

    <!-- 闲置超时警告：到期前 60 秒弹倒计时，不可点背景关闭，需显式选择 -->
    <transition
      enter-active-class="transition-opacity duration-200"
      leave-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div
        v-if="idle.warning.value"
        class="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 p-4"
      >
        <div class="flex w-full max-w-sm flex-col bg-background shadow-2xl">
          <div class="flex flex-col items-center gap-3 px-6 pt-7 text-center">
            <div
              class="flex size-12 items-center justify-center rounded-full bg-warning/[0.12] text-warning"
            >
              <ShieldAlert class="size-6" />
            </div>
            <h3 class="text-base font-semibold">即将自动退出登录</h3>
            <p class="text-sm leading-relaxed text-muted-foreground">
              检测到您已 {{ idle.idleLimitMinutes }} 分钟未操作，为保障账号安全，
              <span class="font-semibold text-warning tabular-nums">{{ idle.remaining.value }}</span>
              秒后将自动退出。
            </p>
          </div>
          <div class="flex items-center justify-center gap-2 px-6 pb-6 pt-5">
            <Button variant="outline" @click="idle.logoutNow()">立即退出</Button>
            <Button @click="idle.stay()">继续操作</Button>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>
