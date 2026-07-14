<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  Layers, Percent, Zap, ShieldCheck, Code2, MonitorSmartphone,
  ArrowRight, Check, QrCode, MessageCircle, Smartphone, AppWindow, Network, Building2,
} from 'lucide-vue-next'
import { Button, Badge } from '@/components/ui'

const router = useRouter()

// 数据背书（数字滚动）
const metrics = [
  { target: 50000, suffix: '+', label: '接入商户', display: (n: number) => Math.round(n).toLocaleString() },
  { target: 12, suffix: '亿+', prefix: '¥', label: '日均交易额', display: (n: number) => Math.round(n).toString() },
  { target: 99.9, suffix: '%', label: '支付成功率', display: (n: number) => n.toFixed(1) },
  { target: 8, suffix: 'ms', label: '平均响应', display: (n: number) => Math.round(n).toString() },
]
const counts = ref(metrics.map(() => 0))
const metricsEl = ref<HTMLElement | null>(null)
onMounted(() => {
  if (!metricsEl.value) return
  const io = new IntersectionObserver((es) => {
    for (const e of es) {
      if (e.isIntersecting) {
        const dur = 1200
        const start = performance.now()
        const step = (now: number) => {
          const p = Math.min(1, (now - start) / dur)
          const eased = 1 - Math.pow(1 - p, 3)
          counts.value = metrics.map((m) => m.target * eased)
          if (p < 1) requestAnimationFrame(step)
          else counts.value = metrics.map((m) => m.target)
        }
        requestAnimationFrame(step)
        io.disconnect()
      }
    }
  }, { threshold: 0.4 })
  io.observe(metricsEl.value)
})

// 核心特性（6 个，统一卡片）
const features = [
  { icon: Layers, title: '多渠道聚合', desc: '一次对接，支持支付宝、微信、QQ钱包、云闪付等主流支付方式。' },
  { icon: Percent, title: '超低费率', desc: '费率低至 0.28%，会员等级越高费率越低，成本清晰可控。' },
  { icon: Zap, title: '实时到账', desc: '买家付款秒级入账，T+1 自动结算，资金周转更快。' },
  { icon: ShieldCheck, title: '安全风控', desc: '多维度风控引擎，实时拦截异常交易，保障资金安全。' },
  { icon: Code2, title: '开放 API', desc: 'RESTful 接口 + MD5/RSA 双签名，文档完善，最快 1 天对接。' },
  { icon: MonitorSmartphone, title: '多端管理', desc: '商户中心、运营后台、SaaS 控制台，PC 与移动端随时管理。' },
]

// 产品矩阵
const products = [
  { icon: QrCode, name: '扫码支付', desc: '线下贴码 / 主扫被扫' },
  { icon: MessageCircle, name: '公众号支付', desc: '微信内 JSAPI 收款' },
  { icon: AppWindow, name: '小程序支付', desc: '小程序一键下单' },
  { icon: Smartphone, name: 'H5 支付', desc: '手机浏览器唤起' },
  { icon: MonitorSmartphone, name: 'APP 支付', desc: '原生 SDK 收款' },
  { icon: Network, name: '网关支付', desc: 'PC 网页收银台' },
  { icon: Building2, name: '企业收付款', desc: '对公代付批量结算' },
  { icon: Percent, name: '跨境支付', desc: '多币种境外收款' },
]

// 支付渠道
const payMethods = [
  { name: '支付宝', key: 'alipay' },
  { name: '微信支付', key: 'wxpay' },
  { name: 'QQ钱包', key: 'qqpay' },
  { name: '云闪付', key: 'bank' },
]
// Hero 概览卡（固定示例值）
const heroShares = [
  { name: '支付宝', share: '46%' },
  { name: '微信', share: '38%' },
  { name: 'QQ钱包', share: '16%' },
]
const bars = [40, 62, 48, 78, 56, 88, 70, 96, 64, 82]

