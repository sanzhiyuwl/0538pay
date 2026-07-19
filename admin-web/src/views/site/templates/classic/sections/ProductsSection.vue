<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight, Check } from 'lucide-vue-next'
import type { SiteContent } from '@/lib/mock/site-content'
import { icon } from '../shared'
import ProductArt3D from '../ProductArt3D.vue'

const props = defineProps<{ content: SiteContent; preview?: boolean }>()
const router = useRouter()

const activeProduct = ref(0)
// 编辑预览时产品数量可能变化，越界则回退首项
const current = computed(() => props.content.products[activeProduct.value] ?? props.content.products[0])
watch(
  () => props.content.products.length,
  (len) => { if (activeProduct.value >= len) activeProduct.value = 0 },
)
</script>

<template>
  <!-- ===== 产品矩阵（浅色风：淡背景图 + 白卡面板 + 左tab右详情 + 发光插画）===== -->
  <section id="products" class="relative scroll-mt-20 overflow-hidden bg-content">
    <!-- 背景图（完整显示，不加渐隐遮罩）-->
    <div class="pointer-events-none absolute inset-0 z-0" aria-hidden="true">
      <img src="/assets/products-bg.jpg" alt="" class="size-full object-cover" @error="($event.target as HTMLImageElement).style.display = 'none'" />
    </div>

    <div class="relative z-10 mx-auto max-w-7xl px-4 py-24 lg:px-6">
      <!-- 标题（背景图偏深，用白字 + 阴影保证可读）-->
      <div v-reveal class="mx-auto max-w-3xl text-center">
        <h2 class="text-3xl font-bold tracking-tight text-white drop-shadow-[0_2px_8px_rgba(0,0,0,0.4)] sm:text-4xl">{{ content.productsTitle }}</h2>
        <p class="mt-3 text-[15px] leading-relaxed text-white/80 drop-shadow-[0_1px_4px_rgba(0,0,0,0.35)]">{{ content.productsSubtitle }}</p>
      </div>

      <!-- 面板（整体深色 #262B39 半透明，透出背景图；左 tab + 中详情 + 右插画）-->
      <div v-reveal class="mt-14 grid overflow-hidden bg-[#262B39]/60 shadow-[0_20px_50px_-20px_rgba(0,0,0,0.5)] backdrop-blur-md lg:grid-cols-[240px_1fr_260px]">
        <!-- 左：产品 tab（浅蓝选中态）-->
        <div class="relative overflow-hidden bg-[#262B39]/45 p-5">
          <!-- 背景装饰：淡品牌色渐变（自上而下轻染，干净不花）-->
          <div class="pointer-events-none absolute inset-0 z-0 bg-gradient-to-b from-primary/15 to-transparent" aria-hidden="true" />
          <div class="relative z-10 hidden px-3 pb-3 pt-1 text-xs font-medium text-white/45 lg:block">支付产品</div>
          <div class="relative z-10 flex flex-row gap-1.5 overflow-x-auto lg:flex-col lg:overflow-visible">
            <button
              v-for="(p, i) in content.products"
              :key="p.name"
              type="button"
              class="group flex shrink-0 items-center gap-3 rounded-lg px-4 py-3.5 text-left text-[15px] font-medium outline-none transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
              :class="activeProduct === i
                ? 'bg-primary/20 font-semibold text-[#7db4ff]'
                : 'text-white/70 hover:bg-white/[0.06] hover:text-white'"
              @click="activeProduct = i"
            >
              <component
                :is="icon(p.icon)"
                class="size-[18px] shrink-0"
                :class="activeProduct === i ? 'text-[#7db4ff]' : 'text-white/55 group-hover:text-white'"
              />
              <span class="flex-1 truncate">{{ p.name }}</span>
              <ArrowRight class="size-4 shrink-0 transition-all" :class="activeProduct === i ? 'opacity-100' : '-translate-x-1 opacity-0'" />
            </button>
          </div>
        </div>

        <!-- 中：选中产品详情（透明，只透出主面板；文案 + 能力项）-->
        <div v-if="current" class="relative overflow-hidden p-8 lg:p-14">
          <div class="relative z-10">
            <!-- 文案 + 能力项 -->
            <div class="min-w-0">
              <!-- 标题行：图标 chip + 名称 -->
              <div class="flex items-center gap-3.5">
                <span class="flex size-12 shrink-0 items-center justify-center rounded-xl bg-primary/15 text-[#7db4ff]">
                  <component :is="icon(current.icon)" class="size-6" />
                </span>
                <h3 class="text-2xl font-bold tracking-tight text-white">{{ current.name }}</h3>
              </div>
              <p class="mt-5 max-w-xl text-[15px] leading-relaxed text-white/60">{{ current.desc }}</p>

              <!-- 能力项：图标 + 文字，装进浅色小卡（图标顶对齐，文字自然占满、换行整齐）-->
              <div class="mt-7 grid grid-cols-1 gap-2.5 sm:grid-cols-2">
                <div
                  v-for="pt in current.points"
                  :key="pt"
                  class="flex items-start gap-2.5 rounded-lg bg-white/[0.05] px-3.5 py-3"
                >
                  <span class="mt-px flex size-[22px] shrink-0 items-center justify-center rounded-md bg-primary/25 text-[#7db4ff]">
                    <Check class="size-3.5" :stroke-width="2.75" />
                  </span>
                  <span class="text-[13px] leading-snug text-white/80">{{ pt }}</span>
                </div>
              </div>

              <!-- 分隔线 -->
              <div class="my-7 h-px bg-white/10" />

              <!-- 场景标签 + CTA -->
              <div class="flex flex-wrap items-center gap-2.5">
                <span class="mr-1 text-xs text-white/50">适用场景</span>
                <span
                  v-for="tag in current.tags"
                  :key="tag"
                  class="inline-flex items-center rounded-full border border-white/15 bg-white/[0.05] px-3 py-1 text-xs text-white/65"
                >{{ tag }}</span>
                <button class="ml-auto inline-flex items-center gap-1.5 rounded-lg bg-primary px-5 py-2.5 text-sm font-medium text-white shadow-[0_8px_20px_-8px_rgba(0,98,239,0.6)] transition-colors hover:bg-[#0052cc]" @click="router.push('/m/reg')">
                  立即接入 <ArrowRight class="size-4" />
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- 右：发光 3D 插画（背景与左 tab 一致的淡品牌色渐变，收住右侧不再乱）-->
        <div v-if="current" class="relative hidden bg-[#262B39]/45 lg:block">
          <!-- 背景装饰：与 tab 同款淡品牌色渐变（自下而上轻染）-->
          <div class="pointer-events-none absolute inset-0 z-0 bg-gradient-to-t from-primary/15 to-transparent" aria-hidden="true" />
          <div class="relative z-10 flex h-full items-center justify-center p-4">
            <!-- 等距 3D 插画（纯 SVG，统一底座台，切 tab 换图标）-->
            <ProductArt3D :icon="icon(current.icon)" />
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
