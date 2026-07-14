<script setup lang="ts">
import { ref, computed } from 'vue'
import { Wallet } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
import { formatMoney } from '@/lib/utils'

// 对齐 epay user/recharge.php
const balance = 12860.55
const payTypes = [
  { value: 'alipay', label: '支付宝', rate: 0 },
  { value: 'wxpay', label: '微信支付', rate: 0 },
  { value: 'qqpay', label: 'QQ钱包', rate: 0.6 },
]
const payOptions = payTypes.map((t) => ({ value: t.value, label: t.rate ? `${t.label}（费率 ${t.rate}%）` : t.label }))

const amount = ref('')
const payType = ref('alipay')
const currentRate = computed(() => payTypes.find((t) => t.value === payType.value)?.rate ?? 0)
const need = computed(() => {
  const m = Number(amount.value) || 0
  return m > 0 ? Math.round(m * (1 + currentRate.value / 100) * 100) / 100 : 0
})
const canSubmit = computed(() => (Number(amount.value) || 0) > 0)
function recharge() {
  if (!canSubmit.value) return
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="余额充值" subtitle="在线充值账户余额，用于平台消费与退款">
      <div class="max-w-2xl space-y-3.5">
        <div class="border border-border/70 p-4">
          <div class="text-[13px] text-muted-foreground">当前余额</div>
          <div class="mt-1.5 text-2xl font-normal tabular-nums text-primary"><span class="text-sm opacity-70">¥</span>{{ formatMoney(balance) }}</div>
        </div>
        <div class="row-field">
          <label class="lbl">充值金额</label>
          <input v-model="amount" class="field-input flex-1" placeholder="请输入充值金额" />
        </div>
        <div class="row-field">
          <label class="lbl">支付方式</label>
          <Select v-model="payType" :options="payOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">共需支付</label>
          <div class="flex-1 text-lg font-medium tabular-nums text-primary">¥{{ formatMoney(need) }}</div>
        </div>
        <div class="border-t border-border/60 pt-4">
          <Button :disabled="!canSubmit" @click="recharge"><Wallet />立即充值</Button>
        </div>
        <p class="rounded bg-warning/[0.08] px-3 py-2 text-xs text-warning">
          充值余额仅限平台消费/退款使用，禁止充值后套现提现。
        </p>
      </div>
    </Panel>
  </div>
</template>
