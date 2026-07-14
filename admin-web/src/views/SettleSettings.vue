<script setup lang="ts">
import { reactive } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
import { settleConfig, settleOpenOptions, settleTypeOptions } from '@/lib/mock/settings'

const settle = reactive({ ...settleConfig })
function save() {}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="结算设置" subtitle="平台默认的结算规则，用户组可单独覆盖">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">结算开关</label>
          <Select v-model="settle.settleOpen" :options="settleOpenOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">结算周期</label>
          <Select v-model="settle.settleType" :options="settleTypeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">结算手续费</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="settle.settleRate" class="field-input w-28" /><span class="text-sm text-muted-foreground">%</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">最低结算金额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="settle.settleMin" class="field-input w-28" /><span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">自动结算时间</label>
          <input v-model="settle.autoSettleTime" class="field-input w-28" />
        </div>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>
