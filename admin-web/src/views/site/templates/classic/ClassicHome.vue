<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  Waypoints, BadgePercent, Gauge, Radar, Webhook, MonitorSmartphone,
  ArrowRight, QrCode, MessageCircle, Smartphone, AppWindow, Network, Building2, Globe,
  Check, ShieldCheck, LayoutDashboard, Store, Plus, Minus, Quote,
} from 'lucide-vue-next'
import type { Component } from 'vue'
import { Button, Badge } from '@/components/ui'
import { useSiteContentStore } from '@/stores/siteContent'
import ProductArt3D from './ProductArt3D.vue'

const router = useRouter()

// 首页营销内容来自后台「官网管理」CMS，实时联动（content 为 reactive）
const content = useSiteContentStore().content

// 图标名 → 组件映射（CMS 存图标名字符串，此处解析为组件）
const iconMap: Record<string, Component> = {
  Waypoints, BadgePercent, Gauge, Radar, Webhook, MonitorSmartphone,
  QrCode, MessageCircle, Smartphone, AppWindow, Network, Building2, Globe,
  Store, ShieldCheck, LayoutDashboard,
}
const icon = (name: string): Component => iconMap[name] ?? Waypoints

// 数据背书（数字滚动）：格式化按 decimals + 千分位
const metrics = computed(() => content.metrics)
function fmtMetric(v: number, decimals: number): string {
  return decimals > 0 ? v.toFixed(decimals) : Math.round(v).toLocaleString()
}
const counts = ref(content.metrics.map(() => 0))
const metricsEl = ref<HTMLElement | null>(null)
onMounted(() => {
  if (!metricsEl.value) return
  const io = new IntersectionObserver((es) => {
    for (const e of es) {
      if (e.isIntersecting) {
        const dur = 1200
        const start = performance.now()
        const targets = content.metrics.map((m) => m.target)
        const step = (now: number) => {
          const p = Math.min(1, (now - start) / dur)
          const eased = 1 - Math.pow(1 - p, 3)
          counts.value = targets.map((t) => t * eased)
          if (p < 1) requestAnimationFrame(step)
          else counts.value = targets
        }
        requestAnimationFrame(step)
        io.disconnect()
      }
    }
  }, { threshold: 0.4 })
  io.observe(metricsEl.value)
})

const activeProduct = ref(0)

const openFaq = ref(0)
function toggleFaq(i: number) {
  openFaq.value = openFaq.value === i ? -1 : i
}

