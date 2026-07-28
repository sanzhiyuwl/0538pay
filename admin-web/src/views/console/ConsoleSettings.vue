<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { Panel, Button, Switch } from '@/components/ui'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 进件计价规则（后端 config 分组 enroll，价格后台可配、程序不硬编码）
const form = reactive({
  enroll_pay_uid: '0',
  enroll_wholesale_price: '100',
  enroll_retail_price: '200',
  enroll_platform_share: '100',
  enroll_agent_share: '100',
  enroll_fail_refund: '1',
  enroll_path1_charge: '1',
  enroll_pay_timeout: '30',
  enroll_link_expire: '24',
})
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const kv = await fetchConfig('enroll')
    for (const k of Object.keys(form) as (keyof typeof form)[]) {
      if (kv[k] !== undefined) form[k] = kv[k]
    }
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载进件设置失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function save() {
  saving.value = true
  try {
    await saveConfig('enroll', { ...form })
    toast.success('已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="进件设置" subtitle="进件业务计价规则，价格后台可配（程序不硬编码金额）">
      <div v-if="loading" class="py-10 text-center dim">加载中…</div>
      <div v-else class="max-w-xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">收款商户 UID</label>
          <input v-model="form.enroll_pay_uid" class="field-input flex-1" />
        </div>
        <p class="pl-[110px] text-[11px] text-muted-foreground">
          进件开户费的收款商户 UID（平台收款方，收开户零售价）。必填且需为已存在的商户，未配置时无法建进件单。
        </p>

        <div class="row-field">
          <label class="lbl">名额批发价(元)</label>
          <input v-model="form.enroll_wholesale_price" class="field-input flex-1" />
        </div>
        <p class="pl-[110px] text-[11px] text-muted-foreground">代理预购名额（路径一）的单个批发价。</p>

        <div class="row-field">
          <label class="lbl">开户零售价(元)</label>
          <input v-model="form.enroll_retail_price" class="field-input flex-1" />
        </div>
        <p class="pl-[110px] text-[11px] text-muted-foreground">特约商户开户付的开户费，两条路径都在创建进件单时前置收取。</p>

        <div class="row-field">
          <label class="lbl">平台分成(元)</label>
          <input v-model="form.enroll_platform_share" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">代理分成(元)</label>
          <input v-model="form.enroll_agent_share" class="field-input flex-1" />
        </div>
        <p class="pl-[110px] text-[11px] text-muted-foreground">路径二（商户自付）进件成功后的分账比例。</p>

        <div class="row-field">
          <label class="lbl">待支付超时(分钟)</label>
          <input v-model="form.enroll_pay_timeout" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">链接有效期(小时)</label>
          <input v-model="form.enroll_link_expire" class="field-input flex-1" />
        </div>
        <p class="pl-[110px] text-[11px] text-muted-foreground">邀请链接终态事件（关单/驳回/退款完成）后起算的有效期。</p>

        <div class="row-field">
          <label class="lbl">失败退款</label>
          <div class="flex flex-1 items-center gap-2">
            <Switch
              :model-value="form.enroll_fail_refund === '1'"
              @update:model-value="(v) => (form.enroll_fail_refund = v ? '1' : '0')"
            />
            <span class="text-sm dim">进件失败/被驳回时自动原路退还商户开户费</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">路径一收费</label>
          <div class="flex flex-1 items-center gap-2">
            <Switch
              :model-value="form.enroll_path1_charge === '1'"
              @update:model-value="(v) => (form.enroll_path1_charge = v ? '1' : '0')"
            />
            <span class="text-sm dim">代理有名额时，客户是否仍单独付开户费（默认付）</span>
          </div>
        </div>

        <div class="pt-2">
          <Button :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存设置' }}</Button>
        </div>
      </div>
    </Panel>

    <Panel title="微信服务商凭证" subtitle="进件用的 sp_mchid 单例，已独立成「服务商配置」页">
      <p class="text-sm text-muted-foreground">
        平台微信服务商凭证（sp_mchid / 商户私钥 / 证书序列号 / APIv3 密钥 / 平台公钥）是全平台唯一一套，
        与"微信服务商模式收单"共用同一份，请在左侧「服务商配置」页维护，本页只管进件计价规则（避免配两遍导致事故）。
      </p>
    </Panel>
  </div>
</template>
