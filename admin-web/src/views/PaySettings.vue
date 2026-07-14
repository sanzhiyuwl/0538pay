<script setup lang="ts">
import { reactive } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Switch } from '@/components/ui'
import { payConfig } from '@/lib/mock/settings'

const pay = reactive({ ...payConfig })
function save() {}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="支付设置" subtitle="全站支付相关的金额限制与商品名称规则">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">最大支付金额</label>
          <input v-model="pay.maxMoney" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">最小支付金额</label>
          <input v-model="pay.minMoney" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">屏蔽关键词</label>
          <input v-model="pay.blockKeywords" placeholder="多个用逗号分隔" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">屏蔽提示语</label>
          <input v-model="pay.blockTip" class="field-input flex-1" />
        </div>
        <div class="row-switch"><span>商品名称自定义</span><Switch v-model="pay.customName" /></div>
        <div class="row-switch"><span>扫码页隐藏商品名</span><Switch v-model="pay.hideName" /></div>
        <div class="row-switch"><span>未填联系QQ禁止支付</span><Switch v-model="pay.qqRequired" /></div>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>
