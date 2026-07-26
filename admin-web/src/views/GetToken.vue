<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Copy, Check } from 'lucide-vue-next'
import QRCodeLib from 'qrcode'
import { Panel, Button, Select } from '@/components/ui'
import { fetchWeixins } from '@/lib/api/paycfg'
import { fetchChannels } from '@/lib/api/channels'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

// 获取用户标识（对齐 epay admin/gettoken.php）：拼接指向 user/openid.php 的授权链接 + 二维码。
// 真实 token 换取在 user/openid.php 内做 OAuth（依赖真实微信/支付宝凭证，属乙类）；本页仅生成入口链接。
const toast = useToast()
const siteurl = window.location.origin + '/'

const tabs = [
  { key: 'wechat', label: '微信 Openid' },
  { key: 'alipayuid', label: '支付宝用户ID' },
  { key: 'apptoken', label: '支付宝应用授权 Token' },
]
const activeTab = ref('wechat')

type Opt = { value: string; label: string }
// 真通道数据：微信公众号(type=0 服务号) / 支付宝系通道(plugin 前缀 alipay)
const wxChannels = ref<Opt[]>([])
const alipayChannels = ref<Opt[]>([]) // alipayuid：alipay/alipaysl/alipayd/alipayrp
const alipaySlChannels = ref<Opt[]>([]) // apptoken：仅 alipaysl（服务商）

const authTypeOptions = [
  { value: '0', label: '基础应用授权' },
  { value: '1', label: '指定应用授权' },
]

const channel = ref('')
const authType = ref('0')

const channelOptions = computed<Opt[]>(() => {
  if (activeTab.value === 'wechat') return wxChannels.value
  if (activeTab.value === 'apptoken') return alipaySlChannels.value
  return alipayChannels.value
})

onMounted(async () => {
  try {
    const [wx, chPage] = await Promise.all([
      fetchWeixins(),
      fetchChannels({ pageSize: 200 }),
    ])
    wxChannels.value = wx.list
      .filter((w) => w.type === 0)
      .map((w) => ({ value: String(w.id), label: w.name }))
    const list = chPage.list || []
    alipayChannels.value = list
      .filter((c) => c.plugin && c.plugin.startsWith('alipay'))
      .map((c) => ({ value: String(c.id), label: `${c.name}（${c.plugin} #${c.id}）` }))
    alipaySlChannels.value = list
      .filter((c) => c.plugin === 'alipaysl')
      .map((c) => ({ value: String(c.id), label: `${c.name}（alipaysl #${c.id}）` }))
    pickDefault()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载通道失败')
  }
})

// 生成链接（对齐 gettoken.php 的 geturl 拼接规则）
const genUrl = computed(() => {
  if (!channel.value) return ''
  if (activeTab.value === 'wechat') return `${siteurl}user/openid.php?wechatid=${channel.value}`
  if (activeTab.value === 'alipayuid') return `${siteurl}user/openid.php?channel=${channel.value}`
  const act = authType.value === '1' ? 'app_auth_assign' : 'app_auth'
  return `${siteurl}user/openid.php?act=${act}&channel=${channel.value}`
})

const tip = computed(() =>
  activeTab.value === 'wechat' ? '复制链接后在微信中打开' : '复制链接后在支付宝中打开',
)

// 真二维码渲染（对齐 epay jquery.qrcode）
const qrDataURL = ref('')
watch(genUrl, async (url) => {
  if (!url) {
    qrDataURL.value = ''
    return
  }
  try {
    qrDataURL.value = await QRCodeLib.toDataURL(url, { width: 176, margin: 1 })
  } catch {
    qrDataURL.value = ''
  }
}, { immediate: true })

const copied = ref(false)
function copy() {
  if (!genUrl.value) return
  navigator.clipboard?.writeText(genUrl.value).then(() => {
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  }).catch(() => {})
}

// 切 Tab 后选中该 Tab 首个可用通道
function pickDefault() {
  channel.value = channelOptions.value[0]?.value ?? ''
}
function switchTab(key: string) {
  activeTab.value = key
  authType.value = '0'
  pickDefault()
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
          <Select v-if="channelOptions.length" v-model="channel" :options="channelOptions" class="flex-1" />
          <span v-else class="flex-1 text-sm text-muted-foreground">
            {{ activeTab === 'wechat' ? '暂无公众号，请先在「公众号小程序」中添加服务号' : '暂无支付宝通道，请先在「支付通道」中添加 alipay/alipaysl 通道' }}
          </span>
        </div>
        <div v-if="activeTab === 'apptoken'" class="row-field">
          <label class="lbl">授权方式</label>
          <Select v-model="authType" :options="authTypeOptions" class="flex-1" />
        </div>
        <template v-if="genUrl">
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

          <!-- 真二维码（对齐 epay jquery.qrcode，由获取链接生成） -->
          <div class="flex flex-col items-center gap-2 border-t border-border/60 pt-5">
            <div class="text-sm text-muted-foreground">或使用{{ activeTab === 'wechat' ? '微信' : '支付宝' }}扫描以下二维码</div>
            <img v-if="qrDataURL" :src="qrDataURL" alt="授权二维码" class="size-44 border border-border" />
            <div class="text-xs text-muted-foreground/70">二维码由获取链接生成</div>
          </div>
        </template>
      </div>

      <p class="mt-4 border-t border-border/60 pt-4 text-xs text-muted-foreground">
        微信需在公众号小程序管理中添加公众号；支付宝需先添加 alipay/alipaysl/alipayd 支付通道，并在支付宝应用授权回调地址中配置当前域名。
      </p>
    </Panel>
  </div>
</template>
