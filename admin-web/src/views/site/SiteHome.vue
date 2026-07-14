<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  Layers, Percent, Zap, ShieldCheck, Code2, MonitorSmartphone,
  ArrowRight, Check, UserPlus, BadgeCheck, KeyRound, Plug, Rocket,
  QrCode, MessageCircle, Smartphone, AppWindow, Network, Building2, Ship,
  ShoppingBag, LockKeyhole, BarChart3, Quote, Plus, Minus,
} from 'lucide-vue-next'
import { Button, Badge } from '@/components/ui'

const router = useRouter()

// 数据背书（带数字滚动）
const metrics = [
  { target: 50000, suffix: '+', label: '接入商户', display: (n: number) => n.toLocaleString() },
  { target: 12, suffix: '亿+', prefix: '¥', label: '日均交易额', display: (n: number) => String(n) },
  { target: 99.9, suffix: '%', label: '支付成功率', display: (n: number) => n.toFixed(1) },
  { target: 8, suffix: 'ms', label: '平均响应', display: (n: number) => String(n) },
]
const counts = ref(metrics.map(() => 0))
function runCountUp() {
  const dur = 1400
  const start = performance.now()
  function tick(now: number) {
    const p = Math.min(1, (now - start) / dur)
    const eased = 1 - Math.pow(1 - p, 3)
    counts.value = metrics.map((m) => m.target * eased)
    if (p < 1) requestAnimationFrame(tick)
    else counts.value = metrics.map((m) => m.target)
  }
  requestAnimationFrame(tick)
}
const metricsSeen = ref(false)
const metricsEl = ref<HTMLElement | null>(null)
onMounted(() => {
  if (!metricsEl.value) return
  const io = new IntersectionObserver((es) => {
    for (const e of es) {
      if (e.isIntersecting && !metricsSeen.value) {
        metricsSeen.value = true
        runCountUp()
        io.disconnect()
      }
    }
  }, { threshold: 0.4 })
  io.observe(metricsEl.value)
})

// 核心特性（Bento：第 1、6 项跨列做大卡）
const features = [
  { icon: Layers, title: '多渠道聚合', desc: '一次对接，支持支付宝、微信、QQ钱包、云闪付等主流支付方式，告别逐家接入的繁琐。', big: true },
  { icon: Percent, title: '超低费率', desc: '费率低至 0.28%，等级越高越低。' },
  { icon: Zap, title: '实时到账', desc: '秒级入账，T+1 自动结算。' },
  { icon: ShieldCheck, title: '安全风控', desc: '多维风控引擎，实时拦截异常交易。' },
  { icon: Code2, title: '开放 API', desc: 'RESTful + MD5/RSA 双签名。' },
  { icon: MonitorSmartphone, title: '多端管理', desc: '商户中心、运营后台、SaaS 控制台三端协同，PC 与移动端随时随地管理你的收款业务。', big: true },
]

const payMethods = [
  { name: '支付宝', key: 'alipay' },
  { name: '微信支付', key: 'wxpay' },
  { name: 'QQ钱包', key: 'qqpay' },
  { name: '云闪付', key: 'bank' },
]
const heroShares = [
  { name: '支付宝', share: '46%' },
  { name: '微信', share: '38%' },
  { name: 'QQ钱包', share: '16%' },
]

// 费率方案
const plans = [
  { name: '普通商户', price: '0.38', unit: '%', desc: '注册即用，适合起步', features: ['支付宝/微信/QQ钱包', 'T+1 自动结算', '基础风控', '开放 API 对接'], cta: '免费注册', highlight: false },
  { name: '会员商户', price: '0.30', unit: '%', desc: '付费升级，费率更低', features: ['全部普通商户权益', '更低费率 0.30%', '优先技术支持', '代付/分账能力'], cta: '升级会员', highlight: true },
  { name: '企业定制', price: '面议', unit: '', desc: '大额/高频专属方案', features: ['专属费率与通道', '独立结算方案', '专属客户经理', 'SLA 服务保障'], cta: '联系我们', highlight: false },
]

