<script setup lang="ts">
import { reactive, computed } from 'vue'
import { Save } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
import {
  oauthConfig,
  qqLoginOptions,
  alipayLoginOptions,
  wxLoginOptions,
} from '@/lib/mock/sysconfig'

const cfg = reactive({ ...oauthConfig })
// QQ 互联需填 AppID/AppKey；聚合登录需填聚合参数
const showQQApp = computed(() => cfg.login_qq === '1')
const showAggregate = computed(
  () => cfg.login_qq === '3' || cfg.login_alipay === '-1' || cfg.login_wx === '-1',
)
function save() {}
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
        <Button @click="save"><Save />保存设置</Button>
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
        <Button @click="save"><Save />保存设置</Button>
      </div>
    </Panel>
  </div>
</template>
