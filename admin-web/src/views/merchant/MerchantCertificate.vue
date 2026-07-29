<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ShieldCheck, User, Building2, Send, CheckCircle2, Info, ScanLine, X, BadgeCheck, TrendingUp, History, ChevronDown } from 'lucide-vue-next'
import QRCodeLib from 'qrcode'
import { Panel, Button, Badge } from '@/components/ui'
import { fetchCertInfo, submitCert, type CertInfo } from '@/lib/api/merchantCenter'
import { recognizeLicense, recognizeIDCard } from '@/lib/api/ocr'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const info = ref<CertInfo | null>(null)
const busy = ref(false)
const certified = computed(() => info.value?.cert === 1)

// —— 腾讯云扫码人脸核身：弹二维码 + 轮询 cert 状态 ——
const scan = reactive({ open: false, qrDataURL: '' })
let pollTimer: ReturnType<typeof setInterval> | null = null

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function closeScan() {
  scan.open = false
  scan.qrDataURL = ''
  stopPolling()
}

// 打开扫码弹窗：把腾讯云 RedirectURL 渲染成二维码，商户微信扫脸；每 3s 轮询一次认证状态。
async function openScan(qrURL: string) {
  try {
    scan.qrDataURL = await QRCodeLib.toDataURL(qrURL, { width: 220, margin: 1 })
  } catch {
    toast.error('二维码生成失败')
    return
  }
  scan.open = true
  stopPolling()
  pollTimer = setInterval(async () => {
    try {
      const latest = await fetchCertInfo()
      if (latest.cert === 1) {
        info.value = latest
        closeScan()
        toast.success('实名认证已通过')
      }
    } catch {
      /* 轮询失败静默，等下一次 */
    }
  }, 3000)
}

onUnmounted(stopPolling)

const certType = ref<0 | 1>(0)
const form = reactive({ certname: '', certno: '', certcorp: '', certcorpno: '' })

// —— 认证变更记录（可折叠）——
// 后端暂无认证变更流水表，这里由当前认证态真实派生「认证完成」一条；
// 后续接后端历史表后替换为真实多条记录（见 docs-代理进件/待优化）。
const logOpen = ref(false)
const certLogs = computed(() => {
  if (!info.value || info.value.cert !== 1) return []
  return [
    {
      action: info.value.certtype === 1 ? '完成企业实名认证' : '完成个人实名认证',
      time: info.value.certtime || '—',
    },
  ]
})

// 升级为企业：已认证个人态点击后进入企业认证表单（强制企业类型，法人默认沿用已实名信息）。
const upgrading = ref(false)
function startUpgrade() {
  upgrading.value = true
  certType.value = 1
  // 法人姓名/证件号沿用已实名的个人信息（脱敏值仅展示，提交需重新填写真实值）。
  form.certname = ''
  form.certno = ''
  form.certcorp = ''
  form.certcorpno = ''
}
function cancelUpgrade() {
  upgrading.value = false
  certType.value = 0
}

// —— 证件识别（上传营业执照/身份证 → OCR 回填公司名/姓名/证件号，商户中心走 /merchant 前缀）——
const recognizing = reactive({ license: false, idcard: false })

function validImage(file: File): boolean {
  if (!/\.(jpe?g|png|bmp)$/i.test(file.name)) {
    toast.error('图片仅支持 JPG/PNG/BMP 格式')
    return false
  }
  if (file.size > 8 * 1024 * 1024) {
    toast.error('图片不能超过 8M，请压缩后重试')
    return false
  }
  return true
}

async function onPickLicense(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !validImage(file)) return
  recognizing.license = true
  try {
    const r = await recognizeLicense('/merchant', file)
    if (r.name) form.certcorp = r.name
    if (r.legal_person) form.certname = r.legal_person
    if (r.reg_number) form.certcorpno = r.reg_number
    toast.success('已识别营业执照并回填，请核对')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : 'OCR 识别失败，请手动填写')
  } finally {
    recognizing.license = false
  }
}

async function onPickIDCard(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !validImage(file)) return
  recognizing.idcard = true
  try {
    const r = await recognizeIDCard('/merchant', file, 'front')
    if (r.name) form.certname = r.name
    if (r.id_number) form.certno = r.id_number
    toast.success('已识别身份证并回填，请核对')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : 'OCR 识别失败，请手动填写')
  } finally {
    recognizing.idcard = false
  }
}

