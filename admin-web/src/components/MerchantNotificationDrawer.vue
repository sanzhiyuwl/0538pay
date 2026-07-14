<script setup lang="ts">
import { ref, computed } from 'vue'
import { X, ReceiptText, Wallet, UserCog, Info, CheckCheck } from 'lucide-vue-next'
import { merchantNotices as source, type MerchantNotice } from '@/lib/mock/merchant/notifications'

defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

const list = ref<MerchantNotice[]>([...source])
const unreadCount = computed(() => list.value.filter((n) => n.unread).length)

const iconMap = {
  order: ReceiptText,
  settle: Wallet,
  account: UserCog,
  system: Info,
} as const
const tintMap = {
  order: 'bg-primary/10 text-primary',
  settle: 'bg-success/12 text-success',
  account: 'bg-warning/12 text-warning',
  system: 'bg-muted text-muted-foreground',
} as const

function markAll() {
  list.value = list.value.map((n) => ({ ...n, unread: false }))
}
function read(n: MerchantNotice) {
  n.unread = false
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
        <button
          v-for="n in list"
          :key="n.id"
          class="flex w-full gap-3 border-b border-border/60 px-5 py-4 text-left transition-colors hover:bg-accent/50"
          @click="read(n)"
        >
          <div :class="['flex size-9 shrink-0 items-center justify-center rounded-lg', tintMap[n.type]]">
            <component :is="iconMap[n.type]" class="size-[18px]" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium">{{ n.title }}</span>
              <span v-if="n.unread" class="size-1.5 shrink-0 rounded-full bg-destructive" />
              <div class="flex-1" />
              <span class="shrink-0 text-xs text-muted-foreground">{{ n.time }}</span>
            </div>
            <p class="mt-1 text-[13px] leading-relaxed text-muted-foreground">{{ n.desc }}</p>
          </div>
        </button>
      </div>

      <!-- 底 -->
      <div class="border-t border-border px-5 py-3 text-center">
        <button class="text-sm text-muted-foreground transition-colors hover:text-primary">
          查看全部消息
        </button>
      </div>
    </aside>
  </transition>
</template>
