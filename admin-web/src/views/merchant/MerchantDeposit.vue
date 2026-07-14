<script setup lang="ts">
import { ref, computed } from 'vue'
import { ArrowDownToLine, ArrowUpFromLine } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
import { formatMoney } from '@/lib/utils'

// 对齐 epay user/deposit.php
const deposit = 500.0 // 当前保证金
const depositMin = 1000 // 最低保证金要求
const balance = 12860.55 // 可用余额

const payOptions = [
  { value: 'balance', label: '余额支付' },
  { value: 'alipay', label: '支付宝' },
  { value: 'wxpay', label: '微信支付' },
]

// 充值保证金
const rechargeAmount = ref(depositMin - deposit > 0 ? String(depositMin - deposit) : '')
const rechargeType = ref('balance')
const canRecharge = computed(() => (Number(rechargeAmount.value) || 0) > 0)

// 提取保证金
const withdrawAmount = ref('')
const canWithdraw = computed(() => {
  const m = Number(withdrawAmount.value) || 0
  return m > 0 && m <= deposit
})

function doRecharge() {}
function doWithdraw() {}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 保证金概况 -->
    <Panel title="保证金" subtitle="平台要求的账户保证金，不足时无法调用部分接口">
      <div class="max-w-2xl">
        <div class="grid grid-cols-2 gap-4">
          <div class="border border-border/70 p-4">
            <div class="text-[13px] text-muted-foreground">当前保证金</div>
            <div class="mt-1.5 text-2xl font-normal tabular-nums" :class="deposit < depositMin ? 'text-destructive' : 'text-success'">
              <span class="text-sm opacity-70">¥</span>{{ formatMoney(deposit) }}
            </div>
          </div>
          <div class="border border-border/70 p-4">
            <div class="text-[13px] text-muted-foreground">最低保证金要求</div>
            <div class="mt-1.5 text-2xl font-normal tabular-nums"><span class="text-sm opacity-70">¥</span>{{ formatMoney(depositMin) }}</div>
          </div>
        </div>
        <p v-if="deposit < depositMin" class="mt-3 rounded bg-destructive/[0.08] px-3 py-2 text-xs text-destructive">
          当前保证金不足 ¥{{ formatMoney(depositMin) }}，部分支付接口将无法调用，请及时充值。
        </p>
      </div>
    </Panel>

    <!-- 充值保证金 -->
    <Panel title="充值保证金">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">充值金额</label>
          <input v-model="rechargeAmount" class="field-input flex-1" placeholder="请输入充值金额" />
        </div>
        <div class="row-field">
          <label class="lbl">支付方式</label>
          <Select v-model="rechargeType" :options="payOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">可用余额</label>
          <div class="flex-1 text-sm tabular-nums text-muted-foreground">¥{{ formatMoney(balance) }}</div>
        </div>
        <div class="border-t border-border/60 pt-4">
          <Button :disabled="!canRecharge" @click="doRecharge"><ArrowDownToLine />立即充值</Button>
        </div>
      </div>
    </Panel>

    <!-- 提取保证金 -->
    <Panel title="提取保证金" subtitle="将保证金提取回商户可用余额">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">提取金额</label>
          <input v-model="withdrawAmount" class="field-input flex-1" :placeholder="`最多可提取 ¥${formatMoney(deposit)}`" />
        </div>
        <div class="border-t border-border/60 pt-4">
          <Button variant="outline" :disabled="!canWithdraw" @click="doWithdraw"><ArrowUpFromLine />提取到余额</Button>
        </div>
        <p class="text-xs text-muted-foreground">提取后若保证金低于最低要求，可能影响接口使用。</p>
      </div>
    </Panel>
  </div>
</template>
