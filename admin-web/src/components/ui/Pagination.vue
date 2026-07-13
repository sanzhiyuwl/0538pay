<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'

/**
 * 分页组件（页码 + 上一页/下一页 + 智能省略号）。全站列表页统一使用。
 * <Pagination :page="page" :page-count="pageCount" :total="total" @change="go" />
 */
const props = defineProps<{
  page: number
  pageCount: number
  total: number
  pageSize?: number
}>()
const emit = defineEmits<{ change: [p: number] }>()

// 生成页码序列，超过 7 页时用省略号：1 … 4 5 6 … 20
const pages = computed<(number | '...')[]>(() => {
  const n = props.pageCount
  const cur = props.page
  if (n <= 7) return Array.from({ length: n }, (_, i) => i + 1)
  const set = new Set<number>([1, n, cur, cur - 1, cur + 1])
  const sorted = [...set].filter((p) => p >= 1 && p <= n).sort((a, b) => a - b)
  const out: (number | '...')[] = []
  let prev = 0
  for (const p of sorted) {
    if (p - prev > 1) out.push('...')
    out.push(p)
    prev = p
  }
  return out
})

function go(p: number) {
  if (p < 1 || p > props.pageCount || p === props.page) return
  emit('change', p)
}
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-3 text-sm">
    <span class="text-muted-foreground">
      共 {{ total }} 条<template v-if="pageSize">，每页 {{ pageSize }} 条</template>
    </span>
    <div class="flex items-center gap-1">
      <button
        class="flex h-8 min-w-8 items-center justify-center rounded border border-border px-2 text-muted-foreground transition-colors hover:border-primary hover:text-primary disabled:pointer-events-none disabled:opacity-40"
        :disabled="page <= 1"
        @click="go(page - 1)"
      >
        <ChevronLeft class="size-4" />
      </button>

      <template v-for="(p, i) in pages" :key="i">
        <span v-if="p === '...'" class="flex h-8 min-w-8 items-center justify-center text-muted-foreground">…</span>
        <button
          v-else
          class="flex h-8 min-w-8 items-center justify-center rounded border px-2 tabular-nums transition-colors"
          :class="p === page
            ? 'border-primary bg-primary text-primary-foreground'
            : 'border-border text-foreground hover:border-primary hover:text-primary'"
          @click="go(p)"
        >
          {{ p }}
        </button>
      </template>

      <button
        class="flex h-8 min-w-8 items-center justify-center rounded border border-border px-2 text-muted-foreground transition-colors hover:border-primary hover:text-primary disabled:pointer-events-none disabled:opacity-40"
        :disabled="page >= pageCount"
        @click="go(page + 1)"
      >
        <ChevronRight class="size-4" />
      </button>
    </div>
  </div>
</template>
