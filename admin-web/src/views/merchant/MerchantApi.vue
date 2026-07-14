<script setup lang="ts">
import { ref, computed } from 'vue'
import { Copy, Check, RefreshCw, KeyRound, ShieldCheck, FileText } from 'lucide-vue-next'
import { Panel, Button, Select, Modal } from '@/components/ui'

// 商户 API 信息（对齐 epay userinfo.php?mod=api）
const api = ref({
  apiurl: 'https://0538pay.com/',
  uid: 1001,
  mdkey: 'a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6',
  platformPublicKey:
    'MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0f8s...（平台公钥，商户验签平台回调用）...IDAQAB',
  merchantPublicKey:
    'MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA9k2p...（商户公钥，已上传平台）...IDAQAB',
  keytype: '0', // 0=MD5+RSA兼容 1=仅RSA
})

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

// 重置 MD5 密钥（原型：本地随机生成展示）
function resetMdKey() {
  const chars = 'abcdef0123456789'
  let s = ''
  for (let i = 0; i < 32; i++) s += chars[(i * 7 + 3) % chars.length]
  api.value.mdkey = s
}

// 生成/重置 RSA 密钥对：私钥一次性弹窗
const rsaOpen = ref(false)
const newPrivateKey = ref('')
function genRsaPair() {
  newPrivateKey.value =
    'MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQ...（这是新生成的商户私钥，仅此一次展示，请妥善保存，平台不存储）...Kj3n8Q=='
  rsaOpen.value = true
}

const keytypeSaved = ref('0')
const keytypeDirty = computed(() => keytypeSaved.value !== api.value.keytype)
function saveKeytype() {
  keytypeSaved.value = api.value.keytype
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
        <div class="row-field">
          <label class="lbl mt-2">平台公钥</label>
          <div class="flex flex-1 items-start gap-2">
            <textarea :value="api.platformPublicKey" readonly rows="3" class="field-input flex-1 resize-none bg-muted/40 py-2 font-mono text-[12px] leading-relaxed" style="height:auto" />
            <Button variant="outline" size="sm" @click="copy('ppk', api.platformPublicKey)">
              <component :is="copiedKey === 'ppk' ? Check : Copy" class="size-4" />
            </Button>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl mt-2">商户公钥</label>
          <div class="flex flex-1 items-start gap-2">
            <textarea :value="api.merchantPublicKey" readonly rows="3" class="field-input flex-1 resize-none bg-muted/40 py-2 font-mono text-[12px] leading-relaxed" style="height:auto" />
            <Button variant="outline" size="sm" @click="copy('mpk', api.merchantPublicKey)">
              <component :is="copiedKey === 'mpk' ? Check : Copy" class="size-4" />
            </Button>
          </div>
        </div>
        <p class="text-xs text-muted-foreground">商户私钥仅在生成时展示一次，平台不存储，请妥善保管。若遗失可重新生成密钥对。</p>
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
