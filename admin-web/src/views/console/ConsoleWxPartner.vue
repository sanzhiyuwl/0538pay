<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { Upload, CheckCircle2, X } from 'lucide-vue-next'
import { Panel, Button, Badge } from '@/components/ui'
import { getWxPartner, saveWxPartner, type WxPartnerView } from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 明文字段（非机密：商户号/序列号/appid/公钥ID），直接编辑回填。
const form = reactive({
  sp_mchid: '',
  sp_appid: '',
  serial_no: '',
  public_key_id: '',
})
// 敏感字段（私钥/公钥/APIv3 密钥）后端只回脱敏视图，不回原文。
// 私钥/公钥由文件上传读入；APIv3 密钥手输。三者均"留空=不改"，非空才提交覆盖。
const view = ref<WxPartnerView | null>(null)
const privateKeyText = ref('') // 新上传的私钥 PEM 全文（空=不改）
const publicKeyText = ref('') // 新上传的公钥 PEM 全文（空=不改）
const privateKeyFile = ref('') // 已选私钥文件名（仅展示）
const publicKeyFile = ref('') // 已选公钥文件名（仅展示）
const apiv3Key = ref('') // 新 APIv3 密钥（空=不改）

const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const v = await getWxPartner()
    view.value = v
    form.sp_mchid = v.sp_mchid
    form.sp_appid = v.sp_appid
    form.serial_no = v.serial_no
    form.public_key_id = v.public_key_id
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载服务商凭证失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

// 读取 .pem 文件文本到内存（不经公开上传通道，随保存走鉴权 PUT 存入后端 secrets/）。
function onPickKey(e: Event, which: 'private' | 'public') {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (file.size > 64 * 1024) {
    toast.error('密钥文件过大（正常 PEM 不超过 64KB），请确认选对了文件')
    input.value = ''
    return
  }
  const reader = new FileReader()
  reader.onload = () => {
    const text = String(reader.result || '')
    if (!text.includes('-----BEGIN')) {
      toast.error('文件不是有效的 PEM（未找到 -----BEGIN 标记）')
      return
    }
    if (which === 'private') {
      privateKeyText.value = text
      privateKeyFile.value = file.name
    } else {
      publicKeyText.value = text
      publicKeyFile.value = file.name
    }
    toast.success(`已读取 ${file.name}，保存后生效`)
  }
  reader.onerror = () => toast.error('读取文件失败')
  reader.readAsText(file)
  input.value = '' // 允许重选同名文件
}

function clearPicked(which: 'private' | 'public') {
  if (which === 'private') {
    privateKeyText.value = ''
    privateKeyFile.value = ''
  } else {
    publicKeyText.value = ''
    publicKeyFile.value = ''
  }
}