// 支付渠道（官方矢量 logo 路径 + 品牌色，用于 hero 标签）
// logo 取自 simple-icons（24×24 viewBox）；云闪付无现成矢量，用文字标记
const payMethods = [
  { name: '支付宝', key: 'alipay', color: '#1677FF', logo: 'M19.695 15.07c3.426 1.158 4.203 1.22 4.203 1.22V3.846c0-2.124-1.705-3.845-3.81-3.845H3.914C1.808.001.102 1.722.102 3.846v16.31c0 2.123 1.706 3.845 3.813 3.845h16.173c2.105 0 3.81-1.722 3.81-3.845v-.157s-6.19-2.602-9.315-4.119c-2.096 2.602-4.8 4.181-7.607 4.181-4.75 0-6.361-4.19-4.112-6.949.49-.602 1.324-1.175 2.617-1.497 2.025-.502 5.247.313 8.266 1.317a16.796 16.796 0 0 0 1.341-3.302H5.781v-.952h4.799V6.975H4.77v-.953h5.81V3.591s0-.409.411-.409h2.347v2.84h5.744v.951h-5.744v1.704h4.69a19.453 19.453 0 0 1-1.986 5.06c1.424.52 2.702 1.011 3.654 1.333m-13.81-2.032c-.596.06-1.71.325-2.321.869-1.83 1.608-.735 4.55 2.968 4.55 2.151 0 4.301-1.388 5.99-3.61-2.403-1.182-4.438-2.028-6.637-1.809' },
  { name: '微信支付', key: 'wxpay', color: '#07C160', logo: 'M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 0 1 .213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 0 0 .167-.054l1.903-1.114a.864.864 0 0 1 .717-.098 10.16 10.16 0 0 0 2.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348zM5.785 5.991c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 0 1-1.162 1.178A1.17 1.17 0 0 1 4.623 7.17c0-.651.52-1.18 1.162-1.18zm5.813 0c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 0 1-1.162 1.178 1.17 1.17 0 0 1-1.162-1.178c0-.651.52-1.18 1.162-1.18zm5.34 2.867c-1.797-.052-3.746.512-5.28 1.786-1.72 1.428-2.687 3.72-1.78 6.22.942 2.453 3.666 4.229 6.884 4.229.826 0 1.622-.12 2.361-.336a.722.722 0 0 1 .598.082l1.584.926a.272.272 0 0 0 .14.047c.134 0 .24-.111.24-.247 0-.06-.023-.12-.038-.177l-.327-1.233a.582.582 0 0 1-.023-.156.49.49 0 0 1 .201-.398C23.024 18.48 24 16.82 24 14.98c0-3.21-2.931-5.837-6.656-6.088V8.89c-.135-.01-.27-.027-.407-.03zm-2.53 3.274c.535 0 .969.44.969.982a.976.976 0 0 1-.969.983.976.976 0 0 1-.969-.983c0-.542.434-.982.97-.982zm4.844 0c.535 0 .969.44.969.982a.976.976 0 0 1-.969.983.976.976 0 0 1-.969-.983c0-.542.434-.982.969-.982z' },
  { name: 'QQ钱包', key: 'qqpay', color: '#12B7F5', logo: 'M21.395 15.035a40 40 0 0 0-.803-2.264l-1.079-2.695c.001-.032.014-.562.014-.836C19.526 4.632 17.351 0 12 0S4.474 4.632 4.474 9.241c0 .274.013.804.014.836l-1.08 2.695a39 39 0 0 0-.802 2.264c-1.021 3.283-.69 4.643-.438 4.673.54.065 2.103-2.472 2.103-2.472 0 1.469.756 3.387 2.394 4.771-.612.188-1.363.479-1.845.835-.434.32-.379.646-.301.778.343.578 5.883.369 7.482.189 1.6.18 7.14.389 7.483-.189.078-.132.132-.458-.301-.778-.483-.356-1.233-.646-1.846-.836 1.637-1.384 2.393-3.302 2.393-4.771 0 0 1.563 2.537 2.103 2.472.251-.03.581-1.39-.438-4.673' },
  { name: '云闪付', key: 'bank', color: '#E60012', logo: '', img: '/assets/pay/unionpay.svg' },
]
// Hero 概览卡（固定示例值）
const heroShares = [
  { name: '支付宝', share: '46%' },
  { name: '微信', share: '38%' },
  { name: 'QQ钱包', share: '16%' },
]
const bars = [40, 62, 48, 78, 56, 88, 70, 96, 64, 82]

// 费率方案：可见方案由 CMS content.plans 提供（过滤 hidden）
const visiblePlans = computed(() => content.plans.filter((p) => !p.hidden))
// 按渠道 key 取 logo 与品牌色（供费率明细复用 payMethods 的矢量 logo）
const payLogo = (key: string) => payMethods.find((m) => m.key === key) ?? payMethods[0]

// 费率卡配色主题：仅顶部彩色头部使用；权益区为白底统一样式
// head=头部背景，headText/headSub=头部文字，btn=按钮（主题强调色）
const planThemes: Record<string, { head: string; headText: string; headSub: string; btn: string }> = {
  gray: {
    head: 'bg-[#f3f4f6]',
    headText: 'text-slate-800', headSub: 'text-slate-500',
    btn: 'bg-slate-700 text-white hover:bg-slate-800',
  },
  wechat: {
    head: 'bg-gradient-to-br from-[#28c24a] to-[#1AAD19]',
    headText: 'text-white', headSub: 'text-white/85',
    btn: 'bg-[#1AAD19] text-white hover:bg-[#159412]',
  },
  alipay: {
    head: 'bg-gradient-to-b from-[#3ba0ff] to-[#1677ff]',
    headText: 'text-white', headSub: 'text-white/85',
    btn: 'bg-[#1677ff] text-white hover:bg-[#0e63e6]',
  },
}
</script>

