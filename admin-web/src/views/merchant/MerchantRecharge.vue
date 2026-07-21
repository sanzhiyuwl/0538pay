<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Wallet } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
import { rechargeBalance } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useMerchantAuthStore } from '@/stores/merchantAuth'
import { useToast } from '@/composables/useToast'
import { formatMoney } from '@/lib/utils'

const toast = useToast()
const auth = useMerchantAuthStore()

// 充值走渠道下单 → 收银台支付 → 回调入账。mock 渠道可真跑，真实渠道待凭证。
const plugins = [
  { value: 'mock', label: '模拟支付（测试）' },
  { value: 'alipay', label: '支付宝（待凭证）' },
  { value: 'wxpay', label: '微信支付（待凭证）' },
]

const balance = computed(() => Number(auth.info?.money ?? 0))
const amount = ref('')
const plugin = ref('mock')
const busy = ref(false)
const canSubmit = computed(() => (Number(amount.value) || 0) > 0)

onMounted(() => auth.refreshInfo().catch(() => {}))

async function recharge() {
  if (!canSubmit.value || busy.value) return
  busy.value = true
  try {
    const res = await rechargeBalance(amount.value, plugin.value)
    toast.success('充值订单已创建，正在跳转收银台…')
    // 跳收银台完成支付（mock 渠道可点"模拟支付成功"回调入账）
    const url = res.pay_url || res.qrcode
    if (url) {
      window.open(url, '_blank')
    }
    amount.value = ''
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '充值下单失败')
  } finally {
    busy.value = false
  }
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
          <Select v-model="plugin" :options="plugins" class="flex-1" />
        </div>
        <div class="border-t border-border/60 pt-4">
          <Button :disabled="!canSubmit || busy" @click="recharge"><Wallet />立即充值</Button>
        </div>
        <p class="rounded bg-warning/[0.08] px-3 py-2 text-xs text-warning">
          充值余额仅限平台消费/退款使用，禁止充值后套现提现。充值下单后跳收银台完成支付，回调到账。
        </p>
      </div>
    </Panel>
  </div>
</template>
