<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  Plus, FileText, CheckCircle2, X, RefreshCw, QrCode, Trash2, Copy,
  PencilLine, Send, Clock, ShieldCheck, ShieldAlert, ChevronRight, AlertTriangle, ChevronDown, ExternalLink,
} from 'lucide-vue-next'
import QRCodeLib from 'qrcode'
import { Panel, Button, Badge, Drawer, Select } from '@/components/ui'
import WechatIcon from '@/components/site/icons/WechatIcon.vue'
import EnrollMaterialDrawer from '@/views/enroll/EnrollMaterialDrawer.vue'
import { merchantGetChannelControl, type ChannelControlView } from '@/lib/api/merchantChannelControl'
import {
  myEnrollableChannels,
  myListChannelEnrolls,
  myGetChannelEnroll,
  myCreateChannelEnroll,
  myGetChannelEnrollMaterial,
  myFillChannelEnrollMaterial,
  mySubmitChannelEnroll,
  mySyncChannelEnroll,
  myDeleteChannelEnroll,
  myUploadChannelEnrollMedia,
  myUploadChannelEnrollVideo,
  type EnrollableChannel,
  type ChannelEnrollView,
  type ChannelEnrollDetail,
} from '@/lib/api/channelEnroll'
import { recognizeLicense, recognizeIDCard } from '@/lib/api/ocr'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'

const toast = useToast()
const confirm = useConfirm()
const router = useRouter()

const channels = ref<EnrollableChannel[]>([])
const list = ref<ChannelEnrollView[]>([])
const loading = ref(false)

// 进件四步流程（商户视角）：填料 → 提交 → 审核 → 开通。空态引导与详情步骤条共用。
const STEPS = [
  { key: 'fill', label: '填写资料', icon: PencilLine, hint: '主体、结算账户、经营信息' },
  { key: 'submit', label: '提交审核', icon: Send, hint: '资料加密直传机构' },
  { key: 'review', label: '机构审核', icon: Clock, hint: '通常 1~2 个工作日' },
  { key: 'open', label: '开通收款', icon: ShieldCheck, hint: '拿到专属子商户号' },
] as const

// 副标题「申请流程」展示用步骤（比状态机 STEPS 多一个前置「选择进件通道」，仅用于说明引导，不参与状态映射）。
const FLOW_STEPS = [
  '选择通道',
  '填写资料',
  '平台初审',
  '机构审核',
  '开通收款',
] as const

// 申请流程说明区展开/收起（默认展开；收起时仅保留合规提示，其余步骤/说明/费率收起）。
const flowOpen = ref(true)

// 微信审核通过后的中间态：机构审核已完成，进入「待账户验证 / 待签约 / 开通权限中」（就差超管扫码激活）。
// 后端把这些都压成本地 submitted，需靠 wx_state 原始码把进度推进过「机构审核」步。
const WX_AFTER_AUDIT = ['APPLYMENT_STATE_TO_BE_CONFIRMED', 'APPLYMENT_STATE_TO_BE_SIGNED', 'APPLYMENT_STATE_SIGNING']

// 状态 → 当前所处步骤序号（0-based）：草稿=填料;审核中=审核;已开通=开通;驳回=退回填料。
// 感知 wx_state：submitted 但微信已过审进入待签约/待验证/开通中 → 推进到「开通收款(待激活)」步。
function stepIndex(status: string, wxState?: string): number {
  switch (status) {
    case 'draft': return 0
    case 'rejected': return 0 // 驳回退回到填料步（修改重提）
    case 'submitted': case 'pending':
      return wxState && WX_AFTER_AUDIT.includes(wxState) ? 3 : 2
    case 'approved': return 3
    default: return 0
  }
}
// 某单某步的状态：done=已完成 / current=进行中 / error=驳回 / todo=未到。
function stepState(e: ChannelEnrollView, idx: number): 'done' | 'current' | 'error' | 'todo' {
  const cur = stepIndex(e.status, e.wx_state)
  if (e.status === 'approved') return 'done' // 已开通：四步全绿
  if (e.status === 'rejected' && idx === 0) return 'error'
  if (idx < cur) return 'done'
  if (idx === cur) return 'current'
  return 'todo'
}

