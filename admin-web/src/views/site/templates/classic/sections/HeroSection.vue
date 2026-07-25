<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight } from 'lucide-vue-next'
import { Button, Badge } from '@/components/ui'
import type { SiteContent } from '@/lib/mock/site-content'
import { useSiteStore } from '@/stores/site'
import { payMethods, heroShares, bars } from '../shared'

const props = defineProps<{ content: SiteContent; preview?: boolean }>()
const router = useRouter()

// ===== 幻灯片：开启且至少一张有效图片时，整个 Hero 换成大图轮播 =====
const siteStore = useSiteStore()
const slides = computed(() => (siteStore.config.slides || []).filter((s) => s && s.image))
const useSlides = computed(() => !!siteStore.config.slidesOn && slides.value.length > 0)

const current = ref(0)
const count = computed(() => slides.value.length)
let timer: number | undefined
function play() {
  stop()
  if (props.preview || count.value <= 1) return
  const ms = Math.max(2, siteStore.config.slideInterval || 5) * 1000
  timer = window.setInterval(() => go(current.value + 1), ms)
}
function stop() {
  if (timer) {
    window.clearInterval(timer)
    timer = undefined
  }
}
function go(i: number) {
  current.value = (i + count.value) % count.value
}
function jump(i: number) {
  go(i)
  play()
}
watch([count, useSlides], () => {
  if (current.value >= count.value) current.value = 0
  play()
})
onMounted(play)
onBeforeUnmount(stop)
</script>

