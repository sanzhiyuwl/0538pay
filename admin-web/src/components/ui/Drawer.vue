<script setup lang="ts">
import { X } from 'lucide-vue-next'

/**
 * 通用右侧抽屉。表单 / 详情等复杂内容的容器。
 * <Drawer v-model="open" title="创建租户" subtitle="...">
 *   ...内容（默认插槽）...
 *   <template #footer><Button>保存</Button></template>
 * </Drawer>
 */
const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title?: string
    subtitle?: string
    width?: string // 抽屉宽度类，默认 max-w-lg
  }>(),
  { width: 'max-w-lg' },
)
const emit = defineEmits<{ 'update:modelValue': [v: boolean] }>()

function close() {
  emit('update:modelValue', false)
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
    <div v-if="props.modelValue" class="fixed inset-0 z-50 bg-black/30" @click="close" />
  </transition>

  <!-- 抽屉 -->
  <transition
    enter-active-class="transition-transform duration-300 ease-out"
    leave-active-class="transition-transform duration-200 ease-in"
    enter-from-class="translate-x-full"
    leave-to-class="translate-x-full"
  >
    <aside
      v-if="props.modelValue"
      :class="['fixed right-0 top-0 z-50 flex h-full w-full flex-col bg-background shadow-2xl', props.width]"
    >
      <!-- 头 -->
      <div class="flex items-center gap-2 border-b border-border px-5 py-4">
        <div>
          <h3 class="text-[15px] font-semibold">{{ title }}</h3>
          <p v-if="subtitle" class="mt-0.5 text-xs text-muted-foreground">{{ subtitle }}</p>
        </div>
        <div class="flex-1" />
        <button
          class="flex size-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent"
          @click="close"
        >
          <X class="size-[18px]" />
        </button>
      </div>

      <!-- 内容 -->
      <div class="flex-1 overflow-y-auto px-5 py-4">
        <slot />
      </div>

      <!-- 底部操作 -->
      <div v-if="$slots.footer" class="flex items-center justify-end gap-2 border-t border-border px-5 py-3">
        <slot name="footer" />
      </div>
    </aside>
  </transition>
</template>