// 进度列当前步文字：submitted 阶段直接显示微信真实态（审核中/待账户验证/待签约/开通权限中），
// 避免把已过审的「待签约」错显成「机构审核」；其余状态回落到本地步骤文案。
function stepLabel(e: ChannelEnrollView): string {
  if ((e.status === 'submitted' || e.status === 'pending') && e.wx_state_text && e.wx_state_text !== '—') {
    return e.wx_state_text
  }
  return STEPS[stepIndex(e.status, e.wx_state)]?.label ?? ''
}

// 状态徽章样式映射（submitted=审核中；pending 为历史兼容态）。
const statusVariant: Record<string, 'default' | 'success' | 'warning' | 'destructive' | 'muted'> = {
  draft: 'muted',
  pending: 'warning',
  submitted: 'warning',
  approved: 'success',
  rejected: 'destructive',
}

async function load() {
  loading.value = true
  try {
    const [ch, res] = await Promise.all([myEnrollableChannels(), myListChannelEnrolls({ pagesize: 100 })])
    channels.value = ch.list
    list.value = res.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

// —— 建单：选服务商通道 ——
const createOpen = ref(false)
const createForm = reactive({ channel_id: 0, merchant_name: '', contact_phone: '' })
const creating = ref(false)

// 服务商通道下拉选项（喂给统一 Select 组件，替代原生 select 保持全站风格一致）。
const channelOptions = computed(() => channels.value.map((ch) => ({ value: ch.id, label: ch.name })))

function openCreate() {
  createForm.channel_id = channels.value[0]?.id ?? 0
  createForm.merchant_name = ''
  createForm.contact_phone = ''
  createOpen.value = true
}

async function doCreate() {
  if (!createForm.channel_id) {
    toast.error('请选择要进件的服务商通道')
    return
  }
  creating.value = true
  try {
    const r = await myCreateChannelEnroll({
      channel_id: createForm.channel_id,
      merchant_name: createForm.merchant_name.trim(),
      contact_phone: createForm.contact_phone.trim(),
    })
    createOpen.value = false
    toast.success('进件单已创建，请填写资料')
    await load()
    await openFill(r.id)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '创建失败')
  } finally {
    creating.value = false
  }
}

// —— 填料：复用共享的 EnrollMaterialDrawer（微信 applyment4sub 完整五大块 + 图片/视频上传 + OCR）——
// ★复用不走代理端文件：注入的是商户线自己的 API 适配器（/merchant/channel-enrolls/*）。
const materialOpen = ref(false)
const activeId = ref<number | null>(null)
const activeName = ref('')

// OCR 走商户端基址 /merchant（后端已开 /merchant/ocr/license|idcard）。
const ocrLicense = (file: File) => recognizeLicense('/merchant', file)
const ocrIdcard = (file: File, side?: 'front' | 'back') => recognizeIDCard('/merchant', file, side)

async function openFill(id: number) {
  activeId.value = id
  const row = list.value.find((e) => e.id === id)
  activeName.value = row?.channel_name || ''
  materialOpen.value = true
}

async function onMaterialSaved() {
  await load()
}

// 填料抽屉保存后若要提交微信：由抽屉「保存」完成资料落库，这里单独提供「提交微信」动作。
async function submitToWx(id: number) {
  if (!(await confirm('确认把资料提交微信进件？提交后进入微信审核，可稍后刷新查看进度。', { title: '提交微信进件' })))
    return
  try {
    const r = await mySubmitChannelEnroll(id)
    toast.success(`已提交微信审核（${r.wx_state || '审核中'}）`)
    materialOpen.value = false
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '提交失败')
  }
}

