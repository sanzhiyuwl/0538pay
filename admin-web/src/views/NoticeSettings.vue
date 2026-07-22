<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Switch } from '@/components/ui'
import { noticeConfig } from '@/lib/mock/sysconfig'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const saving = ref(false)
const cfg = reactive({ ...noticeConfig })

onMounted(async () => {
  try {
    Object.assign(cfg, await fetchConfig('notice'))
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  }
})
// 用 0/1 字符串 与 Switch 的布尔值转换
function bind(key: keyof typeof cfg) {
  return {
    get: () => cfg[key] === '1',
    set: (v: boolean) => (cfg[key] = v ? '1' : '0'),
  }
}
const wxOn = bind('wxnotice')
const mailOn = bind('mailnotice')
const regauditOn = bind('msgconfig_regaudit')
const applyOn = bind('msgconfig_apply')
const domainOn = bind('msgconfig_domain')
const orderOn = bind('msgconfig_order')
const settleOn = bind('msgconfig_settle')
const balanceOn = bind('msgconfig_balance')
async function save() {
  saving.value = true
  try {
    await saveConfig('notice', { ...cfg })
    toast.success('消息提醒设置已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 微信公众号消息 -->
    <Panel title="微信公众号消息提醒" subtitle="需同时开启微信快捷登录，且用户在用户中心绑定后生效">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-switch"><span>公众号消息总开关</span><Switch :model-value="wxOn.get()" @update:model-value="wxOn.set" /></div>
        <div class="row-field">
          <label class="lbl">新订单通知模板ID</label>
          <input v-model="cfg.wxnotice_tpl_order" placeholder="留空则不开启此消息" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">结算通知模板ID</label>
          <input v-model="cfg.wxnotice_tpl_settle" placeholder="留空则不开启此消息" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">登录通知模板ID</label>
          <input v-model="cfg.wxnotice_tpl_login" placeholder="留空则不开启此消息" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">余额不足提醒模板ID</label>
          <input v-model="cfg.wxnotice_tpl_balance" placeholder="留空则不开启此消息" class="field-input flex-1" />
        </div>
      </div>
    </Panel>

    <!-- 邮件消息 -->
    <Panel title="邮件消息提醒" subtitle="管理员与用户接收邮件的开关设置">
      <div class="max-w-2xl space-y-5">
        <div class="space-y-3.5">
          <h4 class="text-sm font-medium">管理员接收</h4>
          <div class="row-switch"><span>新注册商户待审核提醒</span><Switch :model-value="regauditOn.get()" @update:model-value="regauditOn.set" /></div>
          <div class="row-switch"><span>商户手动提现提醒</span><Switch :model-value="applyOn.get()" @update:model-value="applyOn.set" /></div>
          <div class="row-switch"><span>授权域名待审核提醒</span><Switch :model-value="domainOn.get()" @update:model-value="domainOn.set" /></div>
        </div>
        <div class="space-y-3.5 border-t border-border/60 pt-5">
          <h4 class="text-sm font-medium">用户接收</h4>
          <div class="row-switch"><span>邮件消息总开关</span><Switch :model-value="mailOn.get()" @update:model-value="mailOn.set" /></div>
          <div class="row-switch"><span>新订单通知</span><Switch :model-value="orderOn.get()" @update:model-value="orderOn.set" /></div>
          <div class="row-switch"><span>结算通知</span><Switch :model-value="settleOn.get()" @update:model-value="settleOn.set" /></div>
          <div class="row-switch"><span>余额不足提醒</span><Switch :model-value="balanceOn.get()" @update:model-value="balanceOn.set" /></div>
        </div>
        <p class="text-xs text-muted-foreground">用户接收邮件除在此开启外，还需用户在用户中心手动开启才能收到。</p>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving" @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>
