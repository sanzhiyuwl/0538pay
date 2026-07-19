<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ArrowRight } from 'lucide-vue-next'
import { Button, Badge } from '@/components/ui'
import type { SiteContent } from '@/lib/mock/site-content'
import { payMethods, heroShares, bars } from '../shared'

defineProps<{ content: SiteContent; preview?: boolean }>()
const router = useRouter()
</script>

<template>
  <!-- ===== Hero（宽屏左右分栏：左文案右视觉）===== -->
  <section class="relative -mt-16 overflow-hidden border-b border-border bg-background">
    <!-- 背景装饰层：斜向流光带 + 精细点阵 + 品牌色柔光晕（延伸至导航背后）-->
    <div class="pointer-events-none absolute inset-0 z-0" aria-hidden="true">
      <div class="hero-beam absolute inset-0" />
      <div class="hero-dots absolute inset-x-0 top-0 h-2/3" />
      <div class="hero-glow absolute -right-28 -top-44 size-[44rem] opacity-90" />
      <div class="hero-glow absolute -left-44 top-16 size-[34rem] opacity-55" />
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
</style>
