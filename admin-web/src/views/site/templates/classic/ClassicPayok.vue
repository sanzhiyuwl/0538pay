<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { CheckCircle2 } from 'lucide-vue-next'
import { Button } from '@/components/ui'

const route = useRoute()
// 支付成功页，参数来自支付回调跳转（原型用示例值兜底）
const tradeNo = computed(() => (route.query.trade_no as string) || '20260714120000123456')
const outTradeNo = computed(() => (route.query.out_trade_no as string) || 'OUT20260714001')
const money = computed(() => (route.query.money as string) || '9.90')
const name = computed(() => (route.query.name as string) || 'VIP会员')
const returnUrl = computed(() => (route.query.return_url as string) || '')

function goBack() {
  if (returnUrl.value) window.location.href = returnUrl.value
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-content px-4">
    <div class="w-full max-w-md rounded-2xl border border-border bg-background p-8 text-center shadow-sm">
      <div class="mx-auto flex size-16 items-center justify-center rounded-full bg-success/10 text-success">
        <CheckCircle2 class="size-9" />
      </div>
      <h1 class="mt-5 text-2xl font-bold tracking-tight">支付成功</h1>
      <p class="mt-2 text-sm text-muted-foreground">您的订单已支付完成，感谢您的使用</p>

      <div class="mt-6 space-y-2.5 rounded-xl bg-content p-5 text-left text-sm">
        <div class="flex justify-between"><span class="text-muted-foreground">商品名称</span><span>{{ name }}</span></div>
        <div class="flex justify-between"><span class="text-muted-foreground">支付金额</span><span class="font-semibold tabular-nums text-primary">¥{{ money }}</span></div>
        <div class="flex justify-between"><span class="text-muted-foreground">系统订单号</span><span class="font-mono text-xs">{{ tradeNo }}</span></div>
        <div class="flex justify-between"><span class="text-muted-foreground">商户订单号</span><span class="font-mono text-xs">{{ outTradeNo }}</span></div>
      </div>

      <div class="mt-6 flex justify-center gap-3">
        <Button v-if="returnUrl" @click="goBack">返回商户</Button>
        <Button variant="outline" @click="$router.push('/')">返回首页</Button>
      </div>
    </div>
  </div>
</template>