const steps = [
  { icon: UserPlus, title: '注册账户', desc: '手机/邮箱快速注册商户' },
  { icon: BadgeCheck, title: '实名认证', desc: '完成个人或企业实名' },
  { icon: KeyRound, title: '获取密钥', desc: '拿到商户 ID 与 API 密钥' },
  { icon: Plug, title: '对接 API', desc: '按文档接入下单与回调' },
  { icon: Rocket, title: '上线收款', desc: '正式开始收款与结算' },
]

const products = [
  { icon: QrCode, name: '扫码支付', desc: '线下贴码 / 主扫被扫收款' },
  { icon: MessageCircle, name: '公众号支付', desc: '微信内 JSAPI 收款' },
  { icon: AppWindow, name: '小程序支付', desc: '小程序内一键下单' },
  { icon: Smartphone, name: 'H5 支付', desc: '手机浏览器唤起支付' },
  { icon: MonitorSmartphone, name: 'APP 支付', desc: '原生 App SDK 收款' },
  { icon: Network, name: '网关支付', desc: 'PC 网页收银台' },
  { icon: Building2, name: '企业收付款', desc: '对公代付与批量结算' },
  { icon: Ship, name: '跨境支付', desc: '多币种境外收款' },
]

const scenes = [
  { icon: ShoppingBag, tag: '多场景收款', title: '一套接口，覆盖全部收款场景', desc: '无论线下扫码、公众号、小程序、H5 还是 PC 网关，只需对接一次即可支持全渠道收款，无需为每个场景单独开发。', points: ['聚合支付宝/微信/QQ钱包/云闪付', '收银台自适应 PC 与移动端', '支持合单、分账、代付'] },
  { icon: LockKeyhole, tag: '资金安全', title: '多重风控，资金到账更放心', desc: '实时风控引擎监控每笔交易，异常拦截、黑名单、限额策略多管齐下；款项实时入账，结算记录清晰可查。', points: ['MD5 / RSA 双签名防篡改', '异步通知重试保障', '风控拦截与黑名单机制'], reverse: true },
  { icon: BarChart3, tag: '数据看板', title: '经营数据，一屏尽览', desc: '商户中心提供实时收款概览、通道费率、结算趋势、资金明细等数据看板，助你随时掌握经营状况。', points: ['实时收款与结算概览', '多维度交易统计', '资金流水明细可追溯'] },
]

const testimonials = [
  { name: '陈先生', role: '电商平台负责人', text: '接入很顺畅，文档清晰，一天就跑通了。多渠道聚合让我们省去了对接多家的麻烦。' },
  { name: '李女士', role: '连锁便利店', text: '费率透明，到账很快，结算记录一目了然。客服响应也及时，用得很安心。' },
  { name: '王先生', role: '独立开发者', text: '作为个人开发者最看重接入成本。0538Pay 的 API 设计简洁，签名规则清楚，体验很好。' },
]

const faqs = [
  { q: '接入需要什么资质？', a: '个人和企业均可接入。个人完成实名认证即可，企业需提供营业执照完成企业认证。' },
  { q: '费率是多少？如何结算？', a: '普通商户费率 0.38% 起，会员可低至 0.28%。支持 T+1 自动结算与手动提现两种方式。' },
  { q: '多久能完成对接？', a: '提供完善的 API 文档与示例代码，一般开发者最快 1 天即可完成对接并上线。' },
  { q: '支持哪些支付方式？', a: '支持支付宝、微信支付、QQ钱包、云闪付等主流渠道，覆盖扫码/公众号/小程序/H5/APP/网关等场景。' },
  { q: '资金安全如何保障？', a: '采用 MD5/RSA 双签名机制，配合实时风控引擎、异步通知重试与黑名单策略，多重保障资金与账户安全。' },
  { q: '如何获取技术支持？', a: '提供在线文档、客服 QQ 与工单支持，会员及企业客户享有优先技术支持与专属客户经理。' },
]
const openFaq = ref<number>(0)
function toggleFaq(i: number) {
  openFaq.value = openFaq.value === i ? -1 : i
}
</script>

