<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Eye, Plus } from 'lucide-vue-next'
import { Panel, Button, Badge, Drawer, Pagination, Select } from '@/components/ui'
import EnrollMaterialDrawer from '@/views/enroll/EnrollMaterialDrawer.vue'
import EnrollSettleDrawer from '@/views/enroll/EnrollSettleDrawer.vue'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import {
  fetchAgents,
  fetchEnrolls,
  getEnroll,
  createEnroll,
  submitEnroll,
  syncEnroll,
  refundEnroll,
  getEnrollMaterial,
  fillEnrollMaterial,
  uploadEnrollMedia,
  uploadEnrollVideo,
  getEnrollSettlement,
  modifyEnrollSettlement,
  getEnrollSettleApplication,
  type Agent,
  type Enroll,
  type EnrollAuditDetail,
} from '@/lib/api/console'
import { recognizeLicense, recognizeIDCard } from '@/lib/api/ocr'
import { ApiError } from '@/lib/api/client'

// OCR 识别（控制台走 /console 前缀，鉴权同控制台 token）。
const ocrLicense = (file: File) => recognizeLicense('/console', file)
const ocrIdcard = (file: File, side?: 'front' | 'back') => recognizeIDCard('/console', file, side)

const enrolls = ref<Enroll[]>([])
const total = ref(0)
const loading = ref(false)
const agents = ref<Agent[]>([])

const filters = reactive({ keyword: '', status: '', agentId: '', source: '' })
const page = ref(1)
const pageSize = 20
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// 本地状态机文案（对齐 docs-代理进件/01 第五节）
const statusMeta: Record<string, { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }> = {
  pending_pay: { label: '待支付', variant: 'warning' },
  paid: { label: '已支付待完善', variant: 'default' },
  submitted: { label: '审核中', variant: 'default' },
  finished: { label: '已完成', variant: 'success' },
  rejected: { label: '已驳回', variant: 'destructive' },
  closed: { label: '已关单', variant: 'muted' },
  refunded: { label: '已退款', variant: 'muted' },
}
const sourceText: Record<number, string> = { 1: '平台代填', 2: '代理代填', 3: '客户自助' }
const pathText: Record<number, string> = { 1: '预购名额', 2: '商户自付' }
const agentName = (id: number) => (id === 0 ? '平台' : agents.value.find((a) => a.id === id)?.name ?? `#${id}`)
const toast = useToast()
const confirm = useConfirm()
const router = useRouter()

// 收单同款 Select 选项
const statusOptions = computed(() => [
  { value: '', label: '全部' },
  ...Object.entries(statusMeta).map(([k, m]) => ({ value: k, label: m.label })),
])
const sourceOptions = [
  { value: '', label: '全部' },
  { value: '1', label: '平台代填' },
  { value: '2', label: '代理代填' },
  { value: '3', label: '客户自助' },
]
// 代理筛选（含「全部」）与建单归属（含「平台自己」）——值统一用字符串
const agentFilterOptions = computed(() => [
  { value: '', label: '全部' },
  ...agents.value.map((a) => ({ value: String(a.id), label: a.name })),
])
const agentOwnerOptions = computed(() => [
  { value: '', label: '平台自己（无代理）' },
  ...agents.value.map((a) => ({ value: String(a.id), label: a.name })),
])
const pathOptions = [
  { value: 2, label: '商户自付（分账）' },
  { value: 1, label: '预购名额（全额归代理）' },
]

async function load() {
  loading.value = true
  try {
    const { list, total: t } = await fetchEnrolls({
      keyword: filters.keyword || undefined,
      status: filters.status || undefined,
      agent_id: filters.agentId ? Number(filters.agentId) : undefined,
      source: filters.source ? Number(filters.source) : undefined,
      page: page.value,
      pageSize,
    })
    enrolls.value = list
    total.value = t
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载进件单失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const { list } = await fetchAgents({ pageSize: 500 })
    agents.value = list
  } catch {
    // ignore
  }
  await load()
})

function search() {
  page.value = 1
  load()
}
function resetFilters() {
  filters.keyword = ''
  filters.status = ''
  filters.agentId = ''
  filters.source = ''
  search()
}
function go(p: number) {
  page.value = p
  load()
}

// ===== 详情抽屉 =====
const drawer = ref(false)
const detail = ref<Enroll | null>(null)
const acting = ref(false)
async function openDetail(e: Enroll) {
  detail.value = e
  drawer.value = true
  try {
    detail.value = await getEnroll(e.id)
  } catch {
    // 用列表行数据兜底
  }
}

