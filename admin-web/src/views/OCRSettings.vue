<script setup lang="ts">
import { reactive, computed, ref, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
import {
  ocrConfig,
  ocrProviderOptions,
  ocrTencentRegionOptions,
} from '@/lib/mock/sysconfig'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const saving = ref(false)
const cfg = reactive({ ...ocrConfig })

onMounted(async () => {
  try {
    Object.assign(cfg, await fetchConfig('ocr'))
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  }
})

const isAliyun = computed(() => cfg.ocr_provider === '1')
const isTencent = computed(() => cfg.ocr_provider === '2')

async function save() {
  saving.value = true
  try {
    await saveConfig('ocr', { ...cfg })
    toast.success('OCR 识别设置已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="OCR 文字识别配置" subtitle="配置证件识别引擎，供商户实名认证与代理进件上传营业执照/身份证时自动识别回填">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">识别引擎</label>
          <Select v-model="cfg.ocr_provider" :options="ocrProviderOptions" class="flex-1" />
        </div>

        <!-- 阿里云 -->
        <template v-if="isAliyun">
          <div class="row-field">
            <label class="lbl">阿里云 AccessKeyId</label>
            <input v-model="cfg.ocr_aliyun_id" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">阿里云 AccessKeySecret</label>
            <input v-model="cfg.ocr_aliyun_key" class="field-input flex-1" />
          </div>
        </template>

        <!-- 腾讯云 -->
        <template v-if="isTencent">
          <div class="row-field">
            <label class="lbl">腾讯云 SecretId</label>
            <input v-model="cfg.ocr_tencent_id" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">腾讯云 SecretKey</label>
            <input v-model="cfg.ocr_tencent_key" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">地域</label>
            <Select v-model="cfg.ocr_tencent_region" :options="ocrTencentRegionOptions" class="flex-1" />
          </div>
        </template>
      </div>

      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving" @click="save"><Save />保存设置</Button>
      </div>
    </Panel>

    <Panel v-if="isAliyun" title="阿里云 OCR 开通说明" flush>
      <div class="space-y-1 px-4 py-3 text-xs leading-relaxed text-muted-foreground">
        <p>在 <a class="text-primary" href="https://ai.aliyun.com/ocr" target="_blank" rel="noreferrer">阿里云 OCR</a> 开通，需开通「个人证照识别」「企业资质识别」。</p>
        <p>密钥在 <a class="text-primary" href="https://usercenter.console.aliyun.com/#/manage/ak" target="_blank" rel="noreferrer">AccessKey 管理</a> 获取，与实名认证的阿里云密钥相互独立，可分别使用不同子账号。</p>
      </div>
    </Panel>

    <Panel v-if="isTencent" title="腾讯云 OCR 开通说明" flush>
      <div class="space-y-1 px-4 py-3 text-xs leading-relaxed text-muted-foreground">
        <p>在 <a class="text-primary" href="https://console.cloud.tencent.com/ocr" target="_blank" rel="noreferrer">腾讯云 OCR</a> 开通「营业执照识别」「身份证识别」。</p>
        <p>密钥在 <a class="text-primary" href="https://console.cloud.tencent.com/cam/capi" target="_blank" rel="noreferrer">API 密钥管理</a> 获取。</p>
      </div>
    </Panel>
  </div>
</template>
