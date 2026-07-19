<script setup lang="ts">
/**
 * Toast 渲染宿主（对齐 wepay 后台 uiToast 风格）。全局挂一个（App.vue），配合 useToast() 使用。
 * 顶部居中，白底 + 彩色细边框 + 彩色实心圆图标(Bootstrap Icons 同款) + 同色文字，下滑淡入、自动消失。
 * 白底用 var(--background) 以兼容暗色模式（wepay 为纯亮色，此处扩展）。
 */
import { useToast } from '@/composables/useToast'

const { toasts } = useToast()

// Bootstrap Icons v1.11 实心圆路径（viewBox 0 0 16 16），与 wepay uiToast 用的 bi-*-fill 一致
const PATH = {
  success: 'M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zm-3.97-3.03a.75.75 0 0 0-1.08.022L7.477 9.417 5.384 7.323a.75.75 0 0 0-1.06 1.06L6.97 11.03a.75.75 0 0 0 1.079-.02l3.992-4.99a.75.75 0 0 0-.01-1.05z',
  error: 'M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zM5.354 4.646a.5.5 0 1 0-.708.708L7.293 8l-2.647 2.646a.5.5 0 0 0 .708.708L8 8.707l2.646 2.647a.5.5 0 0 0 .708-.708L8.707 8l2.647-2.646a.5.5 0 0 0-.708-.708L8 7.293z',
  info: 'M8 16A8 8 0 1 0 8 0a8 8 0 0 0 0 16zm.93-9.412-1 4.705c-.07.34.029.533.304.533.194 0 .487-.07.686-.246l-.088.416c-.287.346-.92.598-1.465.598-.703 0-1.002-.422-.808-1.319l.738-3.468c.064-.293.006-.399-.287-.47l-.451-.081.082-.381 2.29-.287zM8 5.5a1 1 0 1 1 0-2 1 1 0 0 1 0 2z',
}
// 1:1 复用 Badge 的 Element UI 淡底描边配色：淡底 + 更浅同色描边 + 同色字/图标
const styleOf: Record<string, string> = {
  success: 'border-[#e1f3d8] bg-[#f0f9eb] text-[#67C23A]',
  error: 'border-[#fde2e2] bg-[#fef0f0] text-[#F56C6C]',
  info: 'border-[#d9ecff] bg-[#ecf5ff] text-[#409EFF]',
}
</script>

<template>
  <Teleport to="body">
    <div class="pointer-events-none fixed left-1/2 top-3.5 z-[100] flex -translate-x-1/2 flex-col items-center gap-2">
      <transition-group
        enter-active-class="transition duration-200 ease-out"
        leave-active-class="transition duration-150 ease-in"
        enter-from-class="opacity-0 -translate-y-3"
        leave-to-class="opacity-0 -translate-y-3"
        move-class="transition duration-200"
      >
        <div
          v-for="t in toasts"
          :key="t.id"
          class="pointer-events-auto flex items-center gap-2 rounded border px-4 py-2 text-[13px] font-medium shadow-[0_4px_16px_rgba(0,0,0,0.06)]"
          :class="styleOf[t.type]"
        >
          <svg viewBox="0 0 16 16" class="size-4 shrink-0" fill="currentColor" aria-hidden="true">
            <path :d="PATH[t.type]" />
          </svg>
          <span class="leading-none">{{ t.message }}</span>
        </div>
      </transition-group>
    </div>
  </Teleport>
</template>