// ===== 建单抽屉（付费前置：建单即下开户费收款）=====
const createDrawer = ref(false)
const creating = ref(false)
const form = reactive({ merchantName: '', contactPhone: '', path: 2, agentId: '', plugin: 'mock' })
function openCreate() {
  Object.assign(form, { merchantName: '', contactPhone: '', path: 2, agentId: '', plugin: 'mock' })
  createDrawer.value = true
}
async function doCreate() {
  if (!form.merchantName.trim()) {
    toast.error('请填写商户名称')
    return
  }
  creating.value = true
  try {
    const r = await createEnroll({
      merchant_name: form.merchantName.trim(),
      contact_phone: form.contactPhone.trim() || undefined,
      path: form.path,
      agent_id: form.agentId ? Number(form.agentId) : undefined,
      plugin: form.plugin,
    })
    createDrawer.value = false
    await load()
    if (r.pay?.trade_no && (r.pay?.qrcode || r.pay?.pay_url)) {
      // 建单成功即引导去支付；即便这里关掉，进件单详情「去支付」按钮随时可重新打开（不会丢链接）。
      const go = await confirm(
        `进件单已建，待支付开户费 ¥${r.pay.money}。\n收款单号：${r.pay.trade_no}\n\n现在去支付？稍后也可在进件单详情点「去支付」重新打开。`,
        { title: '进件单已创建', confirmText: '去支付', cancelText: '稍后' },
      )
      if (go) openCashier(r.pay.trade_no)
    } else {
      toast.success('进件单已建（无需收费，已放行填料）')
    }
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '建单失败')
  } finally {
    creating.value = false
  }
}

// 待支付 → 可重新打开收款台付开户费（复用已建的收款单，不新建单，避免重复收费）
const canPay = computed(() => detail.value?.status === 'pending_pay' && !!detail.value?.pay_order_no)
function openCashier(tradeNo: string) {
  // 新窗口打开收款台：进件单列表/详情状态不丢，付完回来直接查状态即可。
  const href = router.resolve(`/pay/cashier/${tradeNo}`).href
  window.open(href, '_blank')
}
function goPay() {
  if (detail.value?.pay_order_no) openCashier(detail.value.pay_order_no)
}
// 已支付待完善 / 被驳回可重提 → 提交微信审核
const canSubmit = computed(() => detail.value?.status === 'paid' || detail.value?.status === 'rejected')
// 审核中 → 可主动查微信最新状态
const canSync = computed(() => detail.value?.status === 'submitted')
// 已开通硬锁定：sub_mchid 非空一律不可退（最高优先级，平台端也不例外）
const subMchLocked = computed(() => !!detail.value?.wx_sub_mchid)
// 可退：已付待完善 / 审核中 / 已驳回，且未开通硬锁定（平台兜底退任意代理名下单）
const canRefund = computed(
  () =>
    !subMchLocked.value &&
    ['paid', 'submitted', 'rejected'].includes(detail.value?.status ?? ''),
)
// 可填料：已支付待完善 / 被驳回（提交微信前的完整资料录入）
const canFill = computed(() => detail.value?.status === 'paid' || detail.value?.status === 'rejected')
// 已开通 → 可管理结算账户（微信接口 6/7/8）
const canSettle = computed(() => detail.value?.status === 'finished' && !!detail.value?.wx_sub_mchid)
// 驳回详情逐字段解析（audit_detail 是 JSON 数组串）
const auditDetails = computed<EnrollAuditDetail[]>(() => {
  const raw = detail.value?.audit_detail
  if (!raw) return []
  try {
    const arr = JSON.parse(raw)
    return Array.isArray(arr) ? arr : []
  } catch {
    return []
  }
})

// ===== 填料抽屉 =====
const materialDrawer = ref(false)
function openMaterial() {
  if (detail.value) materialDrawer.value = true
}
async function onMaterialSaved() {
  if (detail.value) detail.value = await getEnroll(detail.value.id)
}

// ===== 结算账户抽屉 =====
const settleDrawer = ref(false)
function openSettle() {
  if (detail.value) settleDrawer.value = true
}
async function onSettleChanged() {
  if (detail.value) detail.value = await getEnroll(detail.value.id)
}

