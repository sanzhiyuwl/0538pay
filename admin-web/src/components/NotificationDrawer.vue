<script setup lang="ts">
import { computed } from 'vue'
import {
  X,
  Info,
  CheckCheck,
} from 'lucide-vue-next'
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { fetchAdminMessages, type AdminMessage } from '@/lib/api/messages'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

const router = useRouter()
const list = ref<AdminMessage[]>([])
const loading = ref(false)
const unreadCount = computed(() => list.value.filter((n) => !n.is_read).length)

// 打开抽屉时拉最近站内信（后台 /messages，我方新增站内信下发的下发列表）。
async function load() {
  loading.value = true
  try {
    const res = await fetchAdminMessages({ page: 1, pageSize: 15 })
    list.value = res.list
  } catch {
    list.value = []
  } finally {
    loading.value = false
  }
}
watch(
  () => props.open,
  (v) => {
    if (v) load()
  },
)

// 站内信无分类字段，统一用 system 图标。保留 tint 以备将来按类型上色。
const tint = 'bg-muted text-muted-foreground'

// 「全部已读」：本地标记（站内信已读态是「谁读过」维度，后台下发侧无逐条已读接口，
// 此处仅前端消未读红点，避免误导有落库能力）。
function markAll() {
  list.value = list.value.map((n) => ({ ...n, is_read: true }))
}
function read(n: AdminMessage) {
  n.is_read = true
}
function viewAll() {
  emit('close')
  router.push('/admin/messages')
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
    <div
      v-if="open"
      class="fixed inset-0 z-50 bg-black/30"
      @click="emit('close')"
    />
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
          v-if="unreadCount"
          class="rounded-full bg-primary/12 px-2 py-0.5 text-xs font-medium text-primary"
          >{{ unreadCount }} 条未读</span
        >
        <div class="flex-1" />
        <button
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
        <div v-if="loading" class="py-16 text-center text-sm text-muted-foreground">加载中…</div>
        <div v-else-if="!list.length" class="py-16 text-center text-sm text-muted-foreground">暂无站内信</div>
        <button
          v-for="n in list"
          :key="n.id"
          class="flex w-full gap-3 border-b border-border/60 px-5 py-4 text-left transition-colors hover:bg-accent/50"
          @click="read(n)"
        >
          <div :class="['flex size-9 shrink-0 items-center justify-center rounded-lg', tint]">
            <Info class="size-[18px]" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium">{{ n.title }}</span>
              <span v-if="!n.is_read" class="size-1.5 shrink-0 rounded-full bg-destructive" />
              <div class="flex-1" />
              <span class="shrink-0 text-xs text-muted-foreground">{{ n.date }}</span>
            </div>
            <p class="mt-1 line-clamp-2 text-[13px] leading-relaxed text-muted-foreground">{{ n.content }}</p>
          </div>
        </button>
      </div>

      <!-- 底 -->
      <div class="border-t border-border px-5 py-3 text-center">
        <button class="text-sm text-muted-foreground transition-colors hover:text-primary" @click="viewAll">
          查看全部消息
        </button>
      </div>
    </aside>
  </transition>
</template>
