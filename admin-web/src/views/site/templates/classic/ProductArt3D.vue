<script setup lang="ts">
/**
 * 产品插画：直接用产品图标，做成干净的发光大图标（不依赖外部图片）。
 * 圆角方牌承托 + 品牌色发光，切 tab 换图标，风格统一。
 */
import type { Component } from 'vue'

defineProps<{ icon: Component }>()
</script>

<template>
  <div class="relative flex size-56 -translate-y-10 items-center justify-center">
    <!-- 品牌色柔光晕 -->
    <div class="pa-glow pointer-events-none absolute size-52 rounded-full" aria-hidden="true" />

    <!-- 底座图（立体块脚下的台座，加载失败自动隐藏）-->
    <img
      src="/assets/dizuo.png"
      alt=""
      aria-hidden="true"
      class="pointer-events-none absolute -bottom-32 left-1/2 z-0 w-64 -translate-x-1/2"
      @error="($event.target as HTMLImageElement).style.display = 'none'"
    />

    <!-- 图标承托方牌 -->
    <div class="pa-tile relative z-10 flex size-28 items-center justify-center rounded-[1.5rem]">
      <component :is="icon" class="size-12 text-white drop-shadow-[0_4px_14px_rgba(0,40,120,0.55)]" :stroke-width="1.6" />
    </div>
  </div>
</template>

<style scoped>
.pa-glow {
  background: radial-gradient(circle, rgba(0, 98, 239, 0.5) 0%, transparent 68%);
  filter: blur(36px);
}

/* 承托方牌：蓝色立体渐变 + 顶部高光 + 外发光，轻柔浮动 */
.pa-tile {
  background: linear-gradient(155deg, #4a90ff 0%, #0062ef 55%, #0048b5 100%);
  border: 1px solid rgba(255, 255, 255, 0.22);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.5),
    0 20px 42px -14px rgba(0, 98, 239, 0.7);
  animation: pa-float 5s ease-in-out infinite;
}
.pa-tile::before {
  content: "";
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: linear-gradient(155deg, rgba(255, 255, 255, 0.28) 0%, transparent 46%);
  pointer-events: none;
}

/* 整体上挑 -32px，浮动在此基准上下 10px */
@keyframes pa-float {
  0%, 100% { transform: translateY(-32px); }
  50% { transform: translateY(-42px); }
}
@media (prefers-reduced-motion: reduce) {
  .pa-tile { animation: none; }
}
</style>
