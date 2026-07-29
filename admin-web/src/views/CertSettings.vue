<script setup lang="ts">
import { reactive, computed, ref, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
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
// 强制认证单选：显示值 1=开启 2=关闭；底层存储仍是 cert_force 0/1（后端按 Bool 读，只认 "1"）。
const forceRadio = computed({
  get: () => (cfg.cert_force === '1' ? '1' : '2'),
  set: (v: string) => (cfg.cert_force = v === '1' ? '1' : '0'),
})
async function save() {
  saving.value = true
  try {
    // 企业认证不再用独立开关：企业校验 APPCODE 填了即开启(cert_corpopen=1)，留空则关闭。
    cfg.cert_corpopen = cfg.cert_appcode2.trim() ? '1' : '0'
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
          <label class="lbl">强制认证</label>
          <div class="flex flex-1 items-center gap-5">
            <label class="flex items-center gap-1.5 text-sm"><input v-model="forceRadio" type="radio" value="1" class="cert-radio" />开启</label>
            <label class="flex items-center gap-1.5 text-sm"><input v-model="forceRadio" type="radio" value="2" class="cert-radio" />关闭</label>
          </div>
        </div>
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
        <template v-if="isPhone3">
          <div class="row-field">
            <label class="lbl">调用地址</label>
            <input v-model="cfg.cert_phone3_url" class="field-input flex-1" placeholder="接口文档里的完整调用地址" />
          </div>
          <div class="row-field">
            <label class="lbl">APPCODE</label>
            <input v-model="cfg.cert_appcode" class="field-input flex-1" placeholder="阿里云云市场购买后在“已购买的服务”获取" />
          </div>
        </template>

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
          <div class="row-field">
            <label class="lbl">企业校验 APPCODE</label>
            <input v-model="cfg.cert_appcode2" class="field-input flex-1" placeholder="留空=不开启企业认证；填写后即开启" />
          </div>
        </template>
      </div>
    </Panel>

    <!-- 实名工本费（对齐 epay mod=certificate cert_money，独立成栏避免与接口配置混杂） -->
    <Panel title="实名工本费" subtitle="认证成功时从商户余额扣除的费用，付费注册商户建议免费">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">工本费</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="cfg.cert_money" class="field-input w-40" /><span class="text-sm text-muted-foreground">元（0=免费）</span>
          </div>
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          认证成功将从商户余额扣除工本费（支付宝身份验证接口 1 元/次）。留 0 表示免费，付费注册商户建议免认证费。
        </p>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving" @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>

<style scoped>
.cert-radio {
  width: 15px;
  height: 15px;
  accent-color: var(--primary);
  cursor: pointer;
}
</style>
