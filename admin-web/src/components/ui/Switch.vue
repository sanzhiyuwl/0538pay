<script setup lang="ts">
import { cn } from '@/lib/utils'

/**
 * 开关（toggle）。统一的启用/禁用切换控件。
 * <Switch v-model="enabled" />
 * <Switch v-model="enabled" size="sm" :disabled="loading" />
 */
const props = withDefaults(
  defineProps<{
    modelValue: boolean
    size?: 'default' | 'sm'
    disabled?: boolean
    class?: string
  }>(),
  { size: 'default', disabled: false },
)
const emit = defineEmits<{ 'update:modelValue': [v: boolean] }>()

function toggle() {
  if (props.disabled) return
  emit('update:modelValue', !props.modelValue)
}
</script>

<template>
  <button
    type="button"
    role="switch"
    :aria-checked="props.modelValue"
    :disabled="props.disabled"
    class="relative inline-flex shrink-0 items-center rounded-full px-0.5 transition-colors duration-200 disabled:cursor-not-allowed disabled:opacity-50"
    :class="
      cn(
        props.size === 'sm' ? 'h-5 w-9' : 'h-6 w-11',
        props.modelValue ? 'bg-primary' : 'bg-muted-foreground/30',
        props.class,
      )
    "
    @click="toggle"
  >
    <span
      class="rounded-full bg-white shadow-sm ring-1 ring-black/5 transition-transform duration-200"
      :class="
        props.size === 'sm'
          ? ['size-4', props.modelValue ? 'translate-x-4' : 'translate-x-0']
          : ['size-5', props.modelValue ? 'translate-x-5' : 'translate-x-0']
      "
    />
  </button>
</template>
