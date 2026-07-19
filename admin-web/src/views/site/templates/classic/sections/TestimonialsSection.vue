<script setup lang="ts">
import { ref } from 'vue'
import { Store, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import type { SiteContent } from '@/lib/mock/site-content'

defineProps<{ content: SiteContent; preview?: boolean }>()

// 合作商户案例：横向滚动轮播（箭头按一屏宽度滚动）
const casesEl = ref<HTMLElement | null>(null)
function scrollCases(dir: 1 | -1) {
  const el = casesEl.value
  if (!el) return
  el.scrollBy({ left: dir * el.clientWidth * 0.8, behavior: 'smooth' })
}
</script>

<template>
  <!-- ===== 合作商户案例（门店实拍图 + 商户名 + 简介，横向滚动轮播）===== -->
  <section class="bg-background">
    <div class="mx-auto max-w-7xl px-4 py-24 lg:px-6">
      <!-- 标题：居中 -->
      <div v-reveal class="mx-auto max-w-2xl text-center">
        <h2 class="text-3xl font-bold tracking-tight">{{ content.testimonialsTitle }}</h2>
        <p class="mt-3 text-muted-foreground">{{ content.testimonialsSubtitle }}</p>
      </div>

      <!-- 轮播区：轨道 + 两侧垂直居中的细线箭头 -->
      <div v-reveal class="relative mt-14">
        <!-- 左右箭头：浮在轨道两侧外侧，垂直居中（细线无边框）-->
        <button
          type="button"
          class="absolute -left-9 top-[38%] z-10 hidden -translate-y-1/2 text-muted-foreground/50 transition-colors hover:text-primary lg:block xl:-left-16"
          aria-label="上一个"
          @click="scrollCases(-1)"
        ><ChevronLeft class="size-8" :stroke-width="1.5" /></button>
        <button
          type="button"
          class="absolute -right-9 top-[38%] z-10 hidden -translate-y-1/2 text-muted-foreground/50 transition-colors hover:text-primary lg:block xl:-right-16"
          aria-label="下一个"
          @click="scrollCases(1)"
        ><ChevronRight class="size-8" :stroke-width="1.5" /></button>

        <!-- 横向滚动轨道：每卡定宽，snap 对齐，隐藏滚动条 -->
        <div
          ref="casesEl"
          class="no-scrollbar flex snap-x snap-mandatory gap-6 overflow-x-auto scroll-smooth pb-2"
        >
          <div
            v-for="t in content.testimonials"
            :key="t.name"
            class="group w-[78%] shrink-0 snap-start sm:w-[calc((100%-1.5rem)/2)] lg:w-[calc((100%-6rem)/5)]"
          >
            <!-- 门店实拍图（或占位图块）：4:3 比例 -->
            <div class="relative aspect-[4/3] overflow-hidden bg-muted">
              <img
                v-if="t.image"
                :src="t.image"
                :alt="t.name"
                class="size-full object-cover transition-transform duration-500 group-hover:scale-105"
              />
              <!-- 占位图块：淡灰渐变 + 首字大字 + 门店图标（后台可换真实门店照）-->
              <div v-else class="flex size-full flex-col items-center justify-center bg-gradient-to-br from-[#eef0f3] to-[#dfe2e7]">
                <span class="text-5xl font-bold text-foreground/15">{{ t.name.slice(0, 1) }}</span>
                <Store class="mt-2 size-6 text-foreground/20" :stroke-width="1.5" />
              </div>
            </div>
            <!-- 商户名 + 简介 -->
            <h3 class="mt-5 text-lg font-bold tracking-tight">{{ t.name }}</h3>
            <p class="mt-2.5 line-clamp-3 text-sm leading-relaxed text-muted-foreground">{{ t.desc }}</p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
/* 合作商户案例：隐藏横向滚动条（保留可滚动）*/
.no-scrollbar {
  scrollbar-width: none;
  -ms-overflow-style: none;
}
.no-scrollbar::-webkit-scrollbar {
  display: none;
}
</style>
