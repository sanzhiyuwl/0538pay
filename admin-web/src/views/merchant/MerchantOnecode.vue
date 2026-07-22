<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import QRCodeLib from 'qrcode'
import { Copy, Check, Download, Pencil } from 'lucide-vue-next'
import { Panel, Button } from '@/components/ui'
import { fetchOnecodeInfo, saveCodeName, type OnecodeInfo } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 聚合收款码（对齐 epay user/onecode.php）
const info = ref<OnecodeInfo>({ open: false, pay_url: '', codename: '' })
const codeName = ref('')
const editing = ref(false)
const busy = ref(false)
const qrDataURL = ref('')

async function load() {
  try {
    info.value = await fetchOnecodeInfo()
    codeName.value = info.value.codename
    await renderQR()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载收款码失败')
  }
}
onMounted(load)

async function renderQR() {
  if (!info.value.pay_url) return
  try {
    qrDataURL.value = await QRCodeLib.toDataURL(info.value.pay_url, { width: 200, margin: 1 })
  } catch {
    qrDataURL.value = ''
  }
}
watch(() => info.value.pay_url, renderQR)

const copied = ref(false)
function copyUrl() {
  if (!info.value.pay_url) return
  navigator.clipboard?.writeText(info.value.pay_url).then(() => {
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  }).catch(() => {})
}

async function saveName() {
  const name = codeName.value.trim()
  if (!name || busy.value) return
  busy.value = true
  try {
    await saveCodeName(name)
    toast.success('收款方名称已保存')
    editing.value = false
    info.value.codename = name
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    busy.value = false
  }
}

function downloadCode() {
  if (!qrDataURL.value) return
  const a = document.createElement('a')
  a.href = qrDataURL.value
  a.download = `收款码-${info.value.codename || '收款'}.png`
  a.click()
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="聚合收款码" subtitle="生成固定收款二维码，客户扫码后自选支付宝/微信/QQ 并输入金额付款">
      <div v-if="!info.open" class="mb-4 rounded bg-muted/40 px-3 py-2.5 text-sm text-muted-foreground">
        平台当前未开启聚合收款功能，如需开通请联系平台管理员。
      </div>
      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <!-- 左：配置 -->
        <div class="space-y-3.5">
          <div class="row-field">
            <label class="lbl">收款链接</label>
            <div class="flex flex-1 items-center gap-2">
              <input :value="info.pay_url" readonly class="field-input flex-1 bg-muted/40 font-mono text-[12px]" />
              <Button variant="outline" size="sm" @click="copyUrl">
                <component :is="copied ? Check : Copy" class="size-4" />
              </Button>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">收款方名称</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="codeName" :readonly="!editing" class="field-input flex-1" :class="!editing && 'bg-muted/40'" />
              <Button v-if="!editing" variant="outline" size="sm" @click="editing = true"><Pencil class="size-4" />修改</Button>
              <Button v-else size="sm" :disabled="busy" @click="saveName"><Check class="size-4" />保存</Button>
            </div>
          </div>
          <p class="text-xs text-muted-foreground">
            收款方名称会展示在客户的收银台顶部。将收款码打印张贴或分享给客户，扫码后进入收银台输入金额付款，款项实时到账。
          </p>
        </div>

        <!-- 右：预览 -->
        <div class="flex flex-col items-center justify-center gap-4 border border-border/70 bg-muted/20 py-8">
          <div class="text-sm font-medium">{{ info.codename }}</div>
          <div class="flex size-52 items-center justify-center border border-border bg-white p-2">
            <img v-if="qrDataURL" :src="qrDataURL" alt="收款码" class="size-full" />
            <span v-else class="text-xs text-muted-foreground/50">二维码生成中…</span>
          </div>
          <div class="text-xs text-muted-foreground">支持支付宝 / 微信 / QQ钱包</div>
          <Button variant="outline" size="sm" :disabled="!qrDataURL" @click="downloadCode"><Download class="size-4" />下载收款码</Button>
        </div>
      </div>
    </Panel>
  </div>
</template>