<template>
  <div>
    <!-- ===== Hero（深色科技风）===== -->
    <section class="relative overflow-hidden bg-[oklch(0.16_0.03_264)] text-white">
      <!-- 网格 + 光晕 -->
      <div class="pointer-events-none absolute inset-0 tech-grid opacity-[0.5]" />
      <div class="pointer-events-none absolute inset-0">
        <div class="absolute -top-24 left-1/2 h-[460px] w-[900px] -translate-x-1/2 rounded-full bg-primary/25 blur-[120px]" />
        <div class="absolute bottom-0 right-10 h-[280px] w-[420px] rounded-full bg-[oklch(0.6_0.2_190)]/20 blur-[100px]" />
      </div>
      <!-- 顶部渐隐到内容底 -->
      <div class="pointer-events-none absolute inset-x-0 bottom-0 h-24 bg-gradient-to-b from-transparent to-background" />

      <div class="relative mx-auto grid max-w-7xl items-center gap-12 px-4 py-24 lg:grid-cols-2 lg:px-6 lg:py-32">
        <div v-reveal>
          <div class="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/5 px-3 py-1 text-xs text-white/80 backdrop-blur">
            <span class="size-1.5 rounded-full bg-[oklch(0.72_0.18_150)]" /> 聚合支付 · 服务商模式
          </div>
          <h1 class="mt-5 text-4xl font-bold leading-[1.15] tracking-tight sm:text-5xl lg:text-6xl">
            让收款<span class="bg-gradient-to-r from-[oklch(0.75_0.16_230)] to-[oklch(0.7_0.19_280)] bg-clip-text text-transparent">前所未有</span>地简单
          </h1>
          <p class="mt-5 max-w-md text-base leading-relaxed text-white/65">
            一次对接，聚合支付宝、微信、QQ钱包、云闪付等多渠道收款。费率低至 0.28%，秒级到账，开放 API，助你快速开启线上收款。
          </p>
          <div class="mt-8 flex flex-wrap gap-3">
            <Button size="lg" @click="router.push('/m/reg')">免费注册 <ArrowRight class="size-4" /></Button>
            <Button variant="outline" size="lg" class="border-white/20 bg-white/5 text-white hover:bg-white/10 hover:text-white" @click="router.push('/docs')">查看文档</Button>
          </div>
        </div>

        <!-- 浏览器窗口 mockup -->
        <div v-reveal="120" class="relative hidden lg:block">
          <div class="overflow-hidden rounded-xl border border-white/10 bg-[oklch(0.2_0.03_264)] shadow-2xl ring-1 ring-white/5">
            <!-- 窗口条 -->
            <div class="flex items-center gap-1.5 border-b border-white/10 px-4 py-3">
              <span class="size-2.5 rounded-full bg-[#ff5f57]" />
              <span class="size-2.5 rounded-full bg-[#febc2e]" />
              <span class="size-2.5 rounded-full bg-[#28c840]" />
              <span class="ml-3 rounded bg-white/5 px-2 py-0.5 text-[11px] text-white/40">merchant.0538pay.com</span>
            </div>
            <div class="p-5">
              <div class="flex items-center justify-between">
                <span class="text-sm text-white/60">今日交易额</span>
                <Badge variant="success">实时</Badge>
              </div>
              <div class="mt-2 text-3xl font-semibold tabular-nums">¥ 328,650.00</div>
              <div class="mt-5 grid grid-cols-3 gap-3">
                <div v-for="m in heroShares" :key="m.name" class="rounded-lg border border-white/10 bg-white/5 p-3 text-center">
                  <div class="text-xs text-white/50">{{ m.name }}</div>
                  <div class="mt-1 text-sm font-semibold tabular-nums text-white/90">{{ m.share }}</div>
                </div>
              </div>
              <!-- 迷你柱状 -->
              <div class="mt-4 flex h-24 items-end gap-1.5">
                <div v-for="(h, i) in [40, 62, 48, 78, 56, 88, 70, 96, 64, 82]" :key="i" class="flex-1 rounded-t bg-gradient-to-t from-primary/40 to-primary" :style="{ height: h + '%' }" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== 数据背书（数字滚动）===== -->
    <section ref="metricsEl" class="border-b border-border bg-content">
      <div class="mx-auto grid max-w-7xl grid-cols-2 gap-6 px-4 py-12 lg:grid-cols-4 lg:px-6">
        <div v-for="(m, i) in metrics" :key="m.label" class="text-center">
          <div class="text-3xl font-bold tabular-nums text-primary sm:text-4xl">
            <span v-if="m.prefix" class="text-2xl">{{ m.prefix }}</span>{{ m.display(counts[i]) }}<span class="text-xl">{{ m.suffix }}</span>
          </div>
          <div class="mt-1.5 text-sm text-muted-foreground">{{ m.label }}</div>
        </div>
      </div>
    </section>

    <!-- ===== 核心特性（Bento 错落）===== -->
    <section id="features" class="mx-auto max-w-7xl scroll-mt-20 px-4 py-24 lg:px-6">
      <div v-reveal class="text-center">
        <h2 class="text-3xl font-bold tracking-tight sm:text-4xl">为什么选择 0538Pay</h2>
        <p class="mt-3 text-muted-foreground">一站式聚合支付解决方案，让收款更简单</p>
      </div>
      <div class="mt-14 grid auto-rows-fr gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="(f, i) in features"
          :key="f.title"
          v-reveal="i * 70"
          class="group relative overflow-hidden rounded-2xl border border-border bg-card p-6 transition-all duration-300 hover:-translate-y-1 hover:border-primary/40 hover:shadow-[0_12px_40px_-12px_oklch(0.58_0.2_262_/_0.35)]"
          :class="f.big ? 'sm:col-span-2 lg:col-span-2' : ''"
        >
          <div class="pointer-events-none absolute -right-8 -top-8 size-28 rounded-full bg-primary/[0.06] blur-2xl transition-opacity duration-300 group-hover:opacity-100" />
          <div class="relative flex size-11 items-center justify-center rounded-xl bg-primary/[0.08] text-primary transition-colors duration-300 group-hover:bg-primary group-hover:text-primary-foreground">
            <component :is="f.icon" class="size-5" />
          </div>
          <h3 class="relative mt-4 text-base font-semibold">{{ f.title }}</h3>
          <p class="relative mt-2 text-sm leading-relaxed text-muted-foreground">{{ f.desc }}</p>
        </div>
      </div>
    </section>

    <!-- ===== 产品矩阵 ===== -->
    <section id="products" class="relative border-y border-border bg-content">
      <div class="pointer-events-none absolute inset-0 tech-dots opacity-40" />
      <div class="relative mx-auto max-w-7xl scroll-mt-20 px-4 py-24 lg:px-6">
        <div v-reveal class="text-center">
          <h2 class="text-3xl font-bold tracking-tight sm:text-4xl">全场景产品矩阵</h2>
          <p class="mt-3 text-muted-foreground">从线上到线下，一站式覆盖你的每个收款场景</p>
        </div>
        <div class="mt-14 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div
            v-for="(p, i) in products"
            :key="p.name"
            v-reveal="i * 50"
            class="group flex items-start gap-3 rounded-xl border border-border bg-card p-5 transition-all duration-300 hover:-translate-y-1 hover:border-primary/40"
          >
            <div class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/[0.08] text-primary transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
              <component :is="p.icon" class="size-5" />
            </div>
            <div>
              <div class="text-sm font-semibold">{{ p.name }}</div>
              <div class="mt-1 text-xs leading-relaxed text-muted-foreground">{{ p.desc }}</div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== 支付方式墙 ===== -->
    <section class="mx-auto max-w-7xl px-4 py-16 lg:px-6">
      <div v-reveal class="text-center text-sm font-medium text-muted-foreground">支持主流支付渠道</div>
      <div class="mt-6 flex flex-wrap items-center justify-center gap-4">
        <div v-for="(p, i) in payMethods" :key="p.key" v-reveal="i * 60" class="flex items-center gap-2 rounded-xl border border-border bg-card px-6 py-4 transition-colors hover:border-primary/40">
          <img :src="`/assets/icon/${p.key}.ico`" class="size-6" onerror="this.style.display='none'" />
          <span class="font-medium">{{ p.name }}</span>
        </div>
      </div>
    </section>

    <!-- ===== 左右图文场景 ===== -->
    <section class="mx-auto max-w-7xl space-y-24 px-4 py-16 lg:px-6">
      <div v-for="s in scenes" :key="s.title" class="grid items-center gap-10 lg:grid-cols-2">
        <div v-reveal :class="s.reverse ? 'lg:order-2' : ''">
          <Badge variant="default" class="mb-3">{{ s.tag }}</Badge>
          <h3 class="text-2xl font-bold tracking-tight sm:text-3xl">{{ s.title }}</h3>
          <p class="mt-3 leading-relaxed text-muted-foreground">{{ s.desc }}</p>
          <ul class="mt-5 space-y-2.5">
            <li v-for="pt in s.points" :key="pt" class="flex items-center gap-2 text-sm">
              <span class="flex size-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary"><Check class="size-3.5" /></span>
              <span>{{ pt }}</span>
            </li>
          </ul>
        </div>
        <div v-reveal="100" :class="s.reverse ? 'lg:order-1' : ''">
          <div class="relative flex aspect-[4/3] items-center justify-center overflow-hidden rounded-2xl border border-border bg-[oklch(0.18_0.03_264)]">
            <div class="absolute inset-0 tech-grid opacity-[0.35]" />
            <div class="absolute left-1/2 top-1/2 size-52 -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary/20 blur-3xl" />
            <component :is="s.icon" class="relative size-20 text-white/70" />
          </div>
        </div>
      </div>
    </section>

    <!-- ===== 费率方案 ===== -->
    <section id="pricing" class="mx-auto max-w-7xl scroll-mt-20 px-4 py-24 lg:px-6">
      <div v-reveal class="text-center">
        <h2 class="text-3xl font-bold tracking-tight sm:text-4xl">透明的费率方案</h2>
        <p class="mt-3 text-muted-foreground">按需选择，费率清晰，无隐藏费用</p>
      </div>
      <div class="mx-auto mt-14 grid max-w-5xl gap-5 lg:grid-cols-3">
        <div
          v-for="(p, i) in plans"
          :key="p.name"
          v-reveal="i * 80"
          class="relative rounded-2xl border bg-card p-7 transition-all duration-300"
          :class="p.highlight ? 'border-primary shadow-[0_20px_50px_-20px_oklch(0.58_0.2_262_/_0.5)] lg:-translate-y-3' : 'border-border hover:-translate-y-1'"
        >
          <Badge v-if="p.highlight" variant="default" class="absolute -top-3 left-7">最受欢迎</Badge>
          <div class="text-lg font-semibold">{{ p.name }}</div>
          <div class="mt-1 text-sm text-muted-foreground">{{ p.desc }}</div>
          <div class="mt-5 flex items-baseline gap-1">
            <span class="text-4xl font-bold tabular-nums">{{ p.price }}</span>
            <span class="text-lg text-muted-foreground">{{ p.unit }}</span>
          </div>
          <ul class="mt-6 space-y-3">
            <li v-for="ft in p.features" :key="ft" class="flex items-center gap-2 text-sm">
              <Check class="size-4 shrink-0 text-success" />
              <span class="text-muted-foreground">{{ ft }}</span>
            </li>
          </ul>
          <Button class="mt-7 w-full" :variant="p.highlight ? 'default' : 'outline'" @click="router.push('/m/reg')">{{ p.cta }}</Button>
        </div>
      </div>
    </section>

    <!-- ===== 接入步骤 ===== -->
    <section id="steps" class="relative border-y border-border bg-content">
      <div class="pointer-events-none absolute inset-0 tech-dots opacity-40" />
      <div class="relative mx-auto max-w-7xl scroll-mt-20 px-4 py-24 lg:px-6">
        <div v-reveal class="text-center">
          <h2 class="text-3xl font-bold tracking-tight sm:text-4xl">5 步快速接入</h2>
          <p class="mt-3 text-muted-foreground">从注册到上线，最快 1 天完成</p>
        </div>
        <div class="mt-14 grid gap-4 sm:grid-cols-3 lg:grid-cols-5">
          <div v-for="(s, i) in steps" :key="s.title" v-reveal="i * 80" class="relative rounded-xl border border-border bg-card p-5 text-center">
            <div class="mx-auto flex size-12 items-center justify-center rounded-full bg-primary/[0.08] text-primary">
              <component :is="s.icon" class="size-6" />
            </div>
            <div class="mt-3 text-xs font-medium text-primary">STEP {{ i + 1 }}</div>
            <div class="mt-1 text-sm font-semibold">{{ s.title }}</div>
            <div class="mt-1 text-xs text-muted-foreground">{{ s.desc }}</div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== 客户评价 ===== -->
    <section class="mx-auto max-w-7xl px-4 py-24 lg:px-6">
      <div v-reveal class="text-center">
        <h2 class="text-3xl font-bold tracking-tight sm:text-4xl">商户怎么说</h2>
        <p class="mt-3 text-muted-foreground">来自各行业商户的真实反馈</p>
      </div>
      <div class="mt-14 grid gap-5 md:grid-cols-3">
        <div v-for="(t, i) in testimonials" :key="t.name" v-reveal="i * 80" class="rounded-2xl border border-border bg-card p-6">
          <Quote class="size-7 text-primary/25" />
          <p class="mt-3 text-sm leading-relaxed text-muted-foreground">{{ t.text }}</p>
          <div class="mt-5 flex items-center gap-3 border-t border-border/60 pt-4">
            <div class="flex size-9 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold text-primary">{{ t.name.slice(0, 1) }}</div>
            <div>
              <div class="text-sm font-medium">{{ t.name }}</div>
              <div class="text-xs text-muted-foreground">{{ t.role }}</div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== FAQ ===== -->
    <section id="faq" class="border-t border-border bg-content">
      <div class="mx-auto max-w-3xl scroll-mt-20 px-4 py-24 lg:px-6">
        <div v-reveal class="text-center">
          <h2 class="text-3xl font-bold tracking-tight sm:text-4xl">常见问题</h2>
          <p class="mt-3 text-muted-foreground">关于接入与使用，你可能想了解的</p>
        </div>
        <div class="mt-10 space-y-3">
          <div v-for="(f, i) in faqs" :key="i" v-reveal class="overflow-hidden rounded-xl border border-border bg-card">
            <button class="flex w-full items-center gap-3 px-5 py-4 text-left" @click="toggleFaq(i)">
              <span class="flex-1 text-sm font-medium">{{ f.q }}</span>
              <component :is="openFaq === i ? Minus : Plus" class="size-4 shrink-0 text-muted-foreground" />
            </button>
            <div v-if="openFaq === i" class="border-t border-border/60 px-5 py-4 text-sm leading-relaxed text-muted-foreground">{{ f.a }}</div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== 底部大 CTA（深色科技）===== -->
    <section class="mx-auto max-w-7xl px-4 py-24 lg:px-6">
      <div v-reveal class="relative overflow-hidden rounded-3xl border border-white/10 bg-[oklch(0.16_0.03_264)] px-8 py-16 text-center text-white">
        <div class="pointer-events-none absolute inset-0 tech-grid opacity-40" />
        <div class="pointer-events-none absolute left-1/2 top-0 h-64 w-[600px] -translate-x-1/2 rounded-full bg-primary/30 blur-[100px]" />
        <div class="relative">
          <h2 class="text-3xl font-bold tracking-tight sm:text-4xl">立即开启你的收款业务</h2>
          <p class="mx-auto mt-3 max-w-lg text-white/65">注册即可免费接入，多渠道聚合、超低费率、实时到账，助你的生意收款无忧。</p>
          <div class="mt-8 flex flex-wrap justify-center gap-3">
            <Button size="lg" @click="router.push('/m/reg')">免费注册 <ArrowRight class="size-4" /></Button>
            <Button variant="outline" size="lg" class="border-white/20 bg-white/5 text-white hover:bg-white/10 hover:text-white" @click="router.push('/docs')">查看接入文档</Button>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