// 删除进件单（提交前放弃，仅草稿/被驳回单）。删除成功后关详情抽屉、刷新列表。
const deleting = ref(0)
async function deleteEnroll(id: number) {
  if (
    !(await confirm('确认删除这条进件单？删除后需重新发起进件，已填写的资料将一并清除。', {
      title: '删除进件单',
      confirmText: '删除',
      danger: true,
    }))
  )
    return
  deleting.value = id
  try {
    await myDeleteChannelEnroll(id)
    toast.success('进件单已删除')
    detailOpen.value = false
    materialOpen.value = false
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  } finally {
    deleting.value = 0
  }
}

// 刷新微信进件状态（审核中→开通/驳回）。
const syncing = ref(0)
async function syncState(id: number) {
  syncing.value = id
  try {
    const r = await mySyncChannelEnroll(id)
    if (r.status === 'approved') toast.success(`已开通，子商户号 ${r.sub_mchid}`)
    else if (r.status === 'rejected') toast.error(`微信驳回：${r.reject_reason || '请查看详情'}`)
    else toast.info(`当前状态：${r.wx_state_text || '处理中'}`)
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '刷新失败')
  } finally {
    syncing.value = 0
  }
}

// —— 详情查看（非可填态：审核中/已开通/已驳回）——
const detailOpen = ref(false)
const detail = ref<ChannelEnrollDetail | null>(null)
async function openDetail(id: number) {
  try {
    detail.value = await myGetChannelEnroll(id)
    detailOpen.value = true
    if (detail.value.status === 'approved') loadControlView(id)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载详情失败')
  }
}
const detailMaterial = computed(() => detail.value?.material)

// 业务受限就地快照（风控第二段商户端，两处不重复造轮子：与 /m/channel-controls 共用同一份快照）。
const controlView = ref<ChannelControlView | null>(null)
async function loadControlView(id: number) {
  controlView.value = null
  try {
    const res = await merchantGetChannelControl(id)
    controlView.value = res.view
  } catch {
    /* 查询失败静默，详情页不因此块阻塞 */
  }
}
function gotoChannelControls() {
  router.push('/m/channel-controls')
}

// 详情抽屉顶部步骤条：复用列表的 stepIndex/stepState 语义，只是数据源换成 detail。
const detailStepIdx = computed(() => (detail.value ? stepIndex(detail.value.status, detail.value.wx_state) : 0))
function detailStepState(idx: number): 'done' | 'current' | 'error' | 'todo' {
  const d = detail.value
  if (!d) return 'todo'
  if (d.status === 'approved') return 'done'
  if (d.status === 'rejected' && idx === 0) return 'error'
  if (idx < detailStepIdx.value) return 'done'
  if (idx === detailStepIdx.value) return 'current'
  return 'todo'
}

