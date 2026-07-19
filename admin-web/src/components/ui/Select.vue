<script setup lang="ts">
import { ref, computed, type Component } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { ChevronDown, Check } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

/**
 * 自定义下拉选择器（替代原生 select，统一设计风格）。
 * <Select v-model="val" :options="[{value,label}]" placeholder="请选择" class="w-32" />
 * 选项可选带 icon：{ value, label, icon: markRaw(SomeIcon) }
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
  class?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [v: string | number] }>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)
onClickOutside(root, () => (open.value = false))

const current = computed(() => props.options.find((o) => o.value === props.modelValue))

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
      enter-active-class="transition duration-150 ease-out"
      leave-active-class="transition duration-100 ease-in"
      enter-from-class="opacity-0 -translate-y-1"
      leave-to-class="opacity-0 -translate-y-1"
    >
      <div
        v-if="open"
        class="absolute left-0 top-full z-30 mt-1 max-h-64 w-full min-w-max overflow-auto rounded border border-border bg-popover py-1 shadow-lg"
      >
        <button
          v-for="o in options"
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
      </div>
    </transition>
  </div>
</template>
