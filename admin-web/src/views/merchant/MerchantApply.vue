<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Wallet, Pencil, History } from 'lucide-vue-next'
import { Panel, Button } from '@/components/ui'
import { applyInfo as a } from '@/lib/mock/merchant/settle'
import { formatMoney } from '@/lib/utils'

const router = useRouter()
const amount = ref('')

function fillAll() {
  amount.value = String(a.enableMoney)
}

// 手续费与到账实时计算
const fee = computed(() => {
  const m = Number(amount.value) || 0
  if (m <= 0) return 0
  return Math.min(Math.max((m * a.settleRate) / 100, a.settleFeeMin), a.settleFeeMax)
})
const realArrive = computed(() => {
  const m = Number(amount.value) || 0
  return Math.max(0, m - fee.value)
})

// 校验
const error = computed(() => {
  const m = Number(amount.value)
  if (!amount.value) return ''
  if (isNaN(m) || m <= 0) return '请输入有效金额'
  if (m < a.settleMin) return `最低提现额为 ¥${a.settleMin}`
  if (m > a.enableMoney) return '超过可提现余额'
  if (a.todayCount >= a.settleMaxLimit) return `今日提现已达上限（${a.settleMaxLimit} 次）`
  return ''
})
const canSubmit = computed(() => !!amount.value && !error.value)

function submit() {
  if (!canSubmit.value) return
  amount.value = ''
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="申请提现" subtitle="手动发起余额提现，到账时间以结算方式为准">
      <template #actions>
        <Button variant="outline" size="sm" @click="router.push('/m/settle')"><History />结算记录</Button>
      </template>

      <div class="max-w-2xl space-y-3.5">
        <!-- 余额概况 -->
        <div class="grid grid-cols-2 gap-4">
          <div class="border border-border/70 p-4">
            <div class="text-[13px] text-muted-foreground">当前余额</div>
            <div class="mt-1.5 text-2xl font-normal tabular-nums"><span class="text-sm opacity-70">¥</span>{{ formatMoney(a.money) }}</div>
          </div>
          <div class="border border-border/70 p-4">
            <div class="text-[13px] text-muted-foreground">可提现余额</div>
            <div class="mt-1.5 text-2xl font-normal tabular-nums text-primary"><span class="text-sm opacity-70">¥</span>{{ formatMoney(a.enableMoney) }}</div>
          </div>
        </div>

        <!-- 提现方式 -->
        <div class="row-field">
          <label class="lbl">提现方式</label>
          <div class="flex flex-1 items-center gap-2">
            <span class="text-sm">{{ a.settleName }}</span>
            <span class="text-sm text-muted-foreground">{{ a.account }}（{{ a.username }}）</span>
            <button class="ml-auto inline-flex items-center gap-1 text-xs text-primary hover:underline" @click="router.push('/m/profile')">
              <Pencil class="size-3.5" />修改收款账号
            </button>
          </div>
        </div>

        <!-- 提现金额 -->
        <div class="row-field">
          <label class="lbl">提现金额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="amount" placeholder="请输入提现金额" class="field-input flex-1" />
            <Button variant="outline" size="sm" @click="fillAll">全部</Button>
          </div>
        </div>
        <p v-if="error" class="pl-[calc(6rem+1rem)] text-xs text-destructive">{{ error }}</p>

        <!-- 费用明细 -->
        <div class="row-field">
          <label class="lbl">手续费</label>
          <div class="flex-1 text-sm tabular-nums">
            <span class="text-destructive">¥{{ formatMoney(fee) }}</span>
            <span class="ml-2 text-xs text-muted-foreground">费率 {{ a.settleRate }}%（¥{{ a.settleFeeMin }}~¥{{ a.settleFeeMax }}）</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">预计到账</label>
          <div class="flex-1 text-lg font-medium tabular-nums text-success">¥{{ formatMoney(realArrive) }}</div>
        </div>

        <div class="border-t border-border/60 pt-4">
          <Button :disabled="!canSubmit" @click="submit"><Wallet />申请提现</Button>
        </div>

        <!-- 规则说明 -->
        <div class="border-t border-border/60 pt-4 text-xs leading-relaxed text-muted-foreground">
          <p>· 最低提现额 ¥{{ a.settleMin }}，每日最多提现 {{ a.settleMaxLimit }} 次（今日已 {{ a.todayCount }} 次）。</p>
          <p>· 结算模式：{{ a.settleType === 1 ? 'D+0（可结算全部余额）' : 'D+1（可结算前一日余额，当日已收次日可提）' }}。</p>
          <p>· 提现手续费按费率 {{ a.settleRate }}% 收取，最低 ¥{{ a.settleFeeMin }}、最高 ¥{{ a.settleFeeMax }}。</p>
        </div>
      </div>
    </Panel>
  </div>
</template>