const canSubmit = computed(() => {
  if (certType.value === 1) return form.certcorp && form.certcorpno && form.certname && form.certno
  return form.certname && form.certno
})

async function load() {
  try {
    info.value = await fetchCertInfo()
    if (!upgrading.value) certType.value = (info.value.certtype === 1 ? 1 : 0)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载实名信息失败')
  }
}
onMounted(load)

async function submit() {
  if (!canSubmit.value || busy.value) return
  busy.value = true
  try {
    const res = await submitCert({
      certtype: certType.value,
      certname: form.certname.trim(),
      certno: form.certno.trim(),
      certcorp: form.certcorp.trim(),
      certcorpno: form.certcorpno.trim(),
    })
    if (res.async && res.qrurl) {
      // 腾讯云扫码人脸核身：弹二维码由商户微信扫脸，前端轮询认证状态。
      await openScan(res.qrurl)
    } else {
      toast.success('实名信息已提交')
      upgrading.value = false
      await load()
    }
  } catch (e) {
    // 后端对"第三方认证待凭证"返回业务错误，如实提示
    toast.error(e instanceof ApiError ? e.message : '提交失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 已认证：成功态（变更为企业进行中则让位给下方表单） -->
    <Panel v-if="certified && info && !upgrading" title="实名认证" subtitle="您的账户已完成实名认证">
      <!-- 个人认证可变更为企业：入口放标题行右侧 -->
      <template v-if="info.certtype === 0 && info.corpopen" #actions>
        <Button variant="outline" size="sm" @click="startUpgrade"><Building2 class="size-4" />变更为企业</Button>
      </template>
      <div class="grid gap-2.5 lg:grid-cols-[minmax(0,1fr)_19rem]">
        <!-- 左栏：状态 + 认证信息 -->
        <div class="space-y-2.5">
          <!-- 成功横幅：淡底 + 印章式验证徽章 -->
          <div class="flex items-center gap-3.5 bg-success/[0.07] px-4 py-4">
            <span class="relative flex size-12 shrink-0 items-center justify-center">
              <span class="absolute inset-0 rounded-full bg-success/[0.12]"></span>
              <span class="absolute inset-1 rounded-full bg-success/[0.16]"></span>
              <BadgeCheck class="relative size-6 text-success" :stroke-width="2.2" />
            </span>
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <span class="text-[15px] font-semibold text-success">{{ info.certtype === 1 ? '已完成企业认证' : '已完成个人认证' }}</span>
                <Badge variant="success">已生效</Badge>
              </div>
              <div class="mt-1 text-xs text-muted-foreground">认证时间 {{ info.certtime || '—' }}</div>
            </div>
          </div>
          <!-- 认证信息：宽屏两列铺开，窄屏单列 -->
          <dl class="grid grid-cols-1 gap-x-8 gap-y-px bg-muted/40 px-4 py-1 text-sm sm:grid-cols-2">
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">认证类型</dt><dd class="font-medium">{{ info.certtype === 1 ? '企业认证' : '个人认证' }}</dd></div>
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">认证方式</dt><dd class="font-medium">{{ info.method }}</dd></div>
            <div class="flex py-2.5" :class="info.certtype === 1 && 'border-b border-border/60'"><dt class="w-24 shrink-0 text-muted-foreground">{{ info.certtype === 1 ? '法人姓名' : '真实姓名' }}</dt><dd class="font-medium">{{ info.certname }}</dd></div>
            <div class="flex py-2.5" :class="info.certtype === 1 && 'border-b border-border/60'"><dt class="w-24 shrink-0 text-muted-foreground">证件号码</dt><dd class="font-mono font-medium tabular-nums">{{ info.certno }}</dd></div>
            <div v-if="info.certtype === 1 && info.certcorp" class="flex py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">企业名称</dt><dd class="font-medium">{{ info.certcorp }}</dd></div>
            <div v-if="info.certtype === 1 && info.certcorpno" class="flex py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">营业执照号</dt><dd class="font-mono font-medium tabular-nums">{{ info.certcorpno }}</dd></div>
          </dl>
        </div>
        <!-- 右栏：认证权益（实名后开通的能力，非装饰） -->
        <aside class="flex flex-col bg-muted/40 px-4 py-3.5">
          <div class="flex items-center gap-2 border-b border-border/60 pb-3 text-sm font-semibold text-foreground">
            <ShieldCheck class="size-4 text-success" />实名认证权益
          </div>
          <ul class="mt-3.5 space-y-3.5 text-xs">
            <li class="flex gap-2">
              <CheckCircle2 class="mt-0.5 size-3.5 shrink-0 text-success" />
              <div class="min-w-0">
                <div class="font-medium text-foreground">结算提现已开通</div>
                <div class="mt-0.5 leading-relaxed text-muted-foreground">收款资金可结算至绑定账户</div>
              </div>
            </li>
            <li class="flex gap-2">
              <CheckCircle2 class="mt-0.5 size-3.5 shrink-0 text-success" />
              <div class="min-w-0">
                <div class="font-medium text-foreground">资金接口已开通</div>
                <div class="mt-0.5 leading-relaxed text-muted-foreground">可调用代付、退款等资金类 API</div>
              </div>
            </li>
            <li class="flex gap-2">
              <component
                :is="info.certtype === 1 ? CheckCircle2 : TrendingUp"
                class="mt-0.5 size-3.5 shrink-0"
                :class="info.certtype === 1 ? 'text-success' : 'text-muted-foreground/70'"
              />
              <div v-if="info.certtype === 1" class="min-w-0">
                <div class="font-medium text-foreground">企业收款额度</div>
                <div class="mt-0.5 leading-relaxed text-muted-foreground">享更高收款额度与对公结算能力</div>
              </div>
              <div v-else class="min-w-0">
                <div class="font-medium text-foreground">可升级企业认证</div>
                <div class="mt-0.5 leading-relaxed text-muted-foreground">提升收款额度、支持对公结算</div>
              </div>
            </li>
          </ul>
        </aside>
      </div>
    </Panel>

    <!-- 认证变更记录（独立 Panel，与实名认证分开） -->
    <Panel v-if="certified && info && !upgrading" flush no-header>
      <button
        class="flex w-full items-center gap-2 px-6 py-3.5 text-left text-sm font-medium text-foreground transition-colors hover:bg-muted/30"
        @click="logOpen = !logOpen"
      >
        <History class="size-4 text-muted-foreground" />
        <span>认证变更记录</span>
        <span v-if="certLogs.length" class="text-xs font-normal text-muted-foreground">（{{ certLogs.length }}）</span>
        <ChevronDown class="ml-auto size-4 text-muted-foreground transition-transform duration-200" :class="logOpen && 'rotate-180'" />
      </button>
      <div v-if="logOpen" class="border-t border-border/60 px-6 py-4">
        <p v-if="!certLogs.length" class="py-2 text-center text-xs text-muted-foreground">暂无变更记录</p>
        <ol v-else class="relative space-y-4 pl-4">
          <li v-for="(lg, i) in certLogs" :key="i" class="relative">
            <span class="absolute -left-4 top-1 size-2 rounded-full" :class="i === 0 ? 'bg-success' : 'bg-border'"></span>
            <span v-if="i < certLogs.length - 1" class="absolute -left-[13px] top-3 h-full w-px bg-border/70"></span>
            <div class="text-sm text-foreground">{{ lg.action }}</div>
            <div class="mt-0.5 font-mono text-xs tabular-nums text-muted-foreground">{{ lg.time }}</div>
          </li>
        </ol>
      </div>
    </Panel>

    <!-- 未认证 / 变更为企业：提交表单 -->
    <Panel v-else title="实名认证" :subtitle="upgrading ? '变更为企业认证：请补充企业主体与法人信息' : (info ? `认证方式：${info.method}` : '')">
      <!-- 变更为企业：取消入口放标题行右侧 -->
      <template v-if="upgrading" #actions>
        <Button variant="ghost" size="sm" @click="cancelUpgrade"><X class="size-4" />取消变更</Button>
      </template>
      <div class="max-w-2xl space-y-4">
        <!-- 变更为企业：主体类型已锁定提示 -->
        <div v-if="upgrading" class="flex items-center gap-2 border-l-2 border-primary bg-primary/[0.05] px-4 py-2.5 text-xs text-muted-foreground">
          <Building2 class="size-4 shrink-0 text-primary" />
          <span>正在从个人认证变更为企业认证，主体类型已锁定为企业，请补充下方企业与法人信息。</span>
        </div>
        <!-- 主体类型（变更中锁定企业，不显示选择器） -->
        <div v-if="info?.corpopen && !upgrading" class="grid grid-cols-2 gap-3">
          <button
            v-for="opt in [
              { t: 0 as const, icon: User, name: '个人认证', desc: '以个人身份实名' },
              { t: 1 as const, icon: Building2, name: '企业认证', desc: '以企业主体实名' },
            ]"
            :key="opt.t"
            class="relative flex items-center gap-3 border px-4 py-3.5 text-left transition-colors"
            :class="certType === opt.t
              ? 'border-primary bg-primary/[0.04]'
              : 'border-border hover:border-primary/40'"
            @click="certType = opt.t"
          >
            <component
              :is="opt.icon"
              class="size-5 shrink-0"
              :class="certType === opt.t ? 'text-primary' : 'text-muted-foreground'"
            />
            <div class="min-w-0">
              <div class="text-sm font-medium" :class="certType === opt.t && 'text-primary'">{{ opt.name }}</div>
              <div class="mt-0.5 text-xs text-muted-foreground">{{ opt.desc }}</div>
            </div>
            <CheckCircle2
              v-if="certType === opt.t"
              class="absolute right-3.5 top-3.5 size-4 text-primary"
            />
          </button>
        </div>

        <!-- 企业信息 -->
        <template v-if="certType === 1">
          <div class="row-field">
            <label class="lbl">营业执照</label>
            <label class="media-btn flex-1 cursor-pointer">
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="recognizing.license" @change="onPickLicense" />
              <ScanLine class="size-3.5" />
              {{ recognizing.license ? '识别中…' : '上传营业执照自动识别公司名/法人' }}
            </label>
          </div>
          <div class="row-field">
            <label class="lbl">公司名称</label>
            <input v-model="form.certcorp" class="field-input flex-1" placeholder="营业执照上的公司全称" />
          </div>
          <div class="row-field">
            <label class="lbl">营业执照号</label>
            <input v-model="form.certcorpno" class="field-input flex-1" placeholder="统一社会信用代码（18 位）" />
          </div>
          <div class="border-t border-border/60 pt-3.5 text-sm font-medium text-muted-foreground">法人信息</div>
        </template>

        <div class="row-field">
          <label class="lbl">身份证识别</label>
          <label class="media-btn flex-1 cursor-pointer">
            <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="recognizing.idcard" @change="onPickIDCard" />
            <ScanLine class="size-3.5" />
            {{ recognizing.idcard ? '识别中…' : '上传身份证（人像面）自动识别姓名/证件号' }}
          </label>
        </div>
        <div class="row-field">
          <label class="lbl">{{ certType === 1 ? '法人姓名' : '真实姓名' }}</label>
          <input v-model="form.certname" class="field-input flex-1" placeholder="请输入真实姓名" />
        </div>
        <div class="row-field">
          <label class="lbl">身份证号</label>
          <input v-model="form.certno" class="field-input flex-1" placeholder="18 位身份证号码" />
        </div>

        <div class="flex gap-2.5 bg-muted/40 px-3.5 py-3 text-xs leading-relaxed text-muted-foreground">
          <Info class="mt-0.5 size-4 shrink-0 text-muted-foreground/70" />
          <p>
            实名认证需通过第三方（支付宝/微信/阿里云）核验，工本费
            <b class="text-foreground">¥{{ info?.certmoney ?? 0 }}</b>，认证成功才扣，失败不扣费。
            当前第三方认证渠道待接入凭证，提交后信息将暂存待核验。
          </p>
        </div>
        <div class="border-t border-border/60 pt-4">
          <Button :disabled="!canSubmit || busy" @click="submit"><Send />提交实名信息</Button>
        </div>
      </div>
    </Panel>

    <!-- 腾讯云扫码人脸核身弹窗 -->
    <div v-if="scan.open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" @click.self="closeScan">
      <div class="w-full max-w-sm bg-background shadow-lg">
        <div class="flex items-center justify-between border-b border-border/60 px-4 py-3">
          <span class="text-sm font-semibold">微信扫码实名认证</span>
          <button class="text-muted-foreground hover:text-foreground" @click="closeScan"><X class="size-4" /></button>
        </div>
        <div class="flex flex-col items-center gap-3 px-6 py-6">
          <img v-if="scan.qrDataURL" :src="scan.qrDataURL" alt="实名认证二维码" class="size-52" />
          <p class="text-center text-sm text-muted-foreground leading-relaxed">
            请用<b class="text-foreground">微信</b>扫描上方二维码，按提示完成人脸活体核身。
          </p>
          <div class="flex items-center gap-2 text-xs text-muted-foreground">
            <span class="inline-block size-1.5 animate-pulse rounded-full bg-primary"></span>
            正在等待认证完成，通过后本页自动更新…
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