<template>
  <div>
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

    <!-- ===== 数据背书（深色背景大图 + 暗色遮罩 + 白色数字滚动）===== -->
    <section ref="metricsEl" class="relative overflow-hidden border-b border-border">
      <!-- 背景图 + 暗色遮罩 -->
      <div class="pointer-events-none absolute inset-0 z-0" aria-hidden="true">
        <img src="/assets/metrics-bg2.jpg" alt="" class="size-full object-cover" />
        <div class="absolute inset-0 bg-black/15" />
      </div>
      <div class="relative z-10 mx-auto grid max-w-7xl grid-cols-2 divide-x divide-white/15 px-4 py-16 lg:grid-cols-4 lg:px-6">
        <div v-for="(m, i) in metrics" :key="m.label" class="px-4 text-center">
          <div class="text-3xl font-bold tabular-nums text-white sm:text-4xl">
            <span v-if="m.prefix" class="text-2xl text-white/70">{{ m.prefix }}</span>{{ fmtMetric(counts[i], m.decimals) }}<span class="text-xl text-white/70">{{ m.suffix }}</span>
          </div>
          <div class="mt-1.5 text-sm text-white/60">{{ m.label }}</div>
        </div>
      </div>
    </section>

    <!-- ===== 核心特性 ===== -->
    <section id="features" class="mx-auto max-w-7xl scroll-mt-20 px-4 py-24 lg:px-6">
      <div v-reveal class="mx-auto max-w-2xl text-center">
        <h2 class="text-3xl font-bold tracking-tight">{{ content.featuresTitle }}</h2>
        <p class="mt-3 text-muted-foreground">{{ content.featuresSubtitle }}</p>
      </div>
      <div v-reveal class="mt-14 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="f in content.features"
          :key="f.title"
          class="flex flex-col items-start bg-background p-7 shadow-[0_1px_3px_rgba(0,0,0,0.04),0_8px_24px_-8px_rgba(0,0,0,0.08)] transition-shadow hover:shadow-[0_4px_12px_rgba(0,0,0,0.06),0_16px_40px_-12px_rgba(0,98,239,0.18)]"
        >
          <!-- 圆形蓝色发光图标 -->
          <span class="feature-orb mb-5 flex size-14 items-center justify-center rounded-full text-white">
            <component :is="icon(f.icon)" class="size-7" :stroke-width="1.75" />
          </span>
          <h3 class="text-base font-semibold">{{ f.title }}</h3>
          <p class="mt-2 text-sm leading-relaxed text-muted-foreground">{{ f.desc }}</p>
        </div>
      </div>
    </section>

    <!-- ===== 费率方案（合规：费率透明披露，紧随核心特性靠前展示）===== -->
    <section id="pricing" class="scroll-mt-20 px-4 py-24 lg:px-6">
      <div v-reveal class="mx-auto max-w-2xl text-center">
        <h2 class="text-3xl font-bold tracking-tight">{{ content.pricingTitle }}</h2>
        <p class="mt-3 text-muted-foreground">{{ content.pricingSubtitle }}</p>
      </div>
      <div class="mx-auto mt-14 grid max-w-4xl items-stretch gap-6 sm:grid-cols-2">
        <div
          v-for="p in visiblePlans"
          :key="p.name"
          v-reveal
          class="relative flex flex-col overflow-hidden bg-background shadow-[0_10px_30px_-10px_rgba(0,0,0,0.18)]"
        >
          <!-- 彩色头部：名称 / 描述 / 价格 -->
          <div class="relative overflow-hidden px-8 py-12 text-center" :class="planThemes[p.theme].head">
            <!-- 装饰层：按主题区分风格 -->
            <div class="pointer-events-none absolute inset-0 overflow-hidden" aria-hidden="true">
              <!-- 普通商户（灰）：中性点阵 + 斜向柔光，不偏向单一渠道 -->
              <template v-if="p.theme === 'gray'">
                <div class="hero-dots absolute inset-0 opacity-60" />
                <div class="absolute -right-16 -top-16 size-44 rounded-full bg-gradient-to-br from-primary/[0.08] to-transparent blur-xl" />
              </template>
              <!-- 微信特约（绿）：斜向细线纹 + 柔光斑（与普通商户点阵区分）-->
              <template v-else>
                <div class="lines-white absolute inset-0 opacity-70" />
                <div class="absolute -right-16 -top-16 size-44 rounded-full bg-gradient-to-br from-white/15 to-transparent blur-xl" />
              </template>
            </div>
            <div v-if="p.highlight" class="absolute left-0 top-0 bg-[#ff5a3c] px-4 py-1 text-xs font-medium text-white">推荐</div>
            <div class="relative">
              <div class="text-lg font-bold" :class="planThemes[p.theme].headText">{{ p.name }}</div>
              <div class="mt-1 text-xs" :class="planThemes[p.theme].headSub">{{ p.desc }}</div>
              <div class="mt-6 flex items-center justify-center gap-2">
                <span class="text-right text-xs leading-tight" :class="planThemes[p.theme].headSub">费率<br />低至</span>
                <span class="text-6xl font-extrabold leading-none tracking-tight tabular-nums" :class="planThemes[p.theme].headText">{{ p.price }}</span>
                <span class="self-end pb-1 text-xl font-bold" :class="planThemes[p.theme].headText">{{ p.unit }}</span>
              </div>
            </div>
          </div>
          <!-- 白色权益容器：分渠道费率 + 字段明细 + 按钮 -->
          <div class="flex flex-1 flex-col px-8 py-12">
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
              :class="planThemes[p.theme].btn"
              @click="router.push('/m/reg')"
            >{{ p.cta }}</button>
          </div>
        </div>
      </div>
    </section>

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
          <p class="mt-4 text-[15px] leading-relaxed text-white/80 drop-shadow-[0_1px_4px_rgba(0,0,0,0.35)]">{{ content.productsSubtitle }}</p>
        </div>

        <!-- 面板（整体深色 #262B39 半透明，透出背景图；左 tab + 中详情 + 右插画）-->
        <div v-reveal class="mt-16 grid overflow-hidden bg-[#262B39]/60 shadow-[0_20px_50px_-20px_rgba(0,0,0,0.5)] backdrop-blur-md lg:grid-cols-[240px_1fr_260px]">
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
          <div class="relative overflow-hidden p-8 lg:p-14">
            <div class="relative z-10">
              <!-- 文案 + 能力项 -->
              <div class="min-w-0">
                <!-- 标题行：图标 chip + 名称 -->
                <div class="flex items-center gap-3.5">
                  <span class="flex size-12 shrink-0 items-center justify-center rounded-xl bg-primary/15 text-[#7db4ff]">
                    <component :is="icon(content.products[activeProduct].icon)" class="size-6" />
                  </span>
                  <h3 class="text-2xl font-bold tracking-tight text-white">{{ content.products[activeProduct].name }}</h3>
                </div>
                <p class="mt-5 max-w-xl text-[15px] leading-relaxed text-white/60">{{ content.products[activeProduct].desc }}</p>

                <!-- 能力项：图标 + 文字，装进浅色小卡（图标顶对齐，文字自然占满、换行整齐）-->
                <div class="mt-7 grid grid-cols-1 gap-2.5 sm:grid-cols-2">
                  <div
                    v-for="pt in content.products[activeProduct].points"
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
                    v-for="tag in content.products[activeProduct].tags"
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
          <div class="relative hidden bg-[#262B39]/45 lg:block">
            <!-- 背景装饰：与 tab 同款淡品牌色渐变（自下而上轻染）-->
            <div class="pointer-events-none absolute inset-0 z-0 bg-gradient-to-t from-primary/15 to-transparent" aria-hidden="true" />
            <div class="relative z-10 flex h-full items-center justify-center p-4">
              <!-- 等距 3D 插画（纯 SVG，统一底座台，切 tab 换图标）-->
              <ProductArt3D :icon="icon(content.products[activeProduct].icon)" />
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== 客户评价 ===== -->
    <section class="bg-background">
      <div class="mx-auto max-w-7xl px-4 py-24 lg:px-6">
        <div v-reveal class="mx-auto max-w-2xl text-center">
          <h2 class="text-3xl font-bold tracking-tight">{{ content.testimonialsTitle }}</h2>
          <p class="mt-3 text-muted-foreground">{{ content.testimonialsSubtitle }}</p>
        </div>
        <div v-reveal class="mt-14 grid gap-6 md:grid-cols-3">
          <div
            v-for="t in content.testimonials"
            :key="t.name"
            class="relative flex flex-col border border-border bg-background p-7"
          >
            <Quote class="size-8 text-primary/15" />
            <p class="mt-4 flex-1 text-sm leading-relaxed text-foreground/90">{{ t.text }}</p>
            <div class="mt-6 flex items-center gap-3 border-t border-border/60 pt-5">
              <div class="flex size-10 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold text-primary">{{ t.avatar }}</div>
              <div>
                <div class="text-sm font-semibold">{{ t.name }}</div>
                <div class="text-xs text-muted-foreground">{{ t.role }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== 常见问题 FAQ（手风琴）===== -->
    <section class="mx-auto max-w-3xl px-4 py-24 lg:px-6">
      <div v-reveal class="mx-auto max-w-2xl text-center">
        <h2 class="text-3xl font-bold tracking-tight">{{ content.faqTitle }}</h2>
        <p class="mt-3 text-muted-foreground">{{ content.faqSubtitle }}</p>
      </div>
      <div v-reveal class="mt-12 divide-y divide-border border-y border-border">
        <div v-for="(f, i) in content.faqs" :key="f.q">
          <button
            type="button"
            class="flex w-full items-center justify-between gap-4 py-5 text-left"
            @click="toggleFaq(i)"
          >
            <span class="text-[15px] font-medium">{{ f.q }}</span>
            <span class="flex size-6 shrink-0 items-center justify-center rounded-full border border-border text-muted-foreground transition-colors" :class="openFaq === i ? 'border-primary/30 bg-primary/[0.06] text-primary' : ''">
              <component :is="openFaq === i ? Minus : Plus" class="size-4" />
            </span>
          </button>
          <div
            class="grid transition-all duration-300 ease-out"
            :class="openFaq === i ? 'grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0'"
          >
            <div class="overflow-hidden">
              <p class="pb-5 pr-10 text-sm leading-relaxed text-muted-foreground">{{ f.a }}</p>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== CTA（背景图 + 暗色遮罩 + 白色文字）===== -->
    <section class="relative overflow-hidden">
      <!-- 背景图 + 遮罩 -->
      <div class="pointer-events-none absolute inset-0 z-0" aria-hidden="true">
        <img src="/assets/cta-bg.jpg" alt="" class="size-full object-cover" />
        <div class="absolute inset-0 bg-gradient-to-r from-[#0b1b3a]/90 to-[#0b1b3a]/70" />
      </div>
      <div class="relative z-10 mx-auto max-w-7xl px-4 py-24 text-center lg:px-6">
        <div v-reveal class="mx-auto max-w-2xl">
          <h2 class="text-3xl font-bold tracking-tight text-white sm:text-4xl">{{ content.cta.title }}</h2>
          <p class="mx-auto mt-4 max-w-lg text-white/75">{{ content.cta.subtitle }}</p>
          <div class="mt-8 flex flex-wrap justify-center gap-3">
            <Button size="lg" @click="router.push('/m/reg')">{{ content.cta.ctaPrimary }} <ArrowRight class="size-4" /></Button>
            <Button variant="outline" size="lg" class="border-white/40 bg-transparent text-white hover:bg-white/10 hover:text-white" @click="router.push('/docs')">{{ content.cta.ctaSecondary }}</Button>
          </div>
        </div>
      </div>
    </section>
  </div>
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

/* 核心特性：圆形蓝色发光图标（品牌色渐变 + 外发光 + 顶部高光） */
.feature-orb {
  background: linear-gradient(150deg, #4a90ff 0%, #0062ef 60%, #0048b5 100%);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.4),
    0 10px 24px -8px rgba(0, 98, 239, 0.55);
}
</style>
