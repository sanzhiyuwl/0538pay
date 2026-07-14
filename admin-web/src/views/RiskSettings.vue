<script setup lang="ts">
import { reactive } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Switch } from '@/components/ui'
import { riskConfig } from '@/lib/mock/settings'

const risk = reactive({ ...riskConfig })
function save() {}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="风控设置" subtitle="配置自动风控的触发阈值，命中后可自动限制收款">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-switch"><span>订单成功率风控</span><Switch v-model="risk.successRateOn" /></div>
        <div class="row-field">
          <label class="lbl">成功率阈值</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="risk.successRateMin" class="field-input w-28" /><span class="text-sm text-muted-foreground">% 以下触发</span>
          </div>
        </div>
        <div class="row-switch"><span>连续通知失败风控</span><Switch v-model="risk.notifyFailOn" /></div>
        <div class="row-field">
          <label class="lbl">连续失败上限</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="risk.notifyFailMax" class="field-input w-28" /><span class="text-sm text-muted-foreground">次触发</span>
          </div>
        </div>
        <div class="row-switch"><span>订单投诉率风控</span><Switch v-model="risk.complaintOn" /></div>
        <div class="row-field">
          <label class="lbl">投诉率阈值</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="risk.complaintMax" class="field-input w-28" /><span class="text-sm text-muted-foreground">% 以上触发</span>
          </div>
        </div>
        <div class="row-switch"><span>触发后自动限制收款</span><Switch v-model="risk.autoBlock" /></div>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>
