<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const settleOpenOptions = [
  { value: '0', label: '关闭结算' },
  { value: '1', label: '仅每日自动结算' },
  { value: '2', label: '仅手动申请结算' },
  { value: '3', label: '自动 + 手动结算' },
]
const settleTypeOptions = [
  { value: '0', label: 'D+0（可结算全部余额）' },
  { value: '1', label: 'D+1（可结算前 1 天余额）' },
]

// 键名对齐 epay set.php mod=settle
const form = reactive({
  settle_open: '3',
  settle_type: '1',
  settle_rate: '0.5',
  settle_money: '30',
  settle_fee_min: '0.1',
  settle_fee_max: '20',
  settle_maxlimit: '5',
})

const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const kv = await fetchConfig('settle')
    Object.assign(form, kv)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载结算设置失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function save() {
  saving.value = true
  try {
    await saveConfig('settle', { ...form })
    toast.success('结算设置已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="结算设置" subtitle="平台默认的结算规则，用户组可单独覆盖">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">结算开关</label>
          <Select v-model="form.settle_open" :options="settleOpenOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">结算周期</label>
          <Select v-model="form.settle_type" :options="settleTypeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">结算手续费</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.settle_rate" class="field-input w-28" /><span class="text-sm text-muted-foreground">%</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">最低结算金额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.settle_money" class="field-input w-28" /><span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">手续费封底</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.settle_fee_min" class="field-input w-28" /><span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">手续费封顶</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.settle_fee_max" class="field-input w-28" /><span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">每日提现次数</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.settle_maxlimit" class="field-input w-28" /><span class="text-sm text-muted-foreground">次（0 不限）</span>
          </div>
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          手续费 = 结算金额 × 费率，并在封底、封顶之间取值。此为平台默认规则，用户组可单独覆盖。
        </p>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving || loading" @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>
