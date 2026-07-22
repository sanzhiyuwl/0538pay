<script setup lang="ts">
import { reactive, computed, ref, onMounted } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
import {
  oauthConfig,
  qqLoginOptions,
  alipayLoginOptions,
  wxLoginOptions,
} from '@/lib/mock/sysconfig'
import { fetchConfig, saveConfig } from '@/lib/api/config'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const cfg = reactive({ ...oauthConfig })
const saving = ref(false)
// QQ 互联需填 AppID/AppKey；聚合登录需填聚合参数
const showQQApp = computed(() => cfg.login_qq === '1')
const showAggregate = computed(
  () => cfg.login_qq === '3' || cfg.login_alipay === '-1' || cfg.login_wx === '-1',
)

onMounted(async () => {
  try {
    Object.assign(cfg, await fetchConfig('oauth'))
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  }
})

async function save() {
  saving.value = true
  try {
    await saveConfig('oauth', { ...cfg })
    toast.success('快捷登录设置已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="快捷登录配置" subtitle="配置 QQ / 支付宝 / 微信 第三方快捷登录">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">QQ 快捷登录</label>
          <Select v-model="cfg.login_qq" :options="qqLoginOptions" class="flex-1" />
        </div>
        <template v-if="showQQApp">
          <div class="row-field">
            <label class="lbl">QQ 登录 AppID</label>
            <input v-model="cfg.login_qq_appid" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">QQ 登录 AppKey</label>
            <input v-model="cfg.login_qq_appkey" class="field-input flex-1" />
          </div>
        </template>
        <div class="row-field">
          <label class="lbl">支付宝快捷登录</label>
          <Select v-model="cfg.login_alipay" :options="alipayLoginOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">微信快捷登录</label>
          <Select v-model="cfg.login_wx" :options="wxLoginOptions" class="flex-1" />
        </div>
        <p class="text-xs text-muted-foreground">
          支付宝需先添加 alipay 支付通道；微信需服务号并配置网页授权域名。
        </p>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving" @click="save"><Save />保存设置</Button>
      </div>
    </Panel>

    <!-- 彩虹聚合登录（选择聚合登录时显示） -->
    <Panel v-if="showAggregate" title="彩虹聚合登录配置" subtitle="使用第三方聚合登录服务时填写">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">聚合 API 地址</label>
          <input v-model="cfg.login_apiurl" placeholder="以 http:// 开头，以 / 结尾" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">应用 APPID</label>
          <input v-model="cfg.login_appid" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">应用 APPKEY</label>
          <input v-model="cfg.login_appkey" class="field-input flex-1" />
        </div>
      </div>
      <div class="mt-5 border-t border-border/60 pt-4">
        <Button :disabled="saving" @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>
