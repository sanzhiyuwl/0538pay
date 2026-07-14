<script setup lang="ts">
import { X } from 'lucide-vue-next'

/**
 * 居中模态弹窗。用于账号设置 / 修改密码等中小型表单（参考 Element 弹窗风格）。
 * <Modal v-model="open" title="账号设置">
 *   ...内容（默认插槽）...
 *   <template #footer><Button>保存</Button></template>
 * </Modal>
 */
const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title?: string
    width?: string // 弹窗宽度类，默认 max-w-md
  }>(),
  { width: 'max-w-md' },
)
const emit = defineEmits<{ 'update:modelValue': [v: boolean] }>()

function close() {
  emit('update:modelValue', false)
}
</script>

<template>
  <transition
    enter-active-class="transition-opacity duration-200"
    leave-active-class="transition-opacity duration-150"
    enter-from-class="opacity-0"
    leave-to-class="opacity-0"
  >
    <div
      v-if="props.modelValue"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      @click.self="close"
    >
      <transition
        enter-active-class="transition-all duration-200 ease-out"
        leave-active-class="transition-all duration-150 ease-in"
        enter-from-class="opacity-0 scale-95"
        leave-to-class="opacity-0 scale-95"
        appear
      >
        <div
          v-if="props.modelValue"
          :class="['flex w-full flex-col border border-border bg-background shadow-2xl', props.width]"
        >
          <!-- 头 -->
          <div class="flex items-center gap-2 px-6 pt-5">
            <h3 class="text-base font-semibold">{{ title }}</h3>
            <div class="flex-1" />
            <button
              class="flex size-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent"
              @click="close"
            >
              <X class="size-[18px]" />
            </button>
          </div>

          <!-- 内容 -->
          <div class="px-6 py-5">
            <slot />
          </div>

          <!-- 底部操作 -->
          <div v-if="$slots.footer" class="flex items-center justify-end gap-2 px-6 pb-5">
            <slot name="footer" />
          </div>
        </div>
      </transition>
    </div>
  </transition>
</template>
