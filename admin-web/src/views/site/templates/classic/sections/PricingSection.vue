<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { SiteContent } from '@/lib/mock/site-content'
import { payLogo } from '../shared'

const props = defineProps<{ content: SiteContent; preview?: boolean }>()
const router = useRouter()

// 费率方案：可见方案由 CMS content.plans 提供（过滤 hidden）
const visiblePlans = computed(() => props.content.plans.filter((p) => !p.hidden))
</script>

<template>
  <!-- ===== 费率方案（合规：费率透明披露，紧随核心特性靠前展示）===== -->
  <section id="pricing" class="scroll-mt-20 bg-background px-4 py-24 lg:px-6">
    <div v-reveal class="mx-auto max-w-2xl text-center">
      <h2 class="text-3xl font-bold tracking-tight">{{ content.pricingTitle }}</h2>
      <p class="mt-3 text-muted-foreground">{{ content.pricingSubtitle }}</p>
    </div>
    <div class="mx-auto mt-14 grid max-w-4xl items-stretch gap-6 sm:grid-cols-2">
      <div
        v-for="p in visiblePlans"
        :key="p.name"
        v-reveal
        class="relative flex flex-col overflow-hidden bg-background shadow-[0_10px_30px_-14px_rgba(0,0,0,0.15)] ring-1 ring-border"
      >
        <!-- 头部：普通卡中性灰渐变底；微信卡微信绿渐变底 + 白字 + 绿装饰层 -->
        <div
          class="relative overflow-hidden px-8 pb-8 pt-11 text-center"
          :class="p.highlight
            ? 'bg-gradient-to-br from-[#2bc24a] to-[#1AAD19]'
            : 'bg-gradient-to-br from-[#f4f5f7] to-[#e9ebef]'"
        >
          <!-- 装饰层：微信卡=白斜线纹+白光斑（绿底）；普通卡=灰斜线纹+中性柔光晕（同构对称，颜色克制）-->
          <template v-if="p.highlight">
            <div class="lines-white pointer-events-none absolute inset-0 opacity-60" aria-hidden="true" />
            <div class="plan-glow pointer-events-none absolute -right-16 -top-20 size-48 bg-[radial-gradient(circle,rgba(255,255,255,0.28)_0%,transparent_70%)]" aria-hidden="true" />
          </template>
          <template v-else>
            <div class="lines-gray pointer-events-none absolute inset-0" aria-hidden="true" />
            <div class="plan-glow plan-glow-gray pointer-events-none absolute -right-16 -top-20 size-48" aria-hidden="true" />
          </template>

          <div v-if="p.highlight" class="absolute right-6 top-6 z-10">
            <span class="inline-flex items-center rounded bg-white/25 px-2 text-xs font-medium leading-5 text-white backdrop-blur-sm">推荐</span>
          </div>
          <div class="relative z-10">
            <div class="text-lg font-bold" :class="p.highlight ? 'text-white' : 'text-foreground'">{{ p.name }}</div>
            <div class="mt-1 text-xs" :class="p.highlight ? 'text-white/85' : 'text-muted-foreground'">{{ p.desc }}</div>
            <div class="mt-6 flex items-end justify-center gap-2" :class="p.highlight ? 'text-white' : 'text-foreground'">
              <span class="pb-2 text-right text-xs leading-tight" :class="p.highlight ? 'text-white/75' : 'text-muted-foreground'">费率<br />低至</span>
              <span class="text-6xl font-extrabold leading-none tracking-tight tabular-nums">{{ p.price }}</span>
              <span class="pb-1 text-2xl font-bold">{{ p.unit }}</span>
            </div>
          </div>
        </div>
        <!-- 权益容器：分渠道费率 + 字段明细 + 按钮（两卡头部均有底色，与白色权益区自然分隔）-->
        <div class="flex flex-1 flex-col px-8 pb-11 pt-8">
          <!-- 分渠道手续费：单渠道整行；多渠道且费率一致时并排展示 -->
          <div class="space-y-2.5">
            <!-- 多渠道：一行内两渠道分列撑开，中间竖线分隔 -->
            <div
              v-if="p.rates.length > 1"
              class="flex items-center bg-content px-5 py-3"
            >
              <template v-for="(r, ri) in p.rates" :key="r.chan">
                <span v-if="ri > 0" class="mx-1 h-5 w-px bg-border" />
                <span class="flex flex-1 items-center gap-2" :class="ri > 0 ? 'justify-end' : ''">
                  <svg viewBox="0 0 24 24" class="size-5 shrink-0" :style="{ fill: payLogo(r.chan).color }" aria-hidden="true"><path :d="payLogo(r.chan).logo" /></svg>
                  <span class="text-sm text-muted-foreground">{{ r.name }}</span>
                  <span class="text-base font-bold tabular-nums" :style="{ color: payLogo(r.chan).color }">{{ r.rate }}</span>
                </span>
              </template>
            </div>
            <!-- 否则逐渠道一行 -->
            <template v-else>
              <div
                v-for="r in p.rates"
                :key="r.chan"
                class="flex items-center gap-3 bg-content px-4 py-3"
              >
                <svg viewBox="0 0 24 24" class="size-5 shrink-0" :style="{ fill: payLogo(r.chan).color }" aria-hidden="true"><path :d="payLogo(r.chan).logo" /></svg>
                <span class="text-sm text-muted-foreground">{{ r.name }}</span>
                <span class="ml-auto text-lg font-bold tabular-nums" :style="{ color: payLogo(r.chan).color }">{{ r.rate }}</span>
              </div>
            </template>
          </div>
          <!-- 字段明细：标题后直接跟内容 -->
          <ul class="mb-10 mt-7 space-y-5 text-sm leading-relaxed">
            <li v-for="ft in p.features" :key="ft.k" class="break-all">
              <span class="text-muted-foreground">{{ ft.k }}：</span><span class="font-medium text-foreground">{{ ft.v }}</span>
            </li>
          </ul>
          <button
            class="mt-auto w-full py-2.5 text-sm font-medium transition-colors"
            :class="p.highlight
              ? 'bg-[#1AAD19] text-white hover:bg-[#159412]'
              : 'bg-foreground text-background hover:bg-foreground/85'"
            @click="router.push('/m/reg')"
          >{{ p.cta }}</button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
/* 费率卡角落柔光晕（与 hero-glow 同款：径向渐变球 + 模糊扩散）*/
.plan-glow {
  border-radius: 9999px;
  filter: blur(40px);
}
.plan-glow-gray {
  background: radial-gradient(
    circle,
    color-mix(in oklch, var(--foreground) 6%, transparent) 0%,
    transparent 70%
  );
}

/* 普通卡：灰色斜线纹（与微信卡 .lines-white 同几何，中性色，克制）*/
.lines-gray {
  background-image: repeating-linear-gradient(
    45deg,
    color-mix(in oklch, var(--foreground) 5%, transparent) 0,
    color-mix(in oklch, var(--foreground) 5%, transparent) 1px,
    transparent 1px,
    transparent 11px
  );
  -webkit-mask-image: radial-gradient(120% 90% at 75% 0%, #000 30%, transparent 78%);
  mask-image: radial-gradient(120% 90% at 75% 0%, #000 30%, transparent 78%);
}
</style>
