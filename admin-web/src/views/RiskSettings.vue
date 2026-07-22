<script setup lang="ts">
import { reactive, ref, computed, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Switch } from '@/components/ui'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 键名对齐 config group=risk（被 scheduler 风控自动关停 cron 消费）
const form = reactive({
  auto_check_notify: '0',
  check_notify_count: '10',
  auto_check_sucrate: '0',
  check_sucrate_second: '600',
  check_sucrate_count: '20',
  check_sucrate_value: '30',
})

const loading = ref(false)
const saving = ref(false)

// 布尔开关经字符串 "0"/"1" 适配 Switch
const notifyOn = computed({
  get: () => form.auto_check_notify === '1',
  set: (v: boolean) => (form.auto_check_notify = v ? '1' : '0'),
})
const sucrateOn = computed({
  get: () => form.auto_check_sucrate === '1',
  set: (v: boolean) => (form.auto_check_sucrate = v ? '1' : '0'),
})

async function load() {
  loading.value = true
  try {
    Object.assign(form, await fetchConfig('risk'))
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载风控设置失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function save() {
  saving.value = true
  try {
    await saveConfig('risk', { ...form })
    toast.success('风控设置已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="风控设置" subtitle="配置自动风控触发阈值，命中后自动关停商户支付权限（由定时任务执行）">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-switch"><span>连续通知失败自动关停</span><Switch v-model="notifyOn" /></div>
        <div class="row-field">
          <label class="lbl">连续失败上限</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.check_notify_count" class="field-input w-28" /><span class="text-sm text-muted-foreground">单（近24小时通知放弃数达此值则关停）</span>
          </div>
        </div>

        <div class="row-switch border-t border-border/60 pt-3.5"><span>订单成功率自动关停</span><Switch v-model="sucrateOn" /></div>
        <div class="row-field">
          <label class="lbl">统计窗口</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.check_sucrate_second" class="field-input w-28" /><span class="text-sm text-muted-foreground">秒</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">最小样本订单数</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.check_sucrate_count" class="field-input w-28" /><span class="text-sm text-muted-foreground">单（达到才判定，避免小样本误伤）</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">成功率阈值</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.check_sucrate_value" class="field-input w-28" /><span class="text-sm text-muted-foreground">% 以下触发关停</span>
          </div>
        </div>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving" @click="save"><Save />保存设置</Button>
      </div>
      <p class="mt-3 text-xs text-muted-foreground">
        触发关停后会将商户支付权限置为关闭并记录一条风控记录（风控管理 → 风控记录可查）。投诉率风控依赖真实渠道投诉数据，暂不提供。
      </p>
    </Panel>
  </div>
</template>