async function doSubmit() {
  if (!detail.value) return
  acting.value = true
  try {
    detail.value = await submitEnroll(detail.value.id)
    toast.success('已提交微信')
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '提交微信失败')
  } finally {
    acting.value = false
  }
}
async function doSync() {
  if (!detail.value) return
  acting.value = true
  try {
    detail.value = await syncEnroll(detail.value.id)
    toast.success('已同步微信状态')
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '查询微信状态失败')
  } finally {
    acting.value = false
  }
}
async function doRefund() {
  if (!detail.value) return
  if (!(await confirm(`确认原路退还「${detail.value.merchant_name}」的开户费全额？退款后不可撤销。`, { title: '退款确认', danger: true }))) return
  acting.value = true
  try {
    const r = await refundEnroll(detail.value.id)
    toast.success(r.msg || '退款成功')
    detail.value = await getEnroll(detail.value.id)
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '退款失败')
  } finally {
    acting.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="进件申请" :subtitle="`共 ${total} 笔进件单`">
      <template #actions>
        <Button size="sm" @click="openCreate"><Plus />建进件单</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">搜索</label>
          <input
            v-model="filters.keyword"
            placeholder="商户名 / 进件单号"
            class="field-input w-48"
            @keyup.enter="search"
          />
        </div>
        <div class="filter-item">
          <label class="filter-label">状态</label>
          <Select v-model="filters.status" :options="statusOptions" class="w-32" />
        </div>
        <div class="filter-item">
          <label class="filter-label">代理</label>
          <Select v-model="filters.agentId" :options="agentFilterOptions" searchable class="w-40" />
        </div>
        <div class="filter-item">
          <label class="filter-label">来源</label>
          <Select v-model="filters.source" :options="sourceOptions" class="w-28" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="search">搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters">重置</Button>
        </div>
      </div>
    </Panel>

    <Panel title="进件单列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[10%]">进件单号</th>
              <th class="w-[20%]">商户名称</th>
              <th class="w-[10%]">联系电话</th>
              <th class="w-[10%]">归属代理</th>
              <th class="w-[10%]">来源</th>
              <th class="w-[10%]">支付方式</th>
              <th class="w-[10%]">状态</th>
              <th class="w-[10%]">特约商户号</th>
              <th class="col-center w-[10%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="e in enrolls" :key="e.id">
              <td class="truncate font-medium tabular-nums">{{ e.enroll_no }}</td>
              <td class="truncate">{{ e.merchant_name }}</td>
              <td class="truncate tabular-nums" :class="e.contact_phone ? '' : 'dim'">{{ e.contact_phone || '—' }}</td>
              <td class="truncate dim">{{ agentName(e.agent_id) }}</td>
              <td class="text-xs dim">{{ sourceText[e.source] ?? '—' }}</td>
              <td>
                <Badge :variant="e.path === 1 ? 'success' : 'default'">{{ pathText[e.path] ?? '—' }}</Badge>
              </td>
              <td>
                <Badge :variant="statusMeta[e.status]?.variant ?? 'muted'">
                  {{ statusMeta[e.status]?.label ?? e.status }}
                </Badge>
              </td>
              <td class="truncate tabular-nums" :class="e.wx_sub_mchid ? '' : 'dim'">{{ e.wx_sub_mchid || '未开通' }}</td>
              <td class="col-center">
                <Button variant="ghost" size="sm" @click="openDetail(e)"><Eye class="size-4" /></Button>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="9" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!enrolls.length">
              <td colspan="9" class="py-10 text-center dim">暂无进件单</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 详情抽屉 -->
    <Drawer v-model="drawer" title="进件单详情">
      <div v-if="detail" class="space-y-2 text-sm">
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">进件单号</span><span class="tabular-nums">{{ detail.enroll_no }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">商户名称</span><span>{{ detail.merchant_name }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">主体类型</span><span>{{ detail.subject_type || '—' }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">联系手机</span><span class="tabular-nums">{{ detail.contact_phone || '—' }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">归属代理</span><span>{{ agentName(detail.agent_id) }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">发起来源</span><span>{{ sourceText[detail.source] ?? '—' }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">资金路径</span><span>{{ pathText[detail.path] ?? '—' }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">状态</span>
          <Badge :variant="statusMeta[detail.status]?.variant ?? 'muted'">
            {{ statusMeta[detail.status]?.label ?? detail.status }}
          </Badge>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">收款单号</span><span class="tabular-nums">{{ detail.pay_order_no || '—' }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">微信申请单号</span><span class="tabular-nums">{{ detail.wx_applyment_id || '—' }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">特约商户号</span><span class="tabular-nums">{{ detail.wx_sub_mchid || '未开通' }}</span>
        </div>
        <div v-if="detail.sign_url" class="py-2">
          <div class="dim mb-1">超管签约链接</div>
          <div class="bg-muted/40 px-3 py-2 space-y-1.5">
            <p class="text-xs text-muted-foreground">超级管理员用微信扫码 / 打开，关注“微信支付商家助手”完成核对信息、账户验证、签约。</p>
            <a :href="detail.sign_url" target="_blank" class="block truncate text-xs text-primary underline">{{ detail.sign_url }}</a>
          </div>
        </div>
        <div v-if="detail.reject_reason" class="py-2">
          <div class="dim mb-1">驳回原因</div>
          <div class="bg-destructive/[0.06] px-3 py-2 text-destructive">{{ detail.reject_reason }}</div>
        </div>
        <div v-if="auditDetails.length" class="py-2">
          <div class="dim mb-1">驳回详情（逐字段）</div>
          <div class="space-y-1.5">
            <div v-for="(a, i) in auditDetails" :key="i" class="bg-destructive/[0.06] px-3 py-2">
              <div class="text-xs font-medium text-destructive">{{ a.field_name || a.field }}</div>
              <div class="text-xs text-destructive/90">{{ a.reject_reason }}</div>
            </div>
          </div>
        </div>
      </div>
      <p class="mt-4 text-xs text-muted-foreground">
        提交微信调用服务商 APIv3 进件接口（applyment4sub）；审核中可点“查状态”拉取微信最新进度。
        退款原路退还开户费全额（不扣手续费）；<span v-if="subMchLocked" class="text-destructive">商户已成功开通（sub_mchid 已下发），硬锁定不可退。</span>
      </p>
      <template #footer>
        <Button variant="outline" @click="drawer = false">关闭</Button>
        <Button v-if="canRefund" :disabled="acting" variant="outline" @click="doRefund">
          {{ acting ? '处理中…' : '退款' }}
        </Button>
        <Button v-if="canSync" :disabled="acting" variant="outline" @click="doSync">
          {{ acting ? '查询中…' : '查状态' }}
        </Button>
        <Button v-if="canFill" :disabled="acting" variant="outline" @click="openMaterial">填料</Button>
        <Button v-if="canSettle" variant="outline" @click="openSettle">结算账户</Button>
        <Button v-if="canPay" @click="goPay">去支付</Button>
        <Button v-if="canSubmit" :disabled="acting" @click="doSubmit">
          {{ acting ? '提交中…' : detail?.status === 'rejected' ? '重新提交' : '提交微信' }}
        </Button>
      </template>
    </Drawer>

    <!-- 填全套资料抽屉 -->
    <EnrollMaterialDrawer
      v-model="materialDrawer"
      :enroll-id="detail?.id ?? null"
      :merchant-name="detail?.merchant_name"
      :fetch-fn="getEnrollMaterial"
      :submit-fn="fillEnrollMaterial"
      :upload-fn="uploadEnrollMedia"
      :upload-video-fn="uploadEnrollVideo"
      :ocr-license-fn="ocrLicense"
      :ocr-idcard-fn="ocrIdcard"
      @saved="onMaterialSaved"
    />

    <!-- 结算账户管理抽屉 -->
    <EnrollSettleDrawer
      v-model="settleDrawer"
      :enroll-id="detail?.id ?? null"
      :merchant-name="detail?.merchant_name"
      :sub-mch-id="detail?.wx_sub_mchid"
      :get-fn="getEnrollSettlement"
      :modify-fn="modifyEnrollSettlement"
      :get-application-fn="getEnrollSettleApplication"
      @changed="onSettleChanged"
    />

    <!-- 建单抽屉（付费前置）-->
    <Drawer v-model="createDrawer" title="建进件单" subtitle="第一步只填基础信息，建单即收开户费，付款成功后放行填全套资料">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">商户名称<span class="text-destructive">*</span></label>
          <input v-model="form.merchantName" placeholder="特约商户主体名称" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">联系手机</label>
          <input v-model="form.contactPhone" placeholder="用于进度查询匹配" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">归属代理</label>
          <Select v-model="form.agentId" :options="agentOwnerOptions" searchable class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">资金路径</label>
          <Select v-model="form.path" :options="pathOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">收款方式</label>
          <input v-model="form.plugin" placeholder="收开户费的渠道 plugin，如 mock / alipay / wxpay" class="field-input flex-1" />
        </div>
        <p class="text-[11px] text-muted-foreground">
          开户零售价读“进件设置”，程序不硬编码。路径一若后台配置“客户免付开户费”，建单直接放行不收款。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" @click="createDrawer = false">取消</Button>
        <Button :disabled="creating" @click="doCreate">{{ creating ? '创建中…' : '创建并收款' }}</Button>
      </template>
    </Drawer>
  </div>
</template>
