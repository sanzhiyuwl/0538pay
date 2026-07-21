<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ArrowDownToLine, ArrowUpFromLine } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
import { fetchDepositInfo, rechargeDeposit, withdrawDeposit, type DepositInfo } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useMerchantAuthStore } from '@/stores/merchantAuth'
import { useToast } from '@/composables/useToast'
import { formatMoney } from '@/lib/utils'

const toast = useToast()
const auth = useMerchantAuthStore()

const info = ref<DepositInfo>({ deposit: 0, depositMin: 0, money: 0 })
const busy = ref(false)

// 余额支付即时到账；渠道支付待凭证，先只放余额支付
const payOptions = [{ value: 'balance', label: '余额支付' }]

const rechargeForm = reactive({ amount: '', type: 'balance' })
const withdrawForm = reactive({ amount: '' })

async function load() {
  try {
    info.value = await fetchDepositInfo()
    // 预填补足到门槛的建议充值额
    const gap = info.value.depositMin - info.value.deposit
    rechargeForm.amount = gap > 0 ? String(gap) : ''
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载保证金信息失败')
  }
}
onMounted(load)

const canRecharge = computed(() => (Number(rechargeForm.amount) || 0) > 0)
const canWithdraw = computed(() => {
  const m = Number(withdrawForm.amount) || 0
  return m > 0 && m <= info.value.deposit
})

async function doRecharge() {
  if (!canRecharge.value || busy.value) return
  busy.value = true
  try {
    await rechargeDeposit(rechargeForm.amount, rechargeForm.type)
    toast.success('保证金充值成功')
    await Promise.all([load(), auth.refreshInfo()])
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '充值失败')
  } finally {
    busy.value = false
  }
}
async function doWithdraw() {
  if (!canWithdraw.value || busy.value) return
  busy.value = true
  try {
    await withdrawDeposit(withdrawForm.amount)
    toast.success('保证金已提取到余额')
    withdrawForm.amount = ''
    await Promise.all([load(), auth.refreshInfo()])
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '提取失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 保证金概况 -->
    <Panel title="保证金" subtitle="平台要求的账户保证金，不足时无法调用部分接口">
      <div class="max-w-2xl">
        <div class="grid grid-cols-2 gap-4">
          <div class="border border-border/70 p-4">
            <div class="text-[13px] text-muted-foreground">当前保证金</div>
            <div class="mt-1.5 text-2xl font-normal tabular-nums" :class="info.deposit < info.depositMin ? 'text-destructive' : 'text-success'">
              <span class="text-sm opacity-70">¥</span>{{ formatMoney(info.deposit) }}
            </div>
          </div>
          <div class="border border-border/70 p-4">
            <div class="text-[13px] text-muted-foreground">最低保证金要求</div>
            <div class="mt-1.5 text-2xl font-normal tabular-nums"><span class="text-sm opacity-70">¥</span>{{ formatMoney(info.depositMin) }}</div>
          </div>
        </div>
        <p v-if="info.deposit < info.depositMin" class="mt-3 rounded bg-destructive/[0.08] px-3 py-2 text-xs text-destructive">
          当前保证金不足 ¥{{ formatMoney(info.depositMin) }}，部分支付接口将无法调用，请及时充值。
        </p>
      </div>
    </Panel>

    <!-- 充值保证金 -->
    <Panel title="充值保证金">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">充值金额</label>
          <input v-model="rechargeForm.amount" class="field-input flex-1" placeholder="请输入充值金额" />
        </div>
        <div class="row-field">
          <label class="lbl">支付方式</label>
          <Select v-model="rechargeForm.type" :options="payOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">可用余额</label>
          <div class="flex-1 text-sm tabular-nums text-muted-foreground">¥{{ formatMoney(info.money) }}</div>
        </div>
        <div class="border-t border-border/60 pt-4">
          <Button :disabled="!canRecharge || busy" @click="doRecharge"><ArrowDownToLine />从余额充值</Button>
        </div>
        <p class="text-xs text-muted-foreground">余额支付即时从可用余额划转至保证金；渠道充值待支付渠道凭证接入。</p>
      </div>
    </Panel>

    <!-- 提取保证金 -->
    <Panel title="提取保证金" subtitle="将保证金提取回商户可用余额">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">提取金额</label>
          <input v-model="withdrawForm.amount" class="field-input flex-1" :placeholder="`最多可提取 ¥${formatMoney(info.deposit)}`" />
        </div>
        <div class="border-t border-border/60 pt-4">
          <Button variant="outline" :disabled="!canWithdraw || busy" @click="doWithdraw"><ArrowUpFromLine />提取到余额</Button>
        </div>
        <p class="text-xs text-muted-foreground">提取后若保证金低于最低要求，可能影响接口使用。</p>
      </div>
    </Panel>
  </div>
</template>
