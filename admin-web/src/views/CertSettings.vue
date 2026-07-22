<script setup lang="ts">
import { reactive, computed, ref, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Select, Switch } from '@/components/ui'
import {
  certConfig,
  certOpenOptions,
  certChannelOptions,
} from '@/lib/mock/sysconfig'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const saving = ref(false)
const cfg = reactive({ ...certConfig })

onMounted(async () => {
  try {
    Object.assign(cfg, await fetchConfig('cert'))
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  }
})
// 各认证方式对应的参数分组显隐（对齐 set.php setform1~6）
const isAlipay = computed(() => cfg.cert_open === '1' || cfg.cert_open === '3')
const isPhone3 = computed(() => cfg.cert_open === '2')
const isWx = computed(() => cfg.cert_open === '4')
const isAliyun = computed(() => cfg.cert_open === '5')
const enabled = computed(() => cfg.cert_open !== '0')
const corpOn = computed({
  get: () => cfg.cert_corpopen === '1',
  set: (v: boolean) => (cfg.cert_corpopen = v ? '1' : '0'),
})
const forceOn = computed({
  get: () => cfg.cert_force === '1',
  set: (v: boolean) => (cfg.cert_force = v ? '1' : '0'),
})
async function save() {
  saving.value = true
  try {
    await saveConfig('cert', { ...cfg })
    toast.success('实名认证设置已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="实名认证配置" subtitle="配置商户实名认证方式与对应接口密钥">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">认证方式</label>
          <Select v-model="cfg.cert_open" :options="certOpenOptions" class="flex-1" />
        </div>

        <!-- 支付宝身份/实名 -->
        <div v-if="isAlipay" class="row-field">
          <label class="lbl">支付宝通道</label>
          <Select v-model="cfg.cert_channel" :options="certChannelOptions" class="flex-1" />
        </div>

        <!-- 手机三要素 -->
        <div v-if="isPhone3" class="row-field">
          <label class="lbl">APPCODE</label>
          <input v-model="cfg.cert_appcode" class="field-input flex-1" />
        </div>

        <!-- 微信扫码（腾讯云） -->
        <template v-if="isWx">
          <div class="row-field">
            <label class="lbl">腾讯云 SecretId</label>
            <input v-model="cfg.cert_qcloudid" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">腾讯云 SecretKey</label>
            <input v-model="cfg.cert_qcloudkey" class="field-input flex-1" />
          </div>
        </template>

        <!-- 阿里云金融级 -->
        <template v-if="isAliyun">
          <div class="row-field">
            <label class="lbl">阿里云 AccessKeyId</label>
            <input v-model="cfg.cert_aliyunid" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">阿里云 AccessKeySecret</label>
            <input v-model="cfg.cert_aliyunkey" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">认证场景ID</label>
            <input v-model="cfg.cert_aliyunsceneid" class="field-input flex-1" />
          </div>
        </template>

        <!-- 企业认证与强制认证（开启任一方式后可用） -->
        <template v-if="enabled">
          <div class="row-switch"><span>开启企业认证方式</span><Switch v-model="corpOn" /></div>
          <div v-if="corpOn" class="row-field">
            <label class="lbl">企业校验 APPCODE</label>
            <input v-model="cfg.cert_appcode2" class="field-input flex-1" />
          </div>
          <div class="row-switch"><span>商户强制认证</span><Switch v-model="forceOn" /></div>
        </template>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving" @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>
