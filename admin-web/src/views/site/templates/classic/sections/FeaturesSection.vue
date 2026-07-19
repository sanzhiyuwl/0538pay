<script setup lang="ts">
import type { SiteContent } from '@/lib/mock/site-content'
import { icon } from '../shared'

defineProps<{ content: SiteContent; preview?: boolean }>()
</script>

<template>
  <!-- ===== 核心特性 ===== -->
  <section id="features" class="scroll-mt-20 bg-background">
    <div class="mx-auto max-w-7xl px-4 pb-0 pt-24 lg:px-6">
      <div v-reveal class="mx-auto max-w-2xl text-center">
        <h2 class="text-3xl font-bold tracking-tight">{{ content.featuresTitle }}</h2>
        <p class="mt-3 text-muted-foreground">{{ content.featuresSubtitle }}</p>
      </div>
      <div v-reveal class="mt-14 grid border-l border-t border-border/50 sm:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="f in content.features"
          :key="f.title"
          class="group relative flex flex-col items-start border-b border-r border-border/50 bg-background p-8 transition-colors hover:bg-content/50"
        >
          <!-- 顶部品牌线：hover 从左往右展开 -->
          <span class="absolute inset-x-0 top-0 h-0.5 origin-left scale-x-0 bg-primary transition-transform duration-300 ease-out group-hover:scale-x-100" aria-hidden="true" />
          <!-- 直角淡蓝方 chip（与产品矩阵图标语言统一），hover 反白 -->
          <span class="mb-5 flex size-12 items-center justify-center rounded-lg bg-primary/10 text-primary transition-colors group-hover:bg-primary group-hover:text-white">
            <component :is="icon(f.icon)" class="size-6" :stroke-width="1.75" />
          </span>
          <h3 class="text-base font-semibold">{{ f.title }}</h3>
          <p class="mt-2 text-sm leading-relaxed text-muted-foreground">{{ f.desc }}</p>
        </div>
      </div>
    </div>
  </section>
</template>
