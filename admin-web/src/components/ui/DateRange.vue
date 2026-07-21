<script setup lang="ts">
import { ref, computed } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Clock, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

/**
 * 连体日期范围选择器（双月并排面板，参考图样式）。
 * <DateRange v-model:start="a" v-model:end="b" class="w-[320px]" />
 */
const props = defineProps<{
  start: string
  end: string
  class?: string
}>()
const emit = defineEmits<{ 'update:start': [v: string]; 'update:end': [v: string] }>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)
onClickOutside(root, () => (open.value = false))

// 面板默认双月并排(540px)。在窄容器（如导出抽屉 max-w-md，且 overflow-y-auto 会横向裁剪）里放不下，
// 故打开时测量"最近的裁剪祖先"可用空间：放得下就并排；两侧都放不下则改为双月竖排并收窄，保证完整可见。
const PANEL_W = 540
const alignRight = ref(false)
const stacked = ref(false)

// 找到最近的会裁剪内容的祖先（overflow 非 visible），拿它作为可用空间边界；找不到则用视口。
function findClip(el: HTMLElement): HTMLElement {
  let p = el.parentElement
  while (p) {
    const s = getComputedStyle(p)
    if (/(auto|scroll|hidden)/.test(s.overflowX + s.overflowY)) return p
    p = p.parentElement
  }
  return document.documentElement
}
function updateLayout() {
  const el = root.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const clip = findClip(el).getBoundingClientRect()
  const rightBound = Math.min(clip.right, window.innerWidth)
  const leftBound = Math.max(clip.left, 0)
  const spaceRight = rightBound - rect.left // 左对齐(向右展开)的可用宽
  const spaceLeft = rect.right - leftBound // 右对齐(向左展开)的可用宽
  if (spaceRight >= PANEL_W + 8) {
    stacked.value = false
    alignRight.value = false
  } else if (spaceLeft >= PANEL_W + 8) {
    stacked.value = false
    alignRight.value = true
  } else {
    // 两侧都塞不下并排双月 → 竖排收窄，对齐到空间更充裕的一侧
    stacked.value = true
    alignRight.value = spaceLeft > spaceRight
  }
}
function toggle() {
  open.value = !open.value
  if (open.value) updateLayout()
}

const weekDays = ['日', '一', '二', '三', '四', '五', '六']
const today = { y: 2026, mo: 7, d: 12 }
// 左侧面板年月（右侧 = 左侧 + 1 个月）
const viewY = ref(today.y)
const viewM = ref(today.mo)
const picking = ref<'start' | 'end'>('start')

function pad(n: number) {
  return String(n).padStart(2, '0')
}
function daysInMonth(y: number, m: number) {
  return new Date(y, m, 0).getDate()
}
function firstWeekday(y: number, m: number) {
  return new Date(y, m - 1, 1).getDay()
}
function buildGrid(y: number, m: number): (number | null)[] {
  const total = daysInMonth(y, m)
  const lead = firstWeekday(y, m)
  const cells: (number | null)[] = []
  for (let i = 0; i < lead; i++) cells.push(null)
  for (let d = 1; d <= total; d++) cells.push(d)
  return cells
}

// 左右两个月
const leftMonth = computed(() => ({ y: viewY.value, mo: viewM.value }))
const rightMonth = computed(() => {
  if (viewM.value === 12) return { y: viewY.value + 1, mo: 1 }
  return { y: viewY.value, mo: viewM.value + 1 }
})
const leftGrid = computed(() => buildGrid(leftMonth.value.y, leftMonth.value.mo))
const rightGrid = computed(() => buildGrid(rightMonth.value.y, rightMonth.value.mo))

function prevMonth() {
  if (viewM.value === 1) { viewM.value = 12; viewY.value-- } else viewM.value--
}
function nextMonth() {
  if (viewM.value === 12) { viewM.value = 1; viewY.value++ } else viewM.value++
}

function dstr(y: number, m: number, d: number) {
  return `${y}-${pad(m)}-${pad(d)}`
}
function inRange(y: number, m: number, d: number) {
  const s = props.start, e = props.end
  if (!s || !e) return false
  const cur = dstr(y, m, d)
  return cur > s && cur < e
}
function isEnd(y: number, m: number, d: number) {
  const cur = dstr(y, m, d)
  return cur === props.start || cur === props.end
}
function pick(y: number, m: number, d: number) {
  const v = dstr(y, m, d)
  if (picking.value === 'start') {
    emit('update:start', v)
    emit('update:end', '')
    picking.value = 'end'
  } else {
    if (v < props.start) {
      emit('update:end', props.start)
      emit('update:start', v)
    } else {
      emit('update:end', v)
    }
    picking.value = 'start'
  }
}
function clear() {
  emit('update:start', '')
  emit('update:end', '')
  picking.value = 'start'
}
function confirm() {
  open.value = false
}
</script>

