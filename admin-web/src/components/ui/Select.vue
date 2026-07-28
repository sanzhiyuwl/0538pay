<script setup lang="ts">
import { ref, computed, nextTick, watch, type Component } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { ChevronDown, Check, Search } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

/**
 * 自定义下拉选择器（替代原生 select，统一设计风格）。
 * <Select v-model="val" :options="[{value,label}]" placeholder="请选择" class="w-32" />
 * 选项可选带 icon：{ value, label, icon: markRaw(SomeIcon) }
 * searchable：选项多时（如代理列表）开启顶部搜索框，输入即过滤 label。
 */
interface Option {
  value: string | number
  label: string
  icon?: Component
}
const props = defineProps<{
  modelValue: string | number
  options: Option[]
  placeholder?: string
  searchable?: boolean
  class?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [v: string | number] }>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)
const keyword = ref('')
const searchInput = ref<HTMLInputElement | null>(null)
onClickOutside(root, () => (open.value = false))

const current = computed(() => props.options.find((o) => o.value === props.modelValue))

const filteredOptions = computed(() => {
  if (!props.searchable) return props.options
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return props.options
  return props.options.filter((o) => o.label.toLowerCase().includes(kw))
})

// 打开时清空关键字并聚焦搜索框；关闭时也清空，下次打开是干净状态。
watch(open, (v) => {
  keyword.value = ''
  if (v && props.searchable) nextTick(() => searchInput.value?.focus())
})

function pick(o: Option) {
  emit('update:modelValue', o.value)
  open.value = false
}
</script>

<template>
  <div ref="root" :class="cn('relative', props.class)">
    <button
      type="button"
      class="flex h-9 w-full items-center gap-2 rounded border border-input bg-background px-3 text-sm outline-none transition-colors hover:border-ring/60 focus:border-ring"
      :class="open && 'border-ring'"
      @click="open = !open"
    >
      <component :is="current.icon" v-if="current?.icon" class="size-4 shrink-0" />
      <span :class="current ? 'text-foreground' : 'text-muted-foreground'" class="flex-1 truncate text-left">
        {{ current?.label ?? placeholder ?? '请选择' }}
      </span>
      <ChevronDown :class="['size-4 shrink-0 text-muted-foreground transition-transform', open && 'rotate-180']" />
    </button>

    <transition
      enter-active-class="transition duration-150 ease-[cubic-bezier(0.23,1,0.32,1)]"
      leave-active-class="transition duration-100 ease-[cubic-bezier(0.23,1,0.32,1)]"
      enter-from-class="opacity-0 -translate-y-1"
      leave-to-class="opacity-0 -translate-y-1"
    >
      <div
        v-if="open"
        class="absolute left-0 top-full z-30 mt-1 flex max-h-72 w-full min-w-max flex-col rounded border border-border bg-popover py-1 shadow-lg"
      >
        <!-- 可搜索：顶部固定搜索框，输入即过滤 -->
        <div v-if="searchable" class="mx-1 mb-1 flex items-center gap-1.5 rounded bg-muted/40 px-2 py-1.5">
          <Search class="size-3.5 shrink-0 text-muted-foreground" />
          <input
            ref="searchInput"
            v-model="keyword"
            placeholder="输入关键字筛选"
            class="w-full bg-transparent text-sm outline-none"
            @keydown.stop
          />
        </div>
        <div class="flex-1 overflow-auto">
          <button
            v-for="o in filteredOptions"
            :key="o.value"
            type="button"
            class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors hover:bg-accent"
            :class="o.value === modelValue && 'bg-primary/[0.06] font-medium text-primary'"
            @click="pick(o)"
          >
            <component :is="o.icon" v-if="o.icon" class="size-4 shrink-0" />
            <span class="flex-1 whitespace-nowrap">{{ o.label }}</span>
            <Check v-if="o.value === modelValue" class="size-3.5 shrink-0 text-primary" />
          </button>
          <div v-if="!filteredOptions.length" class="px-3 py-4 text-center text-sm text-muted-foreground">无匹配项</div>
        </div>
      </div>
    </transition>
  </div>
</template>