<template>
  <!-- ===== 首屏 Hero：文案/按钮/概览卡恒在；背景为「幻灯片轮播」或「默认点阵流光」===== -->
  <section
    class="relative -mt-16 overflow-hidden border-b border-border"
    :class="useSlides ? 'hero-has-slides' : 'bg-background'"
    @mouseenter="useSlides && stop()"
    @mouseleave="useSlides && play()"
  >
    <!-- 背景 A：幻灯片轮播（后台配置了幻灯片时）+ 半透明遮罩保证前景文字可读 -->
    <div v-if="useSlides" class="pointer-events-none absolute inset-0 z-0" aria-hidden="true">
      <div
        v-for="(s, i) in slides"
        :key="i"
        class="slide-bg absolute inset-0"
        :class="{ active: i === current }"
      >
        <img :src="s.image" :alt="s.title || `幻灯片 ${i + 1}`" class="size-full object-cover" />
      </div>
      <!-- 遮罩：浅色蒙层，让深色文案在图上可读（左侧更重，呼应左文案）-->
      <div class="absolute inset-0 bg-gradient-to-r from-background/92 via-background/78 to-background/45" />
    </div>

    <!-- 背景 B：默认点阵流光（未配置幻灯片时）-->
    <div v-else class="pointer-events-none absolute inset-0 z-0" aria-hidden="true">
      <div class="hero-beam absolute inset-0" />
      <div class="hero-dots absolute inset-x-0 top-0 h-2/3" />
      <div class="hero-glow absolute -right-28 -top-44 size-[44rem] opacity-90" />
      <div class="hero-glow absolute -left-44 top-16 size-[34rem] opacity-55" />
    </div>

    <!-- 幻灯片圆点指示器（多于一张时）-->
    <div
      v-if="useSlides && count > 1"
      class="absolute inset-x-0 bottom-5 z-20 flex items-center justify-center gap-2"
    >
      <button
        v-for="(_, i) in slides"
        :key="i"
        type="button"
        class="dot"
        :class="{ active: i === current }"
        :aria-label="`第 ${i + 1} 张`"
        @click="jump(i)"
      />
    </div>

    <div class="relative z-10 mx-auto grid max-w-7xl items-center gap-12 px-4 pb-20 pt-36 lg:grid-cols-[1.05fr_1fr] lg:px-8 lg:pb-28 lg:pt-44">
      <!-- 左：文案 -->
      <div v-reveal>
        <div class="inline-flex items-center gap-2 rounded-full border border-border bg-background px-3 py-1 text-xs text-muted-foreground">
          <span class="size-1.5 rounded-full bg-primary" /> {{ content.hero.badge }}
        </div>
        <h1 class="mt-6 text-4xl font-bold leading-[1.12] tracking-tight sm:text-5xl lg:text-6xl">
          {{ content.hero.titleLead }}<br /><span class="text-primary">{{ content.hero.titleAccent }}</span>
        </h1>
        <p class="mt-6 max-w-xl text-base leading-relaxed text-muted-foreground lg:text-lg">
          {{ content.hero.subtitle }}
        </p>
        <div class="mt-9 flex flex-wrap gap-3">
          <Button size="lg" @click="router.push('/m/reg')">{{ content.hero.ctaPrimary }} <ArrowRight class="size-4" /></Button>
          <Button variant="outline" size="lg" @click="router.push('/docs')">{{ content.hero.ctaSecondary }}</Button>
        </div>
        <div class="mt-8">
          <div class="text-xs text-muted-foreground">{{ content.hero.payMethodsLabel }}</div>
          <div class="mt-3 flex flex-wrap items-center gap-2.5">
            <span
              v-for="p in payMethods"
              :key="p.key"
              class="inline-flex items-center gap-2 rounded-full border border-border bg-background px-3 py-1.5 text-sm text-muted-foreground shadow-sm transition-colors hover:text-foreground"
              :style="{ '--chip': p.color }"
            >
              <svg v-if="p.logo" viewBox="0 0 24 24" class="size-4" :style="{ fill: 'var(--chip)' }" aria-hidden="true"><path :d="p.logo" /></svg>
              <img v-else-if="p.img" :src="p.img" :alt="p.name" class="size-4" />
              <span
                v-else
                class="flex size-4 items-center justify-center rounded-full text-[9px] font-bold text-white"
                :style="{ background: 'var(--chip)' }"
              >闪</span>{{ p.name }}
            </span>
          </div>
        </div>
      </div>

      <!-- 右：收款概览卡片 -->
      <div v-reveal="120" class="relative hidden lg:block">
        <div class="rounded-2xl border border-border bg-background p-6 shadow-sm">
          <div class="flex items-center justify-between border-b border-border/60 pb-4">
            <span class="text-sm font-medium">今日收款概览</span>
            <Badge variant="success">实时</Badge>
          </div>
          <div class="mt-4">
            <div class="text-xs text-muted-foreground">交易总额</div>
            <div class="mt-1 text-3xl font-semibold tabular-nums">¥ 328,650.00</div>
          </div>
          <div class="mt-5 grid grid-cols-3 gap-3">
            <div v-for="s in heroShares" :key="s.name" class="rounded-lg bg-content p-3 text-center">
              <div class="text-xs text-muted-foreground">{{ s.name }}</div>
              <div class="mt-1 text-sm font-semibold tabular-nums">{{ s.share }}</div>
            </div>
          </div>
          <!-- 迷你柱状图 -->
          <div class="mt-5 flex h-24 items-end gap-1.5">
            <div v-for="(h, i) in bars" :key="i" class="flex-1 rounded-t bg-gradient-to-t from-primary/30 to-primary" :style="{ height: h + '%' }" />
          </div>
        </div>
        <!-- 悬浮小卡：成功率 -->
        <div class="absolute -bottom-5 -left-5 rounded-xl border border-border bg-background px-4 py-3 shadow-md">
          <div class="text-xs text-muted-foreground">支付成功率</div>
          <div class="mt-0.5 text-xl font-bold tabular-nums text-primary">99.9%</div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
/* 品牌色柔光晕（径向渐变球，模糊扩散） */
.hero-glow {
  border-radius: 9999px;
  background: radial-gradient(
    circle,
    color-mix(in oklch, var(--primary) 22%, transparent) 0%,
    transparent 70%
  );
  filter: blur(56px);
}

/* ===== 幻灯片背景轮播 ===== */
.slide-bg {
  opacity: 0;
  transition: opacity 0.8s ease;
}
.slide-bg.active {
  opacity: 1;
}
.dot {
  width: 9px;
  height: 9px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.5);
  transition: background 0.2s, width 0.2s;
}
.dot.active {
  width: 22px;
  background: #fff;
}
</style>
