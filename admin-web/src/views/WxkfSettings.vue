<script setup lang="ts">
import { reactive, computed } from 'vue'
import { Save, Copy, ArrowRight } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { Panel, Button, Select, Switch } from '@/components/ui'
import { wxkfConfig, paymsgModeOptions, kfAccountOptions } from '@/lib/mock/wework'

const router = useRouter()
const cfg = reactive({ ...wxkfConfig })
const payOn = computed({
  get: () => cfg.wework_payopen === '1',
  set: (v: boolean) => (cfg.wework_payopen = v ? '1' : '0'),
})
function copyCallback() {
  navigator.clipboard?.writeText(cfg.callbackUrl).catch(() => {})
}
function save() {}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="H5 跳转微信客服支付" subtitle="从手机网站跳转到微信客服完成支付，需先配置企业微信账号">
      <template #actions>
        <Button variant="outline" size="sm" @click="router.push('/wework')">
          企业微信账号列表<ArrowRight />
        </Button>
      </template>
      <div class="max-w-2xl space-y-3.5">
        <!-- 回调配置 -->
        <div class="row-field">
          <label class="lbl">回调 URL</label>
          <div class="flex flex-1 items-center gap-2">
            <input :value="cfg.callbackUrl" readonly class="field-input flex-1 bg-muted/40 font-mono text-xs" />
            <Button variant="outline" size="sm" @click="copyCallback"><Copy class="size-4" /></Button>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">Token</label>
          <input v-model="cfg.wework_token" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">EncodingAESKey</label>
          <input v-model="cfg.wework_aeskey" class="field-input flex-1" />
        </div>

        <!-- 支付配置 -->
        <div class="border-t border-border/60 pt-3.5">
          <div class="row-switch"><span>开启 H5 跳转微信客服支付</span><Switch v-model="payOn" /></div>
        </div>
        <div class="row-field">
          <label class="lbl">支付消息模式</label>
          <Select v-model="cfg.wework_paymsgmode" :options="paymsgModeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">客服账号</label>
          <Select v-model="cfg.wework_paykfid" :options="kfAccountOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">人工客服链接</label>
          <input v-model="cfg.wework_contact" placeholder="选填，追加在支付消息后面" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">消息尾部内容</label>
          <input v-model="cfg.wework_remark" placeholder="选填，支持变量 [qq] 当前商户联系QQ" class="field-input flex-1" />
        </div>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button @click="save"><Save />保存设置</Button>
      </div>
      <p class="mt-4 border-t border-border/60 pt-4 text-xs text-muted-foreground">
        开启前请确保配置正确，否则会导致手机浏览器无法微信支付。仅能使用独立版微信客服获取 token 并配置回调，不能开启企业微信内的微信客服应用。
      </p>
    </Panel>
  </div>
</template>