<template>
  <div ref="root" :class="cn('relative', props.class)">
    <!-- 连体触发框 -->
    <button
      type="button"
      class="flex h-9 w-full items-center gap-2 rounded border border-input bg-background px-3 text-sm outline-none transition-colors hover:border-ring/60"
      :class="open && 'border-ring'"
      @click="toggle"
    >
      <Clock class="size-4 shrink-0 text-muted-foreground" />
      <span :class="start ? 'text-foreground' : 'text-muted-foreground'" class="tabular-nums">{{ start || '开始时间' }}</span>
      <span class="text-muted-foreground">-</span>
      <span :class="end ? 'text-foreground' : 'text-muted-foreground'" class="tabular-nums">{{ end || '结束时间' }}</span>
    </button>

    <transition
      enter-active-class="transition duration-150 ease-out"
      leave-active-class="transition duration-100 ease-in"
      enter-from-class="opacity-0 -translate-y-1"
      leave-to-class="opacity-0 -translate-y-1"
    >
      <div
        v-if="open"
        class="absolute top-full z-30 mt-1 rounded border border-border bg-popover shadow-lg"
        :class="[
          alignRight ? 'right-0' : 'left-0',
          stacked ? 'w-[300px] max-w-[calc(100vw-2rem)]' : 'w-[540px]',
        ]"
      >
        <!-- 提示 -->
        <div class="border-b border-border/70 px-4 py-2 text-xs text-muted-foreground">
          {{ picking === 'start' ? '请选择开始日期' : '请选择结束日期' }}
        </div>

        <div :class="stacked ? 'flex flex-col' : 'flex'">
          <!-- 左月 -->
          <div class="flex-1 p-3" :class="stacked ? 'border-b border-border/70' : 'border-r border-border/70'">
            <div class="mb-2 flex items-center justify-between">
              <button type="button" class="flex size-7 items-center justify-center rounded-sm text-muted-foreground hover:bg-accent" @click="prevMonth"><ChevronLeft class="size-4" /></button>
              <span class="text-sm font-medium tabular-nums">{{ leftMonth.y }} 年 {{ leftMonth.mo }} 月</span>
              <span class="size-7" />
            </div>
            <div class="grid grid-cols-7 text-center text-xs text-muted-foreground">
              <span v-for="w in weekDays" :key="w" class="py-1">{{ w }}</span>
            </div>
            <div class="grid grid-cols-7 gap-0.5 text-center text-sm">
              <template v-for="(c, i) in leftGrid" :key="i">
                <span v-if="c === null" />
                <button
                  v-else
                  type="button"
                  class="flex size-8 items-center justify-center rounded-sm tabular-nums transition-colors hover:bg-accent"
                  :class="[
                    isEnd(leftMonth.y, leftMonth.mo, c) && '!bg-primary font-medium text-primary-foreground hover:!bg-primary',
                    !isEnd(leftMonth.y, leftMonth.mo, c) && inRange(leftMonth.y, leftMonth.mo, c) && 'bg-primary/10 text-primary',
                  ]"
                  @click="pick(leftMonth.y, leftMonth.mo, c)"
                >
                  {{ c }}
                </button>
              </template>
            </div>
          </div>

          <!-- 右月 -->
          <div class="flex-1 p-3">
            <div class="mb-2 flex items-center justify-between">
              <span class="size-7" />
              <span class="text-sm font-medium tabular-nums">{{ rightMonth.y }} 年 {{ rightMonth.mo }} 月</span>
              <button type="button" class="flex size-7 items-center justify-center rounded-sm text-muted-foreground hover:bg-accent" @click="nextMonth"><ChevronRight class="size-4" /></button>
            </div>
            <div class="grid grid-cols-7 text-center text-xs text-muted-foreground">
              <span v-for="w in weekDays" :key="w" class="py-1">{{ w }}</span>
            </div>
            <div class="grid grid-cols-7 gap-0.5 text-center text-sm">
              <template v-for="(c, i) in rightGrid" :key="i">
                <span v-if="c === null" />
                <button
                  v-else
                  type="button"
                  class="flex size-8 items-center justify-center rounded-sm tabular-nums transition-colors hover:bg-accent"
                  :class="[
                    isEnd(rightMonth.y, rightMonth.mo, c) && '!bg-primary font-medium text-primary-foreground hover:!bg-primary',
                    !isEnd(rightMonth.y, rightMonth.mo, c) && inRange(rightMonth.y, rightMonth.mo, c) && 'bg-primary/10 text-primary',
                  ]"
                  @click="pick(rightMonth.y, rightMonth.mo, c)"
                >
                  {{ c }}
                </button>
              </template>
            </div>
          </div>
        </div>

        <!-- 底部 -->
        <div class="flex items-center justify-end gap-2 border-t border-border/70 px-4 py-2.5">
          <button type="button" class="text-xs text-muted-foreground transition-colors hover:text-primary" @click="clear">清空</button>
          <button type="button" class="rounded bg-primary px-3 py-1 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90" @click="confirm">确定</button>
        </div>
      </div>
    </transition>
  </div>
</template>
