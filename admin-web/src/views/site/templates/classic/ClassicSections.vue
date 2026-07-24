<script setup lang="ts">
/**
 * classic 模板首页板块顺序渲染器。
 * 按 content.sections 过滤 visible 并按序渲染各板块组件。
 * 首页(ClassicHome)传 store.content；后台 CMS 预览传 draft + preview，天然实时联动。
 */
import { computed } from 'vue'
import type { Component } from 'vue'
import type { SectionKey, SiteContent } from '@/lib/mock/site-content'
import HeroSection from './sections/HeroSection.vue'
import MetricsSection from './sections/MetricsSection.vue'
import FeaturesSection from './sections/FeaturesSection.vue'
import PricingSection from './sections/PricingSection.vue'
import ProductsSection from './sections/ProductsSection.vue'
import NewsSection from './sections/NewsSection.vue'
import TestimonialsSection from './sections/TestimonialsSection.vue'
import CtaSection from './sections/CtaSection.vue'

const props = defineProps<{ content: SiteContent; preview?: boolean }>()

const compMap: Record<SectionKey, Component> = {
  hero: HeroSection,
  metrics: MetricsSection,
  features: FeaturesSection,
  pricing: PricingSection,
  products: ProductsSection,
  news: NewsSection,
  testimonials: TestimonialsSection,
  cta: CtaSection,
}

// 过滤可见板块，按 sections 顺序渲染
const visibleSections = computed(() => props.content.sections.filter((s) => s.visible))
</script>

<template>
  <div :class="preview ? 'is-preview' : ''">
    <component
      :is="compMap[s.key]"
      v-for="s in visibleSections"
      :key="s.key"
      :content="content"
      :preview="preview"
    />
  </div>
</template>

<style scoped>
/* 预览态：缩放容器里 IntersectionObserver 不触发进场，强制显示 v-reveal 元素 */
.is-preview :deep(.reveal) {
  opacity: 1 !important;
  transform: none !important;
}
</style>