// 费率方案
const plans = [
  { name: '普通商户', price: '0.38', unit: '%', desc: '注册即用，适合起步', features: ['支付宝/微信/QQ钱包', 'T+1 自动结算', '基础风控', '开放 API 对接'], cta: '免费注册', highlight: false },
  { name: '会员商户', price: '0.30', unit: '%', desc: '付费升级，费率更低', features: ['全部普通商户权益', '更低费率 0.30%', '优先技术支持', '代付 / 分账能力'], cta: '升级会员', highlight: true },
  { name: '企业定制', price: '面议', unit: '', desc: '大额 / 高频专属方案', features: ['专属费率与通道', '独立结算方案', '专属客户经理', 'SLA 服务保障'], cta: '联系我们', highlight: false },
]
</script>

<template>
  <div>
    <!-- ===== Hero（宽屏左右分栏：左文案右视觉）===== -->
    <section class="site-surface relative overflow-hidden border-b border-border">
      <div class="mx-auto grid max-w-7xl items-center gap-12 px-4 py-20 lg:grid-cols-[1.05fr_1fr] lg:px-8 lg:py-28">
        <!-- 左：文案 -->
        <div v-reveal>
          <div class="inline-flex items-center gap-2 rounded-full border border-border bg-background px-3 py-1 text-xs text-muted-foreground">
            <span class="size-1.5 rounded-full bg-primary" /> 聚合支付 · 服务商模式
          </div>
          <h1 class="mt-6 text-4xl font-bold leading-[1.12] tracking-tight sm:text-5xl lg:text-6xl">
            让每一笔收款<br /><span class="text-primary">简单而可靠</span>
          </h1>
          <p class="mt-6 max-w-xl text-base leading-relaxed text-muted-foreground lg:text-lg">
            一次对接，聚合支付宝、微信、QQ钱包、云闪付等多渠道收款。费率低至 0.28%，秒级到账，开放 API，助你快速开启线上收款。
          </p>
          <div class="mt-9 flex flex-wrap gap-3">
            <Button size="lg" @click="router.push('/m/reg')">免费注册 <ArrowRight class="size-4" /></Button>
            <Button variant="outline" size="lg" @click="router.push('/docs')">查看文档</Button>
          </div>
          <div class="mt-8 flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-muted-foreground">
            <span v-for="p in payMethods" :key="p.key" class="flex items-center gap-1.5">
              <img :src="`/assets/icon/${p.key}.ico`" class="size-4" onerror="this.style.display='none'" />{{ p.name }}
            </span>
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

    <!-- ===== 数据背书（数字滚动）===== -->
    <section ref="metricsEl" class="border-b border-border">
      <div class="mx-auto grid max-w-7xl grid-cols-2 divide-x divide-border px-4 py-12 lg:grid-cols-4 lg:px-6">
        <div v-for="(m, i) in metrics" :key="m.label" class="px-4 text-center">
          <div class="text-3xl font-bold tabular-nums sm:text-4xl">
            <span v-if="m.prefix" class="text-2xl text-muted-foreground">{{ m.prefix }}</span>{{ m.display(counts[i]) }}<span class="text-xl text-muted-foreground">{{ m.suffix }}</span>
          </div>
          <div class="mt-1.5 text-sm text-muted-foreground">{{ m.label }}</div>
        </div>
      </div>
    </section>

    <!-- ===== 核心特性 ===== -->
    <section id="features" class="mx-auto max-w-7xl scroll-mt-20 px-4 py-24 lg:px-6">
      <div v-reveal class="mx-auto max-w-2xl text-center">
        <h2 class="text-3xl font-bold tracking-tight">为什么选择 0538Pay</h2>
        <p class="mt-3 text-muted-foreground">一站式聚合支付解决方案，让收款更简单</p>
      </div>
      <div v-reveal class="mt-14 grid gap-px overflow-hidden rounded-2xl border border-border bg-border sm:grid-cols-2 lg:grid-cols-3">
        <div v-for="f in features" :key="f.title" class="bg-background p-7">
          <div class="flex size-10 items-center justify-center rounded-lg bg-primary/[0.08] text-primary">
            <component :is="f.icon" class="size-5" />
          </div>
          <h3 class="mt-4 text-base font-semibold">{{ f.title }}</h3>
          <p class="mt-2 text-sm leading-relaxed text-muted-foreground">{{ f.desc }}</p>
        </div>
      </div>
    </section>

    <!-- ===== 产品矩阵 ===== -->
    <section id="products" class="site-surface scroll-mt-20 border-y border-border">
      <div class="mx-auto max-w-7xl px-4 py-24 lg:px-6">
        <div v-reveal class="mx-auto max-w-2xl text-center">
          <h2 class="text-3xl font-bold tracking-tight">全场景产品矩阵</h2>
          <p class="mt-3 text-muted-foreground">从线上到线下，覆盖你的每个收款场景</p>
        </div>
        <div v-reveal class="mt-14 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div v-for="p in products" :key="p.name" class="rounded-xl border border-border bg-background p-5">
            <div class="flex size-10 items-center justify-center rounded-lg bg-primary/[0.08] text-primary">
              <component :is="p.icon" class="size-5" />
            </div>
            <div class="mt-3 text-sm font-semibold">{{ p.name }}</div>
            <div class="mt-1 text-xs text-muted-foreground">{{ p.desc }}</div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== 费率方案 ===== -->
    <section id="pricing" class="mx-auto max-w-7xl scroll-mt-20 px-4 py-24 lg:px-6">
      <div v-reveal class="mx-auto max-w-2xl text-center">
        <h2 class="text-3xl font-bold tracking-tight">透明的费率方案</h2>
        <p class="mt-3 text-muted-foreground">按需选择，费率清晰，无隐藏费用</p>
      </div>
      <div class="mx-auto mt-14 grid max-w-5xl gap-6 lg:grid-cols-3">
        <div
          v-for="p in plans"
          :key="p.name"
          v-reveal
          class="rounded-2xl border bg-background p-7"
          :class="p.highlight ? 'border-primary ring-1 ring-primary' : 'border-border'"
        >
          <div class="flex items-center gap-2">
            <span class="text-lg font-semibold">{{ p.name }}</span>
            <Badge v-if="p.highlight" variant="default">推荐</Badge>
          </div>
          <div class="mt-1 text-sm text-muted-foreground">{{ p.desc }}</div>
          <div class="mt-5 flex items-baseline gap-1">
            <span class="text-4xl font-bold tabular-nums">{{ p.price }}</span>
            <span class="text-lg text-muted-foreground">{{ p.unit }}</span>
          </div>
          <ul class="mt-6 space-y-3">
            <li v-for="ft in p.features" :key="ft" class="flex items-center gap-2 text-sm">
              <Check class="size-4 shrink-0 text-primary" />
              <span class="text-muted-foreground">{{ ft }}</span>
            </li>
          </ul>
          <Button class="mt-7 w-full" :variant="p.highlight ? 'default' : 'outline'" @click="router.push('/m/reg')">{{ p.cta }}</Button>
        </div>
      </div>
    </section>

    <!-- ===== CTA ===== -->
    <section class="border-t border-border">
      <div class="mx-auto max-w-7xl px-4 py-24 text-center lg:px-6">
        <div v-reveal class="mx-auto max-w-2xl">
          <h2 class="text-3xl font-bold tracking-tight sm:text-4xl">立即开启你的收款业务</h2>
          <p class="mx-auto mt-4 max-w-lg text-muted-foreground">注册即可免费接入，多渠道聚合、超低费率、实时到账，助你的生意收款无忧。</p>
          <div class="mt-8 flex flex-wrap justify-center gap-3">
            <Button size="lg" @click="router.push('/m/reg')">免费注册 <ArrowRight class="size-4" /></Button>
            <Button variant="outline" size="lg" @click="router.push('/docs')">查看接入文档</Button>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
