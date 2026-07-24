<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { Building2, Target, Mail, Phone, MessageSquare, MapPin } from 'lucide-vue-next'
import { useSiteStore } from '@/stores/site'

// 关于我们：公司名/联系方式从网站设置 CMS 读取（后台可改，官网实时联动）。
const site = useSiteStore()
onMounted(() => site.hydrate())

const stats = [
  { value: '2016', label: '成立年份' },
  { value: '50,000+', label: '服务商户' },
  { value: '99.9%', label: '支付成功率' },
  { value: '7×24h', label: '技术支持' },
]

const contacts = computed(() => [
  { icon: MapPin, label: '公司地址', value: '山东省泰安市高新区数智大厦' },
  { icon: Phone, label: '联系电话', value: '0538-8888888' },
  { icon: Mail, label: '邮箱', value: site.config.email || 'service@epvia.com' },
  { icon: MessageSquare, label: '客服 QQ', value: site.config.qq || '800820538' },
])
</script>

<template>
  <div>
    <!-- 页头 -->
    <section class="site-surface border-b border-border">
      <div class="mx-auto max-w-7xl px-4 py-20 text-center lg:px-8">
        <h1 v-reveal class="text-4xl font-bold tracking-tight sm:text-5xl">关于 0538<span class="text-primary">Pay</span></h1>
        <p v-reveal class="mx-auto mt-5 max-w-2xl leading-relaxed text-muted-foreground">
          我们致力于为商户提供稳定、安全、易用的聚合支付服务，让每一笔收款简单而可靠。
        </p>
      </div>
    </section>

    <!-- 数据 -->
    <section class="border-b border-border">
      <div class="mx-auto grid max-w-7xl grid-cols-2 divide-x divide-border px-4 py-12 lg:grid-cols-4 lg:px-8">
        <div v-for="s in stats" :key="s.label" class="px-4 text-center">
          <div class="text-3xl font-bold tabular-nums text-primary sm:text-4xl">{{ s.value }}</div>
          <div class="mt-1.5 text-sm text-muted-foreground">{{ s.label }}</div>
        </div>
      </div>
    </section>

    <!-- 简介 + 使命 -->
    <section class="mx-auto max-w-7xl px-4 py-20 lg:px-8">
      <div class="grid gap-10 lg:grid-cols-2">
        <div v-reveal class="rounded-2xl border border-border bg-background p-8">
          <div class="flex size-11 items-center justify-center rounded-lg bg-primary/[0.08] text-primary"><Building2 class="size-5" /></div>
          <h2 class="mt-4 text-xl font-semibold">公司简介</h2>
          <p class="mt-3 leading-relaxed text-muted-foreground">
            0538Pay 是一家专注聚合支付的技术服务商，为电商、零售、生活服务等各行业商户提供支付宝、微信、QQ钱包、云闪付等多渠道聚合收款方案。
            平台以服务商模式运营，一次对接即可覆盖全渠道，配合开放 API 与完善的商户中心，帮助商户快速开启并高效管理线上收款业务。
          </p>
        </div>
        <div v-reveal="100" class="rounded-2xl border border-border bg-background p-8">
          <div class="flex size-11 items-center justify-center rounded-lg bg-primary/[0.08] text-primary"><Target class="size-5" /></div>
          <h2 class="mt-4 text-xl font-semibold">我们的使命</h2>
          <p class="mt-3 leading-relaxed text-muted-foreground">
            让支付接入前所未有地简单。我们相信，好的支付服务应当是稳定的、透明的、以商户为中心的。
            通过持续打磨产品与技术，我们希望降低每一位创业者与开发者的接入门槛，让他们把精力放在业务本身。
          </p>
        </div>
      </div>
    </section>

    <!-- 联系方式 -->
    <section class="site-surface border-t border-border">
      <div class="mx-auto max-w-7xl px-4 py-20 lg:px-8">
        <div v-reveal class="text-center">
          <h2 class="text-3xl font-bold tracking-tight">联系我们</h2>
          <p class="mt-3 text-muted-foreground">工作日 9:00-18:00，技术支持 7×24 小时在线</p>
        </div>
        <div v-reveal class="mx-auto mt-12 grid max-w-4xl gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div v-for="c in contacts" :key="c.label" class="rounded-xl border border-border bg-background p-5 text-center">
            <div class="mx-auto flex size-11 items-center justify-center rounded-full bg-primary/[0.08] text-primary"><component :is="c.icon" class="size-5" /></div>
            <div class="mt-3 text-sm font-medium">{{ c.label }}</div>
            <div class="mt-1 text-sm text-muted-foreground">{{ c.value }}</div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
