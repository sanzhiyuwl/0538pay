<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { X, Mail, MailOpen, Megaphone, CheckCheck } from 'lucide-vue-next'
import { fetchMessages, readMessage, type MerchantMessage } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

const toast = useToast()

const list = ref<MerchantMessage[]>([])
const total = ref(0)
const unread = ref(0)
const loading = ref(false)
const page = ref(1)
const pageSize = 20
const expanded = ref<number | null>(null)

const hasMore = computed(() => list.value.length < total.value)

async function load(reset = true) {
  if (reset) {
    page.value = 1
    expanded.value = null
  }
  loading.value = true
  try {
    const res = await fetchMessages({ page: page.value, pageSize })
    const rows = res.list || []
    list.value = reset ? rows : [...list.value, ...rows]
    total.value = res.total || 0
    unread.value = res.unread || 0
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载站内信失败')
  } finally {
    loading.value = false
  }
}

// 打开时加载最新数据（每次开都刷新，保证未读实时）
watch(
  () => props.open,
  (v) => {
    if (v) load(true)
  },
)

function loadMore() {
  if (loading.value || !hasMore.value) return
  page.value += 1
  load(false)
}

async function toggle(m: MerchantMessage) {
  if (expanded.value === m.id) {
    expanded.value = null
    return
  }
  expanded.value = m.id
  if (!m.is_read) {
    try {
      await readMessage(m.id)
      m.is_read = true
      unread.value = Math.max(0, unread.value - 1)
    } catch {
      /* 已读失败不打扰 */
    }
  }
}

async function markAll() {
  const unreadRows = list.value.filter((n) => !n.is_read)
  if (!unreadRows.length) return
  try {
    await Promise.all(unreadRows.map((n) => readMessage(n.id)))
    unreadRows.forEach((n) => (n.is_read = true))
    unread.value = 0
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
}
</script>

<template>
  <!-- 遮罩 -->
  <transition
    enter-active-class="transition-opacity duration-200"
    leave-active-class="transition-opacity duration-200"
    enter-from-class="opacity-0"
    leave-to-class="opacity-0"
  >
    <div v-if="open" class="fixed inset-0 z-50 bg-black/30" @click="emit('close')" />
  </transition>

  <!-- 右侧抽屉 -->
  <transition
    enter-active-class="transition-transform duration-300 ease-out"
    leave-active-class="transition-transform duration-200 ease-in"
    enter-from-class="translate-x-full"
    leave-to-class="translate-x-full"
  >
    <aside
      v-if="open"
      class="fixed right-0 top-0 z-50 flex h-full w-full max-w-md flex-col bg-background shadow-2xl"
    >
      <!-- 头 -->
      <div class="flex items-center gap-2 border-b border-border px-5 py-4">
        <h3 class="text-[15px] font-semibold">站内信</h3>
        <span
          v-if="unread"
          class="rounded-full bg-primary/12 px-2 py-0.5 text-xs font-medium text-primary"
          >{{ unread }} 条未读</span
        >
        <div class="flex-1" />
        <button
          v-if="unread"
          class="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-primary"
          @click="markAll"
        >
          <CheckCheck class="size-3.5" /> 全部已读
        </button>
        <button
          class="flex size-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-accent"
          @click="emit('close')"
        >
          <X class="size-[18px]" />
        </button>
      </div>

      <!-- 列表 -->
      <div class="flex-1 overflow-y-auto">
        <button
          v-for="n in list"
          :key="n.id"
          class="flex w-full gap-3 border-b border-border/60 px-5 py-4 text-left transition-colors hover:bg-accent/50"
          @click="toggle(n)"
        >
          <div class="flex size-9 shrink-0 items-center justify-center rounded-lg" :class="!n.is_read ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground'">
            <component :is="n.uid === 0 ? Megaphone : n.is_read ? MailOpen : Mail" class="size-[18px]" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="text-sm" :class="!n.is_read && 'font-medium'">{{ n.title }}</span>
              <span v-if="!n.is_read" class="size-1.5 shrink-0 rounded-full bg-destructive" />
              <div class="flex-1" />
              <span class="shrink-0 text-xs text-muted-foreground">{{ n.date }}</span>
            </div>
            <p
              class="mt-1 text-[13px] leading-relaxed text-muted-foreground"
              :class="expanded === n.id ? 'whitespace-pre-wrap' : 'line-clamp-2'"
            >
              {{ n.content }}
            </p>
          </div>
        </button>

        <!-- 加载更多 / 空态 -->
        <div v-if="hasMore" class="px-5 py-3 text-center">
          <button class="text-sm text-muted-foreground transition-colors hover:text-primary" :disabled="loading" @click="loadMore">
            {{ loading ? '加载中…' : '加载更多' }}
          </button>
        </div>
        <div v-else-if="!loading && !list.length" class="py-16 text-center text-sm text-muted-foreground">
          暂无站内信
        </div>
      </div>
    </aside>
  </transition>
</template>
