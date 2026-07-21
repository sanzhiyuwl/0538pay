<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Copy, Check, RefreshCw, KeyRound, ShieldCheck, FileText } from 'lucide-vue-next'
import { Panel, Button, Select, Modal } from '@/components/ui'
import { fetchApiInfo, resetApiKey } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 商户 API 信息（V1 MD5 真接口；V2 RSA 部分待 V2 协议上线）
const api = ref({
  apiurl: '',
  uid: 0,
  mdkey: '',
  keytype: '0', // 0=MD5+RSA兼容 1=仅RSA（V2，暂只前端展示）
})
async function loadApi() {
  try {
    const info = await fetchApiInfo()
    api.value.apiurl = info.apiurl
    api.value.uid = info.uid
    api.value.mdkey = info.mdkey
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : 'API 信息加载失败')
  }
}
onMounted(loadApi)

const keytypeOptions = [
  { value: '0', label: 'MD5 + RSA 兼容模式' },
  { value: '1', label: '仅 RSA 安全模式' },
]

// 复制
const copiedKey = ref<string | null>(null)
function copy(key: string, val: string) {
  navigator.clipboard?.writeText(val).then(() => {
    copiedKey.value = key
    setTimeout(() => (copiedKey.value = null), 1500)
  }).catch(() => {})
}

// 重置 MD5 密钥（真接口，原密钥立即失效）
const resetting = ref(false)
async function resetMdKey() {
  if (resetting.value) return
  resetting.value = true
  try {
    const res = await resetApiKey()
    api.value.mdkey = res.mdkey
    toast.success('密钥已重置，请同步更新对接代码')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '重置失败')
  } finally {
    resetting.value = false
  }
}

// V2 RSA 密钥对：V2 协议尚未上线，暂提示待开放
const rsaOpen = ref(false)
const newPrivateKey = ref('')
function genRsaPair() {
  toast.info('RSA（V2 接口）即将开放，敬请期待')
}

const keytypeSaved = ref('0')
const keytypeDirty = computed(() => keytypeSaved.value !== api.value.keytype)
function saveKeytype() {
  toast.info('签名模式（V2）即将开放')
  api.value.keytype = keytypeSaved.value
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 基础信息 -->
    <Panel title="接口信息" subtitle="对接支付接口所需的地址与商户标识">
      <div class="max-w-3xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">接口地址</label>
          <div class="flex flex-1 items-center gap-2">
            <input :value="api.apiurl" readonly class="field-input flex-1 bg-muted/40 font-mono text-[13px]" />
            <Button variant="outline" size="sm" @click="copy('url', api.apiurl)">
              <component :is="copiedKey === 'url' ? Check : Copy" class="size-4" />
            </Button>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">商户 ID</label>
          <div class="flex flex-1 items-center gap-2">
            <input :value="api.uid" readonly class="field-input flex-1 bg-muted/40 font-mono text-[13px]" />
            <Button variant="outline" size="sm" @click="copy('uid', String(api.uid))">
              <component :is="copiedKey === 'uid' ? Check : Copy" class="size-4" />
            </Button>
          </div>
        </div>
      </div>
    </Panel>

    <!-- V1 MD5 接口 -->
    <Panel title="V1 接口（MD5 签名）" subtitle="传统接口，使用商户密钥进行 MD5 签名">
      <div class="max-w-3xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">商户密钥</label>
          <div class="flex flex-1 items-center gap-2">
            <input :value="api.mdkey" readonly class="field-input flex-1 bg-muted/40 font-mono text-[13px]" />
            <Button variant="outline" size="sm" @click="copy('mdkey', api.mdkey)">
              <component :is="copiedKey === 'mdkey' ? Check : Copy" class="size-4" />
            </Button>
            <Button variant="outline" size="sm" class="text-destructive hover:text-destructive" @click="resetMdKey">
              <RefreshCw class="size-4" />重置
            </Button>
          </div>
        </div>
        <p class="text-xs text-muted-foreground">重置密钥后，原密钥立即失效，请同步更新你的对接代码。</p>
      </div>
    </Panel>

    <!-- V2 RSA 接口 -->
    <Panel title="V2 接口（RSA 签名）" subtitle="推荐使用，SHA256withRSA 非对称加密签名，更安全">
      <template #actions>
        <Button size="sm" @click="genRsaPair"><KeyRound />生成/重置密钥对</Button>
      </template>
      <div class="max-w-3xl space-y-3.5">
        <div class="rounded bg-muted/40 px-4 py-3 text-sm text-muted-foreground">
          RSA（V2 接口）签名与密钥对管理即将开放。当前请使用上方 V1 接口（MD5 签名）对接。
        </div>
      </div>
    </Panel>

    <!-- 签名方式 -->
    <Panel title="签名方式" subtitle="控制平台接受的签名类型">
      <div class="max-w-3xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">签名模式</label>
          <div class="flex flex-1 items-center gap-2">
            <Select v-model="api.keytype" :options="keytypeOptions" class="w-64" />
            <Button size="sm" :disabled="!keytypeDirty" @click="saveKeytype"><ShieldCheck class="size-4" />保存</Button>
          </div>
        </div>
        <p class="text-xs text-muted-foreground">仅 RSA 安全模式下，MD5 签名的请求将被拒绝，安全性更高。</p>
      </div>
    </Panel>

    <!-- 开发文档 -->
    <Panel title="开发文档" subtitle="接口对接说明">
      <div class="flex flex-wrap gap-3">
        <a href="#" class="inline-flex items-center gap-2 border border-border px-4 py-2 text-sm transition-colors hover:border-primary/40 hover:text-primary">
          <FileText class="size-4" />V1 接口文档（MD5）
        </a>
        <a href="#" class="inline-flex items-center gap-2 border border-border px-4 py-2 text-sm transition-colors hover:border-primary/40 hover:text-primary">
          <FileText class="size-4" />V2 接口文档（RSA）
        </a>
      </div>
    </Panel>

    <!-- 新私钥弹窗 -->
    <Modal v-model="rsaOpen" title="商户私钥（仅展示一次）" width="max-w-lg">
      <div class="space-y-3">
        <div class="rounded bg-warning/[0.1] px-3 py-2 text-xs text-warning">
          请立即复制并妥善保存此私钥，关闭后将无法再次查看。平台不存储你的私钥。
        </div>
        <textarea :value="newPrivateKey" readonly rows="5" class="field-input w-full resize-none bg-muted/40 py-2 font-mono text-[12px] leading-relaxed" style="height:auto" />
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="copy('newpk', newPrivateKey)">
          <component :is="copiedKey === 'newpk' ? Check : Copy" class="size-4" />复制私钥
        </Button>
        <Button size="sm" @click="rsaOpen = false">我已保存</Button>
      </template>
    </Modal>
  </div>
</template>
