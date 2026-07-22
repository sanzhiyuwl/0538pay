<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const refundFeeOptions = [
  { value: '0', label: '平台承担（退款退回商户分成）' },
  { value: '1', label: '商户承担（全额退时扣实付）' },
]

// 键名对齐 epay set.php mod=pay
const form = reactive({
  pay_maxmoney: '50000',
  pay_minmoney: '0.01',
  blockname: '博彩|赌博|违禁|毒品|枪支',
  blockalert: '温馨提醒该商品禁止出售',
  refund_fee_type: '0',
})

const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const kv = await fetchConfig('pay')
    Object.assign(form, kv)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载支付设置失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function save() {
  saving.value = true
  try {
    await saveConfig('pay', { ...form })
    toast.success('支付设置已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="支付设置" subtitle="全站支付金额限制、商品屏蔽词与退款手续费策略">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">最大支付金额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.pay_maxmoney" class="field-input w-40" /><span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">最小支付金额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.pay_minmoney" class="field-input w-40" /><span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">屏蔽关键词</label>
          <input v-model="form.blockname" placeholder="多个用竖线 | 分隔" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">屏蔽提示语</label>
          <input v-model="form.blockalert" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">退款手续费</label>
          <Select v-model="form.refund_fee_type" :options="refundFeeOptions" class="flex-1" />
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          屏蔽关键词命中商品名时拦截下单并记风控。退款手续费策略决定退款时从商户扣分成还是扣实付全额。
        </p>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving || loading" @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>
