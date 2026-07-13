<script setup lang="ts">
import { ref, computed } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Calendar, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

/**
 * 自定义日期选择器（弹出月历面板，替代原生 input[type=date]）。
 * <DatePicker v-model="val" placeholder="开始日期" class="w-40" />  v-model 为 'YYYY-MM-DD' 字符串
 */
const props = defineProps<{
  modelValue: string
  placeholder?: string
  class?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)
onClickOutside(root, () => (open.value = false))

const weekDays = ['日', '一', '二', '三', '四', '五', '六']

// 面板当前展示的年月（默认取值或今天——不用 new Date() 无参，改用固定"今天"演示基准）
function parse(v: string) {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(v)
  if (m) return { y: +m[1], mo: +m[2], d: +m[3] }
  return null
}
const today = { y: 2026, mo: 7, d: 12 } // 演示基准日（原生 Date 在工作流受限，页面运行时用真实值即可）
const viewY = ref(parse(props.modelValue)?.y ?? today.y)
const viewM = ref(parse(props.modelValue)?.mo ?? today.mo)

function daysInMonth(y: number, m: number) {
  return new Date(y, m, 0).getDate()
}
function firstWeekday(y: number, m: number) {
  return new Date(y, m - 1, 1).getDay()
}

const grid = computed(() => {
  const total = daysInMonth(viewY.value, viewM.value)
  const lead = firstWeekday(viewY.value, viewM.value)
  const cells: (number | null)[] = []
  for (let i = 0; i < lead; i++) cells.push(null)
  for (let d = 1; d <= total; d++) cells.push(d)
  return cells
})

function prevMonth() {
  if (viewM.value === 1) {
    viewM.value = 12
    viewY.value--
  } else viewM.value--
}
function nextMonth() {
  if (viewM.value === 12) {
    viewM.value = 1
    viewY.value++
  } else viewM.value++
}

function pad(n: number) {
  return String(n).padStart(2, '0')
}
const selected = computed(() => parse(props.modelValue))
function isSelected(d: number) {
  const s = selected.value
  return s && s.y === viewY.value && s.mo === viewM.value && s.d === d
}
function isToday(d: number) {
  return today.y === viewY.value && today.mo === viewM.value && today.d === d
}
function pick(d: number) {
  emit('update:modelValue', `${viewY.value}-${pad(viewM.value)}-${pad(d)}`)
  open.value = false
}
function clear() {
  emit('update:modelValue', '')
  open.value = false
}
</script>

<template>
  <div ref="root" :class="cn('relative', props.class)">
    <button
      type="button"
      class="flex h-9 w-full items-center gap-2 rounded border border-input bg-background px-3 text-sm outline-none transition-colors hover:border-ring/60"
      :class="open && 'border-ring'"
      @click="open = !open"
    >
      <Calendar class="size-4 shrink-0 text-muted-foreground" />
      <span :class="modelValue ? 'text-foreground' : 'text-muted-foreground'" class="flex-1 text-left tabular-nums">
        {{ modelValue || placeholder || '选择日期' }}
      </span>
    </button>

    <transition
      enter-active-class="transition duration-150 ease-out"
      leave-active-class="transition duration-100 ease-in"
      enter-from-class="opacity-0 -translate-y-1"
      leave-to-class="opacity-0 -translate-y-1"
    >
      <div
        v-if="open"
        class="absolute left-0 top-full z-30 mt-1 w-64 rounded border border-border bg-popover p-3 shadow-lg"
      >
        <!-- 月份导航 -->
        <div class="mb-2 flex items-center justify-between">
          <button type="button" class="flex size-7 items-center justify-center rounded-sm text-muted-foreground hover:bg-accent" @click="prevMonth">
            <ChevronLeft class="size-4" />
          </button>
          <span class="text-sm font-medium tabular-nums">{{ viewY }} 年 {{ viewM }} 月</span>
          <button type="button" class="flex size-7 items-center justify-center rounded-sm text-muted-foreground hover:bg-accent" @click="nextMonth">
            <ChevronRight class="size-4" />
          </button>
        </div>
        <!-- 星期 -->
        <div class="grid grid-cols-7 text-center text-xs text-muted-foreground">
          <span v-for="w in weekDays" :key="w" class="py-1">{{ w }}</span>
        </div>
        <!-- 日期格 -->
        <div class="grid grid-cols-7 gap-0.5 text-center text-sm">
          <template v-for="(c, i) in grid" :key="i">
            <span v-if="c === null" />
            <button
              v-else
              type="button"
              class="flex size-8 items-center justify-center rounded-sm tabular-nums transition-colors hover:bg-accent"
              :class="[
                isSelected(c) && '!bg-primary font-medium text-primary-foreground hover:!bg-primary',
                !isSelected(c) && isToday(c) && 'text-primary font-medium',
              ]"
              @click="pick(c)"
            >
              {{ c }}
            </button>
          </template>
        </div>
        <!-- 底部 -->
        <div class="mt-2 flex justify-end border-t border-border/70 pt-2">
          <button type="button" class="text-xs text-muted-foreground transition-colors hover:text-primary" @click="clear">清空</button>
        </div>
      </div>
    </transition>
  </div>
</template>