async function save() {
  saving.value = true
  try {
    const v = await saveWxPartner({
      sp_mchid: form.sp_mchid,
      sp_appid: form.sp_appid,
      serial_no: form.serial_no,
      public_key_id: form.public_key_id,
      // 留空=不改，非空才提交
      private_key: privateKeyText.value || undefined,
      public_key: publicKeyText.value || undefined,
      apiv3_key: apiv3Key.value || undefined,
    })
    view.value = v
    // 清掉本地暂存的敏感原文（已落后端，前端不留）
    clearPicked('private')
    clearPicked('public')
    apiv3Key.value = ''
    toast.success('已保存')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="微信服务商凭证" subtitle="进件以「平台服务商」身份调微信，全平台唯一一套；与微信服务商模式收单共用同一份">
      <template #actions>
        <Badge :variant="view?.configured ? 'success' : 'warning'">
          {{ view?.configured ? '已配齐' : '未配齐' }}
        </Badge>
      </template>

      <div v-if="loading" class="py-10 text-center dim">加载中…</div>
      <div v-else class="max-w-2xl space-y-3.5">
        <div class="rounded bg-muted/40 px-3 py-2 text-[11px] text-muted-foreground">
          必填四项：服务商商户号、证书序列号、商户 API 私钥、微信支付公钥。私钥 / 公钥经文件上传，落后端
          secrets/ 目录（不对外访问、不入代码库）；APIv3 密钥仅存后端。敏感字段保存后一律不回显原文，
          只显示「已配置 + 内容指纹」供核对，留空即保持原值不变。
        </div>

        <div class="row-field">
          <label class="lbl">服务商商户号<span class="text-destructive">*</span></label>
          <input v-model="form.sp_mchid" placeholder="微信支付服务商平台的商户号 sp_mchid" class="field-input flex-1" />
        </div>

        <div class="row-field">
          <label class="lbl">服务商 AppID</label>
          <input v-model="form.sp_appid" placeholder="服务商公众号 / 小程序 AppID（部分场景需要，可留空）" class="field-input flex-1" />
        </div>

        <div class="row-field">
          <label class="lbl">证书序列号<span class="text-destructive">*</span></label>
          <input v-model="form.serial_no" placeholder="商户 API 证书的序列号（发请求签名用）" class="field-input flex-1" />
        </div>

        <!-- 商户 API 私钥：文件上传（说明并入按钮旁提示） -->
        <div class="row-field items-start">
          <label class="lbl pt-1.5">商户 API 私钥<span class="text-destructive">*</span></label>
          <div class="flex flex-1 flex-wrap items-center gap-2">
            <label class="key-upload">
              <Upload class="size-3.5" />
              选择 .pem 文件
              <input type="file" accept=".pem,.key,.txt" class="hidden" @change="(e) => onPickKey(e, 'private')" />
            </label>
            <span v-if="privateKeyFile" class="key-tag key-tag-picked">
              <CheckCircle2 class="size-3.5" />
              {{ privateKeyFile }} · 保存后生效
              <button type="button" class="key-tag-close" title="撤销" @click="clearPicked('private')"><X class="size-3" /></button>
            </span>
            <span v-else-if="view?.has_private_key" class="key-tag key-tag-set">
              <CheckCircle2 class="size-3.5" />
              已配置 · 指纹 {{ view.private_key_fp }}
            </span>
            <span v-else class="key-tag key-tag-unset">未配置</span>
            <span class="key-hint">apiclient_key.pem，上传后不回显原文</span>
          </div>
        </div>

        <!-- 微信支付公钥：文件上传（说明并入按钮旁提示） -->
        <div class="row-field items-start">
          <label class="lbl pt-1.5">微信支付公钥<span class="text-destructive">*</span></label>
          <div class="flex flex-1 flex-wrap items-center gap-2">
            <label class="key-upload">
              <Upload class="size-3.5" />
              选择 .pem 文件
              <input type="file" accept=".pem,.key,.txt" class="hidden" @change="(e) => onPickKey(e, 'public')" />
            </label>
            <span v-if="publicKeyFile" class="key-tag key-tag-picked">
              <CheckCircle2 class="size-3.5" />
              {{ publicKeyFile }} · 保存后生效
              <button type="button" class="key-tag-close" title="撤销" @click="clearPicked('public')"><X class="size-3" /></button>
            </span>
            <span v-else-if="view?.has_public_key" class="key-tag key-tag-set">
              <CheckCircle2 class="size-3.5" />
              已配置 · 指纹 {{ view.public_key_fp }}
            </span>
            <span v-else class="key-tag key-tag-unset">未配置</span>
            <span class="key-hint">平台证书 PEM，验签 + 敏感字段加密用</span>
          </div>
        </div>

        <div class="row-field">
          <label class="lbl">公钥 / 证书序列号</label>
          <input v-model="form.public_key_id" placeholder="Wechatpay-Serial，发加密请求时标识加密用证书（可选）" class="field-input flex-1" />
        </div>

        <!-- APIv3 密钥：password 输入 + 脱敏状态 -->
        <div class="row-field">
          <label class="lbl">APIv3 密钥</label>
          <input
            v-model="apiv3Key"
            type="password"
            :placeholder="view?.has_apiv3_key ? '已配置（指纹 ' + view.apiv3_key_fp + '），留空则不改' : '32 位，微信回调 AES-GCM 解密用（敏感，不回显）'"
            class="field-input flex-1"
            autocomplete="new-password"
          />
        </div>

        <div class="pt-2">
          <Button :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存凭证' }}</Button>
        </div>
      </div>
    </Panel>
  </div>
</template>

<style scoped>
/* 上传按钮：直角描边 + 浅灰底，图标+文案，hover 提亮边框（不用 color-mix，避免发粉） */
.key-upload {
  display: inline-flex;
  cursor: pointer;
  align-items: center;
  gap: 0.35rem;
  border: 1px solid var(--border);
  background: var(--muted);
  padding: 0.36rem 0.75rem;
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--foreground);
  transition: border-color 0.15s, background 0.15s;
}
.key-upload:hover {
  border-color: var(--primary);
  color: var(--primary);
}
/* 状态胶囊：淡底描边配色直接抄 Badge 的 Element UI 精确 hex */
.key-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  border: 1px solid transparent;
  padding: 0.28rem 0.6rem;
  font-size: 0.75rem;
  line-height: 1;
}
.key-tag-set,
.key-tag-picked {
  border-color: #e1f3d8;
  background: #f0f9eb;
  color: #67c23a;
}
.key-tag-unset {
  border-color: #fde2e2;
  background: #fef0f0;
  color: #f56c6c;
}
.key-tag-close {
  display: inline-flex;
  cursor: pointer;
  align-items: center;
  margin-left: 0.15rem;
  border-radius: 9999px;
  color: currentColor;
  opacity: 0.7;
}
.key-tag-close:hover {
  opacity: 1;
}
/* 文件上传行的补充说明（无法用 placeholder，故放按钮旁） */
.key-hint {
  font-size: 0.6875rem;
  color: var(--muted-foreground);
}
</style>
