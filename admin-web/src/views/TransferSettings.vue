<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button } from '@/components/ui'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 键名对齐 epay set.php mod=transfer / config group=transfer
const form = reactive({
  transfer_rate: '',
  transfer_minmoney: '1',
  transfer_maxmoney: '20000',
  transfer_maxlimit: '10',
})

const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    Object.assign(form, await fetchConfig('transfer'))
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载转账设置失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function save() {
  saving.value = true
  try {
    await saveConfig('transfer', { ...form })
    toast.success('转账付款设置已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="转账付款设置" subtitle="代付/转账的手续费率与单笔限额、每日次数上限">
      <template #actions>
        <Button size="sm" :disabled="saving" @click="save"><Save />保存</Button>
      </template>
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">手续费率</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.transfer_rate" class="field-input w-40" placeholder="留空则复用结算费率" /><span class="text-sm text-muted-foreground">%</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">单笔最小金额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.transfer_minmoney" class="field-input w-40" /><span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">单笔最大金额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.transfer_maxmoney" class="field-input w-40" /><span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">同账号每日次数上限</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.transfer_maxlimit" class="field-input w-40" /><span class="text-sm text-muted-foreground">次（0=不限）</span>
          </div>
        </div>
        <p class="border-t border-border/60 pt-3 text-xs text-muted-foreground">
          手续费率留空时复用结算费率。以上配置被代付发起逻辑实时消费（保存即生效，无需重启）。
        </p>
      </div>
    </Panel>
  </div>
</template>
