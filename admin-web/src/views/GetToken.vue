<script setup lang="ts">
import { ref, computed } from 'vue'
import { Copy, Check, QrCode } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'

const siteurl = 'https://epvia.com/'

const tabs = [
  { key: 'wechat', label: '微信 Openid' },
  { key: 'alipayuid', label: '支付宝用户ID' },
  { key: 'apptoken', label: '支付宝应用授权 Token' },
]
const activeTab = ref('wechat')

// 各 Tab 的通道选项（原型：示例通道）
const wxChannels = [
  { value: '1', label: '默认服务号' },
  { value: '2', label: '备用服务号' },
]
const alipayChannels = [
  { value: '1', label: '支付宝主通道（alipay #1）' },
  { value: '2', label: '支付宝服务商（alipaysl #2）' },
]
const authTypeOptions = [
  { value: '0', label: '基础应用授权' },
  { value: '1', label: '指定应用授权' },
]

const channel = ref('1')
const authType = ref('0')

const channelOptions = computed(() =>
  activeTab.value === 'wechat' ? wxChannels : alipayChannels,
)

// 生成链接（对齐 gettoken.php 的 geturl 拼接规则）
const genUrl = computed(() => {
  if (activeTab.value === 'wechat') return `${siteurl}user/openid.php?wechatid=${channel.value}`
  if (activeTab.value === 'alipayuid') return `${siteurl}user/openid.php?channel=${channel.value}`
  const act = authType.value === '1' ? 'app_auth_assign' : 'app_auth'
  return `${siteurl}user/openid.php?act=${act}&channel=${channel.value}`
})

const tip = computed(() =>
  activeTab.value === 'wechat' ? '复制链接后在微信中打开' : '复制链接后在支付宝中打开',
)

const copied = ref(false)
function copy() {
  navigator.clipboard?.writeText(genUrl.value).then(() => {
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  }).catch(() => {})
}

function switchTab(key: string) {
  activeTab.value = key
  channel.value = '1'
  authType.value = '0'
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="获取用户标识" subtitle="生成授权链接 / 二维码，用于获取微信 Openid 或支付宝用户 ID">
      <!-- Tab -->
      <div class="mb-4 flex flex-wrap gap-1 border-b border-border">
        <button
          v-for="t in tabs"
          :key="t.key"
          class="-mb-px border-b-2 px-4 py-2 text-sm transition-colors"
          :class="
            activeTab === t.key
              ? 'border-primary font-medium text-primary'
              : 'border-transparent text-muted-foreground hover:text-foreground'
          "
          @click="switchTab(t.key)"
        >
          {{ t.label }}
        </button>
      </div>

      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">{{ activeTab === 'wechat' ? '选择公众号' : '选择支付通道' }}</label>
          <Select v-model="channel" :options="channelOptions" class="flex-1" />
        </div>
        <div v-if="activeTab === 'apptoken'" class="row-field">
          <label class="lbl">授权方式</label>
          <Select v-model="authType" :options="authTypeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">获取链接</label>
          <div class="flex flex-1 items-center gap-2">
            <input :value="genUrl" readonly class="field-input flex-1 bg-muted/40 font-mono text-xs" />
            <Button variant="outline" size="sm" @click="copy">
              <component :is="copied ? Check : Copy" class="size-4" />
              {{ copied ? '已复制' : '复制' }}
            </Button>
          </div>
        </div>
        <p class="text-xs text-success">{{ tip }}</p>

        <!-- 二维码占位（原型不生成真实二维码） -->
        <div class="flex flex-col items-center gap-2 border-t border-border/60 pt-5">
          <div class="text-sm text-muted-foreground">或使用{{ activeTab === 'wechat' ? '微信' : '支付宝' }}扫描以下二维码</div>
          <div class="flex size-44 items-center justify-center border border-border bg-muted/30 text-muted-foreground/40">
            <QrCode class="size-16" />
          </div>
          <div class="text-xs text-muted-foreground/70">二维码由获取链接生成</div>
        </div>
      </div>

      <p class="mt-4 border-t border-border/60 pt-4 text-xs text-muted-foreground">
        微信需在公众号小程序管理中添加公众号；支付宝需先添加 alipay/alipaysl/alipayd 支付通道，并在支付宝应用授权回调地址中配置当前域名。
      </p>
    </Panel>
  </div>
</template>