// 待签约二维码：内嵌显示，超管扫码即签约。与后台 admin/channel-enrolls 同款逻辑。
// 微信两种 sign_url 形态：
//   ① 图片直链 `mp.weixin.qq.com/cgi-bin/showqrcode?ticket=...` —— 本身就是二维码 PNG，直接当图显示；
//   ② 普通签约页链接 —— 用 qrcode 库把 URL 编成二维码 PNG。
const signQrImageURL = ref('') // 直链图片形态
const signQrDataURL = ref('')  // 前端编码 dataURL 形态
watch(
  () => detail.value?.sign_url,
  async (url) => {
    signQrImageURL.value = ''
    signQrDataURL.value = ''
    if (!url) return
    if (/mp\.weixin\.qq\.com\/cgi-bin\/showqrcode/i.test(url)) {
      signQrImageURL.value = url
      return
    }
    try {
      signQrDataURL.value = await QRCodeLib.toDataURL(url, { width: 176, margin: 1 })
    } catch {
      /* 生成失败则回退到文字链接 */
    }
  },
  { immediate: true },
)
async function copySignURL() {
  if (!detail.value?.sign_url) return
  try {
    await navigator.clipboard.writeText(detail.value.sign_url)
    toast.success('签约链接已复制')
  } catch {
    toast.error('复制失败，请手动选择链接')
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 申请流程：步骤作副标题，说明在正文，可折叠（收起仅留合规提示） -->
    <Panel title="申请流程">
      <template #title-extra>
        <div v-show="flowOpen" class="ml-1 flex items-center gap-1 overflow-x-auto text-xs">
          <template v-for="(label, i) in FLOW_STEPS" :key="label">
            <div class="flex items-center gap-1.5 whitespace-nowrap text-muted-foreground">
              <span class="flex size-5 items-center justify-center rounded-full bg-muted/60 text-[11px] font-medium tabular-nums text-muted-foreground/80">{{ i + 1 }}</span>
              <span>{{ label }}</span>
            </div>
            <ChevronRight v-if="i < FLOW_STEPS.length - 1" class="size-3.5 shrink-0 text-muted-foreground/40" />
          </template>
        </div>
      </template>
      <template #actions>
        <button
          type="button"
          class="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
          @click="flowOpen = !flowOpen"
        >
          {{ flowOpen ? '收起' : '展开' }}
          <ChevronDown class="size-4 transition-transform" :class="{ '-rotate-180': flowOpen }" />
        </button>
      </template>

      <!-- 合规提示：始终显示（收起也保留） -->
      <p class="flex items-start gap-1.5 text-xs leading-relaxed text-destructive">
        <AlertTriangle class="mt-0.5 size-3.5 shrink-0" />
        <span><b>合规提示：</b>本平台严禁任何违法违规业务，一经发现将立即冻结账号、账户余额不予退还，情节严重者移交公安机关处理。出租、出售、出借银行账户、支付账户或支付接口均需承担相应法律责任并受惩戒——网络非法外之地，请勿以身试法！</span>
      </p>

      <!-- 可折叠区：进件说明 + 费率 -->
      <template v-if="flowOpen">
        <div class="mt-3 border-t border-border/50 pt-3">
          <p class="flex items-start gap-1.5 text-xs leading-relaxed text-muted-foreground">
            <ShieldCheck class="mt-0.5 size-3.5 shrink-0" />
            <span>部分服务商通道需先进件开通，审核通过后系统自动为您开通；收款资金由收单机构直接清算至您的<b class="text-foreground">专属子商户号</b>，平台全程不经手资金，敏感信息加密直传不留明文。</span>
          </p>
        </div>
        <div class="mt-3 border-t border-border/50 pt-3">
          <p class="flex items-start gap-1.5 text-xs leading-relaxed text-muted-foreground">
            <WechatIcon class="mt-0.5 size-4 shrink-0 text-[#07C160]" />
            <span><b class="text-foreground">微信特约商户费率：</b>进件开通后，您的微信收款将按微信支付官方特约商户费率结算，标准费率低至 <b class="text-foreground">0.2%</b>（最终以微信支付官方根据您的实际经营类目核定为准）。该手续费由微信支付官方按每笔交易实收金额收取、并从交易金额中自动扣除后清算至您的专属子商户号，平台不额外加收任何通道费用。</span>
          </p>
        </div>
      </template>
    </Panel>

    <Panel title="通道进件" subtitle="先进件开通，收款直清至您的专属子商户号">
      <template #actions>
        <Button v-if="channels.length && list.length" size="sm" @click="openCreate"><Plus class="size-4" />发起进件</Button>
      </template>

      <!-- 无可进件通道 -->
      <p v-if="!channels.length && !loading" class="py-8 text-center text-sm text-muted-foreground">
        当前没有需要进件的服务商通道。
      </p>

      <!-- 空态：清晰引导 + 四步图 + 发起进件 -->
      <div v-else-if="!list.length && !loading" class="flex flex-col items-center gap-5 py-12">
        <div class="flex size-14 items-center justify-center rounded-full bg-muted/60 text-muted-foreground">
          <FileText class="size-7" />
        </div>
        <div class="text-center">
          <div class="text-sm font-medium">还没有进件记录</div>
          <p class="mt-1 text-xs text-muted-foreground">发起进件，开通属于您自己的收款子商户号</p>
        </div>
        <!-- 四步大图引导 -->
        <div class="flex items-start gap-2">
          <template v-for="(s, i) in STEPS" :key="s.key">
            <div class="flex w-24 flex-col items-center gap-1.5 text-center">
              <div class="flex size-10 items-center justify-center rounded-full bg-muted/60 text-muted-foreground">
                <component :is="s.icon" class="size-[18px]" />
              </div>
              <div class="text-xs font-medium">{{ s.label }}</div>
              <div class="text-[11px] leading-tight text-muted-foreground">{{ s.hint }}</div>
            </div>
            <ChevronRight v-if="i < STEPS.length - 1" class="mt-3 size-4 shrink-0 text-muted-foreground/40" />
          </template>
        </div>
        <Button v-if="channels.length" @click="openCreate"><Plus class="size-4" />发起进件</Button>
      </div>

      <!-- 精简进件列表：申请单号 / 进件通道 / 进度 / 说明 / 商户号 / 收款开关 / 下一步 -->
      <!-- 精简进件列表：申请单号 / 进件通道 / 进度 / 状态 / 子商户号 / 操作 -->
      <!-- 底层样式优化：表头加 bg-muted/40 浅灰底带（去掉表头下边线，靠色差区分），仅作用于本表不改全站 .tbl -->
      <table v-else class="tbl w-full table-fixed [&_thead_th]:border-b-0 [&_thead_th]:bg-muted/40 [&_thead_th]:py-2.5 [&_thead_th:first-child]:pl-3 [&_tbody_td:first-child]:pl-3 [&_tbody_td]:py-3.5 [&_tbody_td]:align-middle">
        <thead>
          <tr>
            <th class="w-[13%]">申请单号</th>
            <th class="w-[20%]">进件通道</th>
            <th class="w-[17%]">进度</th>
            <th class="w-[21%]">状态</th>
            <th class="w-[13%]">子商户号</th>
            <th class="w-[16%] !pl-10">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in list" :key="e.id">
            <!-- 申请单号 -->
            <td>
              <div class="font-mono text-[13px] tabular-nums tracking-tight" :class="e.enroll_no ? 'text-foreground' : 'text-muted-foreground/50'">{{ e.enroll_no || '—' }}</div>
            </td>
            <!-- 进件通道：仅通道名 -->
            <td>
              <div class="truncate font-medium leading-tight" :title="e.channel_name">{{ e.channel_name || '—' }}</div>
            </td>
            <!-- 进度：徽章 + 迷你步骤条 + 当前步文字 -->
            <td>
              <div class="flex items-center gap-2">
                <span class="inline-flex w-14 shrink-0">
                  <Badge :variant="statusVariant[e.status] || 'muted'">{{ e.status_text }}</Badge>
                </span>
                <div class="flex shrink-0 items-center gap-1">
                  <span
                    v-for="(s, i) in STEPS"
                    :key="s.key"
                    class="size-1.5 rounded-full transition-colors"
                    :class="{
                      'bg-success': stepState(e, i) === 'done',
                      'bg-primary': stepState(e, i) === 'current',
                      'bg-destructive': stepState(e, i) === 'error',
                      'bg-border': stepState(e, i) === 'todo',
                    }"
                    :title="s.label"
                  />
                </div>
                <span class="shrink-0 whitespace-nowrap text-xs text-muted-foreground">{{ stepLabel(e) }}</span>
              </div>
            </td>
            <!-- 说明：单列承载微信状态 / 驳回原因（其余状态占位弱化） -->
            <td>
              <div v-if="e.status === 'submitted' && e.wx_state_text" class="flex gap-1.5 text-xs">
                <span class="shrink-0 text-muted-foreground/70">审核状态</span>
                <span class="text-muted-foreground">{{ e.wx_state_text }}</span>
              </div>
              <div v-else-if="e.status === 'rejected' && e.reject_reason" class="flex gap-1.5 text-xs">
                <span class="shrink-0 text-destructive/80">驳回原因</span>
                <span class="min-w-0 flex-1 truncate text-destructive" :title="e.reject_reason">{{ e.reject_reason }}</span>
              </div>
              <span v-else class="text-xs text-muted-foreground/50">—</span>
            </td>
            <!-- 子商户号：获批子商户号，未开通弱化占位 -->
            <td>
              <div class="font-mono text-[13px] tabular-nums tracking-tight" :class="e.sub_mchid ? 'text-foreground' : 'text-muted-foreground/50'">{{ e.sub_mchid || '未开通' }}</div>
            </td>
            <!-- 操作：按状态直给主操作 + 查看（整列标题+内容一起右移，减少右侧留白） -->
            <td class="!pl-10">
              <div class="flex flex-wrap items-center gap-1">
                <Button
                  v-if="e.status === 'draft'"
                  size="sm"
                  @click="openFill(e.id)"
                >
                  <PencilLine class="size-3.5" />填写资料
                </Button>
                <Button
                  v-else-if="e.status === 'rejected'"
                  size="sm"
                  @click="openFill(e.id)"
                >
                  <PencilLine class="size-3.5" />修改重提
                </Button>
                <Button
                  v-else-if="e.status === 'submitted'"
                  variant="outline"
                  size="sm"
                  :disabled="syncing === e.id"
                  @click="syncState(e.id)"
                >
                  <RefreshCw class="size-3.5" :class="{ 'animate-spin': syncing === e.id }" />刷新状态
                </Button>
                <span v-else-if="e.status === 'approved'" class="inline-flex items-center gap-1 text-xs text-success">
                  <CheckCircle2 class="size-3.5" />已开通
                </span>
                <Button variant="ghost" size="sm" @click="openDetail(e.id)">查看</Button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </Panel>

    <!-- 建单抽屉：选服务商通道 -->
    <Drawer v-model="createOpen" title="发起通道进件" subtitle="选择要进件开通的服务商通道">
      <div class="space-y-4">
        <div class="row-field">
          <label class="lbl">服务商通道</label>
          <Select
            v-model="createForm.channel_id"
            :options="channelOptions"
            placeholder="请选择要进件的服务商通道"
            class="flex-1"
          />
        </div>
        <div class="row-field">
          <label class="lbl">主体名称</label>
          <input v-model="createForm.merchant_name" class="field-input flex-1" placeholder="商户/主体名称（可稍后在资料中完善）" />
        </div>
        <div class="row-field">
          <label class="lbl">联系手机</label>
          <input v-model="createForm.contact_phone" class="field-input flex-1" placeholder="用于进度联系" />
        </div>
      </div>
      <template #footer>
        <Button variant="ghost" @click="createOpen = false">取消</Button>
        <Button :disabled="creating" @click="doCreate"><Plus class="size-4" />创建并填写资料</Button>
      </template>
    </Drawer>

    <!-- 填料抽屉：复用共享 EnrollMaterialDrawer（注入商户线自己的 API）——
         保存资料后，点表格「查看」里的「提交微信」正式送审。 -->
    <EnrollMaterialDrawer
      v-model="materialOpen"
      :enroll-id="activeId"
      :merchant-name="activeName"
      :fetch-fn="myGetChannelEnrollMaterial"
      :submit-fn="myFillChannelEnrollMaterial"
      :upload-fn="myUploadChannelEnrollMedia"
      :upload-video-fn="myUploadChannelEnrollVideo"
      :ocr-license-fn="ocrLicense"
      :ocr-idcard-fn="ocrIdcard"
      @saved="onMaterialSaved"
    />

    <!-- 详情抽屉：审核中/已开通/已驳回查看 + 提交微信/刷新状态动作 -->
    <Drawer v-model="detailOpen" title="进件详情" :subtitle="detail?.channel_name" width="max-w-xl">
      <div v-if="detail" class="space-y-4">
        <!-- 顶部四步步骤条：一眼看到这单走到哪 -->
        <div class="flex items-center gap-1 bg-muted/40 px-4 py-4">
          <template v-for="(s, i) in STEPS" :key="s.key">
            <div class="flex flex-1 flex-col items-center gap-1.5 text-center">
              <div
                class="flex size-8 items-center justify-center rounded-full transition-colors"
                :class="{
                  'bg-success text-white': detailStepState(i) === 'done',
                  'bg-primary text-primary-foreground': detailStepState(i) === 'current',
                  'bg-destructive text-white': detailStepState(i) === 'error',
                  'bg-background text-muted-foreground/70': detailStepState(i) === 'todo',
                }"
              >
                <CheckCircle2 v-if="detailStepState(i) === 'done'" class="size-[18px]" />
                <X v-else-if="detailStepState(i) === 'error'" class="size-[18px]" />
                <component :is="s.icon" v-else class="size-[16px]" />
              </div>
              <div
                class="text-[11px] font-medium leading-tight"
                :class="detailStepState(i) === 'todo' ? 'text-muted-foreground' : 'text-foreground'"
              >
                {{ s.label }}
              </div>
            </div>
            <div
              v-if="i < STEPS.length - 1"
              class="mb-5 h-px flex-1 shrink-0"
              :class="i < detailStepIdx ? 'bg-success/60' : 'bg-border'"
            />
          </template>
        </div>

        <!-- 驳回原因：整体原因 + 微信逐字段详情（照此逐项修改后用相同单据重新提交） -->
        <div
          v-if="detail.status === 'rejected' && (detail.reject_reason || detail.audit_detail?.length)"
          class="border-l-2 border-destructive bg-destructive/[0.05] px-4 py-2.5 text-xs"
        >
          <div v-if="detail.reject_reason" class="flex gap-2 text-muted-foreground">
            <X class="size-4 shrink-0 text-destructive" />
            <span>微信驳回：{{ detail.reject_reason }}，请按下方逐项修改资料后重新提交。</span>
          </div>
          <dl v-if="detail.audit_detail?.length" class="mt-2 space-y-1.5" :class="{ 'border-t border-destructive/20 pt-2': detail.reject_reason }">
            <div v-for="(a, i) in detail.audit_detail" :key="i" class="flex gap-2">
              <dt class="w-28 shrink-0 font-medium text-foreground">{{ a.field_name || a.field || '—' }}</dt>
              <dd class="min-w-0 flex-1 text-muted-foreground">{{ a.reject_reason || '—' }}</dd>
            </div>
          </dl>
        </div>
        <!-- 已开通 -->
        <div v-if="detail.status === 'approved'" class="flex items-center gap-2 bg-success/[0.07] px-4 py-3 text-sm">
          <CheckCircle2 class="size-5 text-success" />
          <span class="text-success font-medium">进件已开通，子商户号 {{ detail.sub_mchid }}，该通道已为您启用。</span>
        </div>
        <!-- 业务受限就地快照（风控第二段：已开通子商户被微信管控时的能力/原因摘要，跳转「业务受限」页看全貌） -->
        <div v-if="detail.status === 'approved' && controlView && (controlView.state === 'controlled' || controlView.state === 'delayed')" class="border-l-2 border-destructive bg-destructive/[0.05] px-4 py-2.5 text-xs">
          <div class="flex items-center gap-2 text-muted-foreground">
            <ShieldAlert class="size-4 shrink-0 text-destructive" />
            <span class="text-foreground">{{ controlView.state_text }}</span>
            <span v-if="controlView.limited_function_texts?.length">{{ controlView.limited_function_texts.join('、') }}</span>
            <button class="ml-auto shrink-0 inline-flex items-center gap-1 text-primary hover:text-primary/80" @click="gotoChannelControls">
              查看详情<ExternalLink class="size-3" />
            </button>
          </div>
        </div>
        <!-- 待签约：二维码直接内嵌，超管扫码即签约，无需另开网页 -->
        <div v-if="detail.sign_url" class="flex gap-4 bg-warning/[0.08] px-4 py-4">
          <div class="shrink-0">
            <img v-if="signQrImageURL" :src="signQrImageURL" alt="签约二维码" class="size-32 rounded bg-white p-1.5" referrerpolicy="no-referrer" />
            <img v-else-if="signQrDataURL" :src="signQrDataURL" alt="签约二维码" class="size-32 rounded bg-white p-1.5" />
            <div v-else class="flex size-32 items-center justify-center rounded bg-white/60 text-xs text-muted-foreground">生成中…</div>
          </div>
          <div class="min-w-0 flex-1 text-sm">
            <div class="flex items-center gap-1.5 font-medium">
              <QrCode class="size-4 text-warning shrink-0" />需超级管理员扫码签约激活
            </div>
            <p class="mt-1.5 text-xs leading-relaxed text-muted-foreground">
              请贵司的超级管理员<span v-if="detailMaterial?.contact_name_masked" class="text-foreground">（{{ detailMaterial.contact_name_masked }}）</span>用<span class="text-foreground">微信</span>扫描左侧二维码，关注「微信支付商家助手」公众号后，按指引完成核对信息、账户验证与签约。
            </p>
            <div class="mt-2.5 flex items-center gap-3">
              <button class="inline-flex items-center gap-1 text-xs text-primary transition-colors hover:text-primary/80" @click="copySignURL">
                <Copy class="size-3.5" />复制链接
              </button>
              <a :href="detail.sign_url" target="_blank" class="text-xs text-muted-foreground underline transition-colors hover:text-foreground">在新窗口打开</a>
            </div>
          </div>
        </div>

        <!-- 状态摘要 -->
        <dl class="grid grid-cols-1 gap-x-8 gap-y-px bg-muted/40 px-4 py-1 text-sm sm:grid-cols-2">
          <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">主体名称</dt><dd class="font-medium">{{ detail.merchant_name || '—' }}</dd></div>
          <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">进件状态</dt><dd class="font-medium">{{ detail.status_text }}</dd></div>
          <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">微信状态</dt><dd class="font-medium">{{ detail.wx_state_text || '—' }}</dd></div>
          <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">申请单号</dt><dd class="font-mono text-xs tabular-nums">{{ detail.wx_applyment_id || '—' }}</dd></div>
          <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">商户简称</dt><dd class="font-medium">{{ detailMaterial?.merchant_shortname || '—' }}</dd></div>
          <div class="flex py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">客服电话</dt><dd class="font-medium">{{ detailMaterial?.service_phone || '—' }}</dd></div>
        </dl>
      </div>
      <template #footer>
        <Button
          v-if="detail && detail.status === 'submitted'"
          variant="outline"
          :disabled="syncing === detail.id"
          @click="syncState(detail.id)"
        >
          <RefreshCw class="size-4" :class="{ 'animate-spin': syncing === detail?.id }" />刷新微信状态
        </Button>
        <Button
          v-if="detail"
          variant="outline"
          class="text-destructive hover:bg-destructive/[0.06]"
          :disabled="deleting === detail.id || !(detail.status === 'draft' || detail.status === 'rejected')"
          :title="
            detail.status === 'submitted'
              ? '已提交微信审核，无法删除。可等审核结果后再操作。'
              : detail.status === 'approved'
                ? '通道已开通，不可删除。如需下线，请使用支付开关停用。'
                : ''
          "
          @click="deleteEnroll(detail.id)"
        >
          <Trash2 class="size-4" />删除
        </Button>
        <Button v-if="detail && (detail.status === 'draft' || detail.status === 'rejected')" @click="submitToWx(detail.id)">
          提交微信进件
        </Button>
      </template>
    </Drawer>
  </div>
</template>
