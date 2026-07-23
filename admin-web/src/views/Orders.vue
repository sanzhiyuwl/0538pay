<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import {
  fetchOrders,
  setOrderStatus,
  freezeOrder,
  unfreezeOrder,
  fetchRefundInfo,
  refundOrder,
  fillOrder,
  renotifyOrder,
  deleteOrder,
  batchOrders,
  exportOrders,
  fetchOrderStats,
  type OrderStats,
  type OrderListParams,
} from '@/lib/api/orders'
import { ApiError } from '@/lib/api/client'
import { addBlacklist } from '@/lib/api/risk'
import {
  Search,
  RotateCcw,
  Download,
  BarChart3,
  MoreHorizontal,
  CheckCircle2,
  Undo2,
  Snowflake,
  Sun,
  Bell,
  FilePlus2,
  FileText,
  Trash2,
  Ban,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, DateRange, Pagination, Drawer, Modal } from '@/components/ui'
import {
  orderStatus,
  payTypes,
  searchColumns,
  type Order,
} from '@/lib/mock/orders'
import { formatMoney } from '@/lib/utils'
import { useToast } from '@/composables/useToast'
import { shouldDropUp } from '@/composables/useRowMenu'

const toast = useToast()

// 下拉选项（适配 Select 组件的 {value,label} 结构）
const columnOptions = searchColumns.map((c) => ({ value: c.value, label: c.label }))
const typeOptions = [
  { value: 0, label: '请选择' },
  ...payTypes.map((t) => ({ value: t.id, label: t.showname })),
]
const statusOptions = [
  { value: -1, label: '请选择' },
  ...Object.entries(orderStatus).map(([k, s]) => ({ value: Number(k), label: s.text })),
]

// ===== 筛选 =====
const filters = ref({
  column: 'trade_no',
  value: '',
  uid: '',
  type: 0,
  channel: '',
  starttime: '',
  endtime: '',
  dstatus: -1,
})

// ===== 分页（服务端）=====
const page = ref(1)
const pageSize = 15
const total = ref(0)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))

// 把当前筛选映射为后端 OrderListParams（筛选与分页整体下推后端，对齐 epay 服务端筛选，
// 修正旧版仅对前 100 条前端筛选、列表/统计在 >100 单或 type/channel/时间筛选时不准的问题）。
function buildParams(): OrderListParams {
  const f = filters.value
  const p: OrderListParams = { page: page.value, pageSize }
  if (f.value.trim()) {
    p.column = f.column
    p.keyword = f.value.trim()
  }
  if (f.uid.trim()) p.uid = Number(f.uid.trim()) || 0
  if (f.type) p.type = f.type
  if (f.channel.trim()) p.channel = Number(f.channel.trim()) || 0
  if (f.dstatus > -1) p.status = f.dstatus
  if (f.starttime) p.starttime = f.starttime
  if (f.endtime) p.endtime = f.endtime
  return p
}

// ===== 数据源：从后端 API 加载（服务端分页 + 筛选）=====
const pageRows = ref<Order[]>([])
const loading = ref(false)
const loadError = ref('')

async function loadOrders() {
  loading.value = true
  loadError.value = ''
  try {
    const res = await fetchOrders(buildParams())
    pageRows.value = res.list
    total.value = res.total
    if (showStats.value) await loadStats()
  } catch (e) {
    loadError.value = e instanceof ApiError ? e.message : '加载订单失败'
    pageRows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function resetFilters() {
  filters.value = {
    column: 'trade_no',
    value: '',
    uid: '',
    type: 0,
    channel: '',
    starttime: '',
    endtime: '',
    dstatus: -1,
  }
  page.value = 1
  loadOrders()
}

// 搜索/翻页/重置都重新拉后端
function search() {
  page.value = 1
  loadOrders()
}
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
  loadOrders()
}

// ===== 统计（后端全量聚合，非当前页）=====
const showStats = ref(false)
const stats = ref<OrderStats>({
  totalMoney: 0, successMoney: 0, unpaidMoney: 0, refundMoney: 0, platformProfit: 0,
  totalCount: 0, successCount: 0, unpaidCount: 0, refundCount: 0, successRate: 0,
})
async function loadStats() {
  try {
    // 统计与列表同筛选，但不带分页（后端按全部匹配订单聚合，对齐 epay 统计按钮）
    const { page: _p, pageSize: _ps, ...f } = buildParams()
    stats.value = await fetchOrderStats(f)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '统计加载失败')
  }
}
async function toggleStats() {
  showStats.value = !showStats.value
  if (showStats.value) await loadStats()
}

// ===== 行操作菜单 =====
const openMenu = ref<string | null>(null)
const dropUp = ref(false)
function toggleMenu(id: string, ev?: MouseEvent) {
  if (openMenu.value === id) { openMenu.value = null; return }
  openMenu.value = id
  dropUp.value = shouldDropUp(ev)
}
// 点击页面任意处关闭菜单（按钮自身的 click 用 .stop 阻止冒泡）
function closeMenu() {
  openMenu.value = null
}
const route = useRoute()
onMounted(() => {
  window.addEventListener('click', closeMenu)
  // 从通道页「订单」按钮跳转而来：预置通道 ID 筛选（对齐 epay pay_channel「订单」→ order.php?channel=id）
  const qc = route.query.channel
  if (qc != null && String(qc).trim()) filters.value.channel = String(qc).trim()
  // 从商户页「查看订单」快捷跳转而来：预置商户号筛选
  const qu = route.query.uid
  if (qu != null && String(qu).trim()) filters.value.uid = String(qu).trim()
  loadOrders()
})
onUnmounted(() => window.removeEventListener('click', closeMenu))

function actionsFor(o: Order): string[] {
  if (o.status === 1) return ['订单详情', '-', '改未完成', 'API退款', '手动退款', '冻结订单', '-', '重新通知', '删除订单']
  if (o.status === 2) return ['订单详情', '-', '改未完成', 'API退款', '改已完成', '-', '重新通知', '删除订单']
  if (o.status === 3) return ['订单详情', '-', '解冻订单', 'API退款', '-', '重新通知', '删除订单']
  const base = ['订单详情', '-', '改已完成', '-', '手动补单', '删除订单']
  if (o.status === 4) return ['订单详情', '-', '授权资金支付', '授权资金解冻', '-', '改已完成', '-', '手动补单', '删除订单']
  return base
}

// 操作项 → 图标
const actionIcons: Record<string, any> = {
  订单详情: FileText,
  改未完成: RotateCcw,
  改已完成: CheckCircle2,
  'API退款': Undo2,
  手动退款: Undo2,
  冻结订单: Snowflake,
  解冻订单: Sun,
  重新通知: Bell,
  手动补单: FilePlus2,
  删除订单: Trash2,
  授权资金支付: CheckCircle2,
  授权资金解冻: Sun,
}

// ===== 一键拉黑（对齐 epay order.php addBlackList：stype 1=IP / 0=支付账号，days=0 永久）=====
const blackDialog = ref(false)
const blackForm = reactive({ type: 1, content: '', days: 0, remark: '' })
const blackSubmitting = ref(false)
const blackTitle = computed(() => (blackForm.type === 1 ? 'IP 地址' : '支付账号'))
function openBlacklist(type: number, content: string) {
  if (!content) {
    toast.error(type === 1 ? '该订单无支付 IP' : '该订单无支付账号')
    return
  }
  openMenu.value = null
  blackForm.type = type
  blackForm.content = content
  blackForm.days = 0
  blackForm.remark = ''
  blackDialog.value = true
}
async function submitBlacklist() {
  if (blackSubmitting.value) return
  if (!blackForm.content.trim()) {
    toast.error('拉黑内容不能为空')
    return
  }
  blackSubmitting.value = true
  try {
    await addBlacklist({
      type: blackForm.type,
      content: blackForm.content.trim(),
      days: Number(blackForm.days) || 0,
      remark: blackForm.remark.trim(),
    })
    toast.success('已加入黑名单')
    blackDialog.value = false
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加入黑名单失败')
  } finally {
    blackSubmitting.value = false
  }
}

// ===== 行操作调度 =====
const busy = ref(false)

// 需二次确认的操作 → 确认弹窗
const confirmState = ref<{ label: string; order: Order } | null>(null)
function askConfirm(label: string, o: Order) {
  confirmState.value = { label, order: o }
  openMenu.value = null
}
const confirmText = computed(() => {
  const c = confirmState.value
  if (!c) return ''
  const map: Record<string, string> = {
    改未完成: `确定把订单 ${c.order.trade_no} 改为「未完成」吗？仅改状态，不涉及资金。`,
    改已完成: `确定把订单 ${c.order.trade_no} 改为「已完成」吗？仅改状态，不入账（如需入账请用"手动补单"）。`,
    冻结订单: `确定冻结订单 ${c.order.trade_no} 吗？将从商户余额扣除该订单分成金额 ¥${c.order.getmoney}。`,
    解冻订单: `确定解冻订单 ${c.order.trade_no} 吗？将把分成金额 ¥${c.order.getmoney} 加回商户余额。`,
    手动补单: `确定为未支付订单 ${c.order.trade_no} 补单吗？将置为已支付并给商户入账 + 触发异步通知。`,
    重新通知: `确定向商户重新发送订单 ${c.order.trade_no} 的异步通知吗？`,
    删除订单: `确定删除订单 ${c.order.trade_no} 吗？物理删除不可恢复，且不退款、不改余额。`,
  }
  return map[c.label] ?? ''
})
async function runConfirm() {
  const c = confirmState.value
  if (!c) return
  busy.value = true
  try {
    const t = c.order.trade_no
    switch (c.label) {
      case '改未完成': await setOrderStatus(t, 0); break
      case '改已完成': await setOrderStatus(t, 1); break
      case '冻结订单': await freezeOrder(t); break
      case '解冻订单': await unfreezeOrder(t); break
      case '手动补单': await fillOrder(t); break
      case '重新通知': await renotifyOrder(t); break
      case '删除订单': await deleteOrder(t); break
    }
    toast.success('操作成功')
    confirmState.value = null
    await loadOrders()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  } finally {
    busy.value = false
  }
}

// 退款弹窗（手动退款 / API 退款共用，api 标识区分）
const refundState = ref<{ order: Order; api: boolean } | null>(null)
const refundForm = ref({ money: '', password: '' })
const refundInfo = ref<{ realmoney: number; refunded: number; refundable: number; can_api: boolean } | null>(null)
async function openRefund(o: Order, api: boolean) {
  openMenu.value = null
  refundState.value = { order: o, api }
  refundForm.value = { money: '', password: '' }
  refundInfo.value = null
  try {
    const info = await fetchRefundInfo(o.trade_no)
    refundInfo.value = info
    refundForm.value.money = info.refundable.toFixed(2)
    if (api && !info.can_api) {
      toast.error('该订单通道不支持原路退款')
    }
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '查询可退金额失败')
    refundState.value = null
  }
}
async function submitRefund() {
  const r = refundState.value
  if (!r) return
  const money = refundForm.value.money.trim()
  if (!money || Number(money) <= 0) return toast.error('请输入有效退款金额')
  if (r.api && !refundForm.value.password) return toast.error('原路退款需输入支付密码')
  busy.value = true
  try {
    await refundOrder({ trade_no: r.order.trade_no, money, api: r.api, password: refundForm.value.password })
    toast.success(r.api ? '退款已提交（渠道原路退款待凭证，余额已处理）' : '手动退款成功')
    refundState.value = null
    await loadOrders()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '退款失败')
  } finally {
    busy.value = false
  }
}

// 预授权类操作（status=4）依赖真实渠道凭证，如实提示
function notYet() {
  openMenu.value = null
  toast.info('预授权资金操作依赖真实渠道凭证，待接入')
}

// ===== 订单详情抽屉（对齐 epay order.php 行展开：看列表放不下的完整字段）=====
// 纯展示已加载行数据，无需额外接口。合单子订单列表依赖合单收单能力(我方未实现，不产生 combine=1 单)，暂不展示。
const detailOrder = ref<Order | null>(null)
function openDetail(o: Order) {
  openMenu.value = null
  detailOrder.value = o
}

// 菜单项点击调度
function onAction(label: string, o: Order) {
  switch (label) {
    case '订单详情': openDetail(o); break
    case 'API退款': openRefund(o, true); break
    case '手动退款': openRefund(o, false); break
    case '授权资金支付':
    case '授权资金解冻': notYet(); break
    default: askConfirm(label, o)
  }
}

// ===== 批量操作 =====
const selected = ref<Set<string>>(new Set())
function toggleSelect(tradeNo: string) {
  if (selected.value.has(tradeNo)) selected.value.delete(tradeNo)
  else selected.value.add(tradeNo)
  selected.value = new Set(selected.value)
}
const allChecked = computed(() => pageRows.value.length > 0 && pageRows.value.every((o) => selected.value.has(o.trade_no)))
function toggleSelectAll() {
  const next = new Set(selected.value)
  if (allChecked.value) pageRows.value.forEach((o) => next.delete(o.trade_no))
  else pageRows.value.forEach((o) => next.add(o.trade_no))
  selected.value = next
}
const batchAction = ref<number>(1)
const batchOptions = [
  { value: 1, label: '批量改已完成' },
  { value: 0, label: '批量改未完成' },
  { value: 2, label: '批量冻结' },
  { value: 3, label: '批量解冻' },
  { value: 5, label: '批量API退款' },
  { value: 4, label: '批量删除' },
]
const batchConfirm = ref(false)
function askBatch() {
  if (selected.value.size === 0) return toast.info('请先勾选订单')
  // API退款走独立流程（逐条 + 支付密码，对齐 epay batch_apirefund）
  if (batchAction.value === 5) return openBatchRefund()
  batchConfirm.value = true
}

// ===== 批量 API 退款（对齐 epay order.php batch_apirefund：逐条调 apirefund，带支付密码，
// 可退状态=已完成(1)/解冻(3)，退款额取每单 realmoney）=====
const batchRefundDialog = ref(false)
const batchRefundPwd = ref('')
const batchRefundRunning = ref(false)
// 选中订单里可退的（status 1 或 3，且有 realmoney）
const refundableSelected = computed(() =>
  pageRows.value.filter(
    (o) => selected.value.has(o.trade_no) && (o.status === 1 || o.status === 3) && !!o.realmoney,
  ),
)
function openBatchRefund() {
  if (refundableSelected.value.length === 0) {
    return toast.info('选中订单中没有可 API 退款的订单（仅已完成/解冻订单可退）')
  }
  batchRefundPwd.value = ''
  batchRefundDialog.value = true
}
async function runBatchRefund() {
  if (batchRefundRunning.value) return
  if (!batchRefundPwd.value) return toast.error('请输入支付密码')
  batchRefundRunning.value = true
  let ok = 0
  let fail = 0
  try {
    // 逐条串行退款（对齐 epay batch_apirefund_order 递归逐条），退款额取该单 realmoney
    for (const o of refundableSelected.value) {
      try {
        await refundOrder({
          trade_no: o.trade_no,
          money: String(o.realmoney),
          api: true,
          password: batchRefundPwd.value,
        })
        ok++
      } catch {
        fail++
      }
    }
    if (ok > 0) toast.success(`成功退款 ${ok} 条${fail > 0 ? `，失败 ${fail} 条` : ''}`)
    else toast.error(`退款失败 ${fail} 条`)
    batchRefundDialog.value = false
    selected.value = new Set()
    await loadOrders()
  } finally {
    batchRefundRunning.value = false
  }
}
async function runBatch() {
  busy.value = true
  try {
    const res = await batchOrders(batchAction.value, [...selected.value])
    toast.success(`已处理 ${res.affected} 条订单`)
    batchConfirm.value = false
    selected.value = new Set()
    await loadOrders()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '批量操作失败')
  } finally {
    busy.value = false
  }
}

// ===== 高级导出（对齐 epay export.php + download.php?act=order）=====
const exportOpen = ref(false)
const exportForm = ref({
  starttime: '',
  endtime: '',
  uid: '',
  type: 0,
  channel: '',
  dstatus: -1,
})
function openExport() {
  // 带入当前列表筛选条件作为默认导出条件，减少重复输入
  exportForm.value = {
    starttime: filters.value.starttime,
    endtime: filters.value.endtime,
    uid: filters.value.uid,
    type: filters.value.type,
    channel: filters.value.channel,
    dstatus: filters.value.dstatus,
  }
  exportOpen.value = true
}
// 服务端全量导出：按抽屉筛选条件从后端流式下载 CSV（不受列表分页限制，筛选整套下推后端）。
const serverExporting = ref(false)
async function serverExport() {
  if (serverExporting.value) return
  const f = exportForm.value
  serverExporting.value = true
  try {
    const p: OrderListParams = {}
    // 导出抽屉无搜索字段/关键词，沿用列表当前搜索条件
    if (filters.value.value.trim()) {
      p.column = filters.value.column
      p.keyword = filters.value.value.trim()
    }
    if (f.uid.trim()) p.uid = Number(f.uid.trim()) || 0
    if (f.type) p.type = f.type
    if (f.channel.trim()) p.channel = Number(f.channel.trim()) || 0
    if (f.dstatus > -1) p.status = f.dstatus
    if (f.starttime) p.starttime = f.starttime
    if (f.endtime) p.endtime = f.endtime
    await exportOrders(p)
    toast.success('已导出全部匹配订单（服务端）')
    exportOpen.value = false
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '导出失败')
  } finally {
    serverExporting.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选（标题即页面标题，参考图“标签+控件”横排样式） -->
    <Panel title="订单管理" :subtitle="`共 ${total} 笔订单`">
      <template #actions>
        <Button variant="outline" size="sm" @click="toggleStats">
          <BarChart3 />统计
        </Button>
        <Button variant="outline" size="sm" @click="openExport"><Download />导出订单</Button>
      </template>
      <div class="space-y-3">
        <!-- 第一行 -->
        <div class="filter-bar">
          <div class="filter-item">
            <label class="filter-label">订单信息</label>
            <Select v-model="filters.column" :options="columnOptions" class="w-32" />
            <input v-model="filters.value" placeholder="搜索内容" class="field-input w-48" />
          </div>
          <div class="filter-item">
            <label class="text-sm text-muted-foreground">商户号</label>
            <input v-model="filters.uid" placeholder="请输入商户号" class="field-input w-40" />
          </div>
          <div class="filter-item">
            <label class="text-sm text-muted-foreground">支付方式</label>
            <Select v-model="filters.type" :options="typeOptions" class="w-28" />
          </div>
          <div class="filter-item">
            <label class="text-sm text-muted-foreground">订单状态</label>
            <Select v-model="filters.dstatus" :options="statusOptions" class="w-28" />
          </div>
          <div class="filter-item">
            <label class="text-sm text-muted-foreground">通道 ID</label>
            <input v-model="filters.channel" placeholder="通道ID" class="field-input w-40" />
          </div>
        </div>
        <!-- 第二行：下单时间与订单信息左对齐 -->
        <div class="filter-bar">
          <div class="filter-item">
            <label class="filter-label">下单时间</label>
            <DateRange v-model:start="filters.starttime" v-model:end="filters.endtime" class="w-[328px]" />
          </div>
          <div class="ml-auto flex items-center gap-2">
            <Button size="sm" @click="search"><Search />搜索</Button>
            <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
          </div>
        </div>
      </div>
    </Panel>

    <!-- 统计概况（可展开） -->
    <Panel v-if="showStats" title="订单统计概况" subtitle="按当前筛选条件全量聚合（服务端，非当前页）">
      <div class="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3 lg:grid-cols-5">
        <div>
          <div class="text-[13px] text-muted-foreground">订单总金额</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.totalMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已支付金额</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-success"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.successMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">未支付金额</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.unpaidMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已退款金额</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-destructive"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.refundMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">总收入利润</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-primary"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.platformProfit) }}</div>
        </div>
      </div>
      <div class="mt-5 flex flex-wrap gap-x-8 gap-y-2 border-t border-border/70 pt-4 text-sm">
        <span class="text-muted-foreground">订单总数 <b class="text-foreground">{{ stats.totalCount }}</b></span>
        <span class="text-muted-foreground">已支付 <b class="text-foreground">{{ stats.successCount }}</b></span>
        <span class="text-muted-foreground">未支付 <b class="text-foreground">{{ stats.unpaidCount }}</b></span>
        <span class="text-muted-foreground">已退款 <b class="text-foreground">{{ stats.refundCount }}</b></span>
        <span class="text-muted-foreground">成功率 <b class="text-primary">{{ stats.successRate }}%</b></span>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel :title="`订单列表`" :subtitle="`${total} 条`">
      <!-- 批量操作条：勾选后出现 -->
      <div v-if="selected.size > 0" class="mb-3 flex items-center gap-2 rounded bg-muted/40 px-3 py-2">
        <span class="text-sm text-muted-foreground">已选 <b class="text-foreground">{{ selected.size }}</b> 条</span>
        <Select v-model="batchAction" :options="batchOptions" class="w-40" />
        <Button size="sm" @click="askBatch">执行</Button>
        <Button variant="ghost" size="sm" @click="selected = new Set()">清空选择</Button>
      </div>
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[4%] pl-1">
                <input type="checkbox" :checked="allChecked" class="align-middle" @change="toggleSelectAll" />
              </th>
              <th class="w-[13%]">订单号 / 商户单号</th>
              <th class="w-[10%]">商户 / 域名</th>
              <th class="w-[12%]">商品 / 金额</th>
              <th class="w-[10%]">实付 / 分成</th>
              <th class="w-[11%]">支付方式</th>
              <th class="w-[13%]">支付IP / 账号</th>
              <th class="w-[12%]">创建 / 完成时间</th>
              <th class="col-center w-[8%]">状态</th>
              <th class="col-center w-[7%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in pageRows" :key="o.trade_no">
              <td class="pl-1">
                <input
                  type="checkbox"
                  :checked="selected.has(o.trade_no)"
                  class="align-middle"
                  @change="toggleSelect(o.trade_no)"
                />
              </td>
              <td>
                <div class="truncate font-medium text-primary">{{ o.trade_no }}</div>
                <div class="truncate text-xs dim">{{ o.out_trade_no }}</div>
              </td>
              <td>
                <div>{{ o.uid }}</div>
                <div class="truncate text-xs dim">{{ o.domain }}</div>
              </td>
              <td>
                <div class="truncate">{{ o.name }}</div>
                <div class="text-xs"><span class="dim">¥</span><b>{{ o.money }}</b></div>
              </td>
              <td>
                <template v-if="o.realmoney">
                  <div><span class="dim text-xs">¥</span>{{ o.realmoney }}</div>
                  <div class="text-xs dim">分成 ¥{{ o.getmoney }}</div>
                </template>
                <span v-else class="dim">—</span>
              </td>
              <td>
                <div class="flex items-center gap-1.5">
                  <img :src="`/assets/icon/${o.typename}.ico`" class="size-4" onerror="this.style.display='none'" />
                  <span>{{ o.typeshowname }}</span>
                  <span class="dim">({{ o.channel }})</span>
                </div>
                <div class="text-xs dim">{{ o.plugin }}</div>
              </td>
              <td>
                <div v-if="o.ip" class="group flex items-center gap-1">
                  <span class="truncate font-mono text-xs">{{ o.ip }}</span>
                  <button
                    class="shrink-0 text-muted-foreground opacity-0 transition-opacity hover:text-destructive group-hover:opacity-100"
                    title="拉黑此 IP"
                    @click.stop="openBlacklist(1, o.ip)"
                  >
                    <Ban class="size-3.5" />
                  </button>
                </div>
                <div v-if="o.buyer" class="group flex items-center gap-1">
                  <span class="truncate text-xs dim">{{ o.buyer }}</span>
                  <button
                    class="shrink-0 text-muted-foreground opacity-0 transition-opacity hover:text-destructive group-hover:opacity-100"
                    title="拉黑此账号"
                    @click.stop="openBlacklist(0, o.buyer)"
                  >
                    <Ban class="size-3.5" />
                  </button>
                </div>
                <span v-if="!o.ip && !o.buyer" class="dim">—</span>
              </td>
              <td>
                <div class="text-xs">{{ o.addtime }}</div>
                <div class="text-xs dim">{{ o.endtime ?? '—' }}</div>
              </td>
              <td class="col-center">
                <Badge :variant="orderStatus[o.status].variant">{{ orderStatus[o.status].text }}</Badge>
                <div v-if="o.status === 2 && +o.refundmoney > 0 && +o.refundmoney < +(o.realmoney ?? 0)" class="mt-1 text-xs text-destructive">
                  部分退款 ¥{{ o.refundmoney }}
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(o.trade_no, $event)">
                    操作 <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === o.trade_no"
                    class="menu-panel absolute right-0 z-20 w-36"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <template v-for="(a, ai) in actionsFor(o)" :key="ai">
                      <div v-if="a === '-'" class="menu-sep" />
                      <button
                        v-else
                        class="menu-item"
                        :class="a === '删除订单' && 'menu-item-danger'"
                        @click="onAction(a, o)"
                      >
                        <component :is="actionIcons[a]" class="size-4 shrink-0 opacity-70" />
                        <span class="flex-1">{{ a }}</span>
                      </button>
                    </template>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="10" class="py-10 text-center dim">没有符合条件的订单</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 分页 -->
      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 高级导出抽屉（对齐 epay export.php：按时间/商户/方式/通道/状态组合导出 CSV）-->
    <Drawer v-model="exportOpen" title="导出订单" subtitle="按条件批量导出订单为 CSV 文件" width="max-w-md">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">时间范围<span class="text-destructive">*</span></label>
          <DateRange v-model:start="exportForm.starttime" v-model:end="exportForm.endtime" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">商户号</label>
          <input v-model="exportForm.uid" placeholder="留空为全部商户" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">支付方式</label>
          <Select v-model="exportForm.type" :options="typeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">通道 ID</label>
          <input v-model="exportForm.channel" placeholder="留空为全部通道" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">订单状态</label>
          <Select v-model="exportForm.dstatus" :options="statusOptions" class="flex-1" />
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          按上述条件从服务端全量导出匹配订单为 CSV（不受列表分页限制）。搜索字段/关键词沿用列表筛选。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="exportOpen = false">取消</Button>
        <Button size="sm" :disabled="serverExporting" @click="serverExport">
          <Download />服务端全量导出
        </Button>
      </template>
    </Drawer>

    <!-- 订单详情抽屉（对齐 epay order.php 行展开：看列表放不下的完整字段）-->
    <Drawer
      :model-value="!!detailOrder"
      title="订单详情"
      :subtitle="detailOrder?.trade_no"
      width="max-w-lg"
      @update:model-value="(v) => { if (!v) detailOrder = null }"
    >
      <div v-if="detailOrder" class="space-y-4 text-sm">
        <div class="rounded bg-muted/40 px-3.5 py-3">
          <div class="flex items-center justify-between">
            <span class="text-muted-foreground">订单状态</span>
            <Badge :variant="orderStatus[detailOrder.status].variant">{{ orderStatus[detailOrder.status].text }}</Badge>
          </div>
        </div>

        <div>
          <div class="mb-2 text-xs font-medium text-muted-foreground">订单标识</div>
          <dl class="space-y-2">
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">系统订单号</dt><dd class="text-right font-mono">{{ detailOrder.trade_no }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">商户订单号</dt><dd class="text-right font-mono">{{ detailOrder.out_trade_no || '—' }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">接口订单号</dt><dd class="text-right font-mono">{{ detailOrder.api_trade_no || '—' }}</dd></div>
            <div v-if="detailOrder.bill_trade_no" class="flex justify-between gap-4"><dt class="dim shrink-0">用户交易单号</dt><dd class="text-right font-mono break-all">{{ detailOrder.bill_trade_no }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">合单</dt><dd class="text-right">{{ detailOrder.combine ? '是（子订单列表依赖合单收单，暂未实现）' : '否' }}</dd></div>
          </dl>
        </div>

        <div>
          <div class="mb-2 text-xs font-medium text-muted-foreground">商户 / 商品</div>
          <dl class="space-y-2">
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">商户 UID</dt><dd class="text-right">{{ detailOrder.uid }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">来源域名</dt><dd class="text-right break-all">{{ detailOrder.domain || '—' }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">商品名称</dt><dd class="text-right break-all">{{ detailOrder.name }}</dd></div>
            <div class="flex justify-between gap-4">
              <dt class="dim shrink-0">买家标识</dt>
              <dd class="flex items-center justify-end gap-1.5 text-right break-all">
                <span>{{ detailOrder.buyer || '—' }}</span>
                <button v-if="detailOrder.buyer" class="shrink-0 text-muted-foreground hover:text-destructive" title="拉黑此账号" @click="openBlacklist(0, detailOrder.buyer)"><Ban class="size-3.5" /></button>
              </dd>
            </div>
            <div v-if="detailOrder.mobile" class="flex justify-between gap-4">
              <dt class="dim shrink-0">手机号码</dt>
              <dd class="flex items-center justify-end gap-1.5 text-right">
                <span>{{ detailOrder.mobile }}</span>
                <button class="shrink-0 text-muted-foreground hover:text-destructive" title="拉黑此手机号" @click="openBlacklist(0, detailOrder.mobile)"><Ban class="size-3.5" /></button>
              </dd>
            </div>
          </dl>
        </div>

        <div>
          <div class="mb-2 text-xs font-medium text-muted-foreground">金额</div>
          <dl class="space-y-2 tabular-nums">
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">订单金额</dt><dd class="text-right">¥{{ detailOrder.money }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">实付金额</dt><dd class="text-right">{{ detailOrder.realmoney ? '¥' + detailOrder.realmoney : '—' }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">商户分成</dt><dd class="text-right">¥{{ detailOrder.getmoney }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">已退金额</dt><dd class="text-right" :class="+detailOrder.refundmoney > 0 ? 'text-destructive' : ''">¥{{ detailOrder.refundmoney }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">已分账</dt><dd class="text-right">¥{{ detailOrder.profitmoney }}</dd></div>
          </dl>
        </div>

        <div>
          <div class="mb-2 text-xs font-medium text-muted-foreground">支付方式 / 通道</div>
          <dl class="space-y-2">
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">支付方式</dt><dd class="text-right">{{ detailOrder.typeshowname }}（{{ detailOrder.typename }}）</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">通道 ID</dt><dd class="text-right">{{ detailOrder.channel }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">插件</dt><dd class="text-right font-mono">{{ detailOrder.plugin }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">结算状态</dt><dd class="text-right">{{ detailOrder.settle ? '已结算' : '未结算' }}</dd></div>
          </dl>
        </div>

        <div>
          <div class="mb-2 text-xs font-medium text-muted-foreground">时间 / 网络</div>
          <dl class="space-y-2">
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">下单时间</dt><dd class="text-right">{{ detailOrder.addtime }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">完成时间</dt><dd class="text-right">{{ detailOrder.endtime ?? '—' }}</dd></div>
            <div v-if="detailOrder.status === 2" class="flex justify-between gap-4"><dt class="dim shrink-0">退款时间</dt><dd class="text-right">{{ detailOrder.refundtime || '—' }}</dd></div>
            <div class="flex justify-between gap-4">
              <dt class="dim shrink-0">下单 IP</dt>
              <dd class="flex items-center justify-end gap-1.5 text-right font-mono">
                <span>{{ detailOrder.ip || '—' }}</span>
                <button v-if="detailOrder.ip" class="shrink-0 text-muted-foreground hover:text-destructive" title="拉黑此 IP" @click="openBlacklist(1, detailOrder.ip)"><Ban class="size-3.5" /></button>
              </dd>
            </div>
          </dl>
        </div>

        <div>
          <div class="mb-2 text-xs font-medium text-muted-foreground">通知 / 扩展</div>
          <dl class="space-y-2">
            <div class="flex justify-between gap-4">
              <dt class="dim shrink-0">通知状态</dt>
              <dd class="text-right">
                <Badge v-if="!detailOrder.notify" variant="success">通知成功</Badge>
                <Badge v-else-if="detailOrder.notify === -1" variant="muted">已放弃</Badge>
                <Badge v-else variant="destructive">通知失败（已通知 {{ detailOrder.notify }} 次）</Badge>
              </dd>
            </div>
            <div class="flex justify-between gap-4"><dt class="dim shrink-0">扩展参数</dt><dd class="text-right break-all">{{ detailOrder.param || '—' }}</dd></div>
          </dl>
        </div>
      </div>
    </Drawer>

    <!-- 操作确认弹窗（改状态/冻结/解冻/补单/重新通知/删除）-->
    <Modal :model-value="!!confirmState" :title="confirmState?.label ?? ''" @update:model-value="(v) => { if (!v) confirmState = null }">
      <p class="text-sm text-muted-foreground">{{ confirmText }}</p>
      <template #footer>
        <Button variant="outline" size="sm" @click="confirmState = null">取消</Button>
        <Button
          :variant="confirmState?.label === '删除订单' ? 'destructive' : 'default'"
          size="sm"
          :disabled="busy"
          @click="runConfirm"
        >确定</Button>
      </template>
    </Modal>

    <!-- 退款弹窗（手动 / API 原路）-->
    <Modal
      :model-value="!!refundState"
      :title="refundState?.api ? 'API 原路退款' : '手动退款'"
      @update:model-value="(v) => { if (!v) refundState = null }"
    >
      <div class="space-y-3.5">
        <p class="text-sm text-muted-foreground">
          订单 <b class="text-foreground">{{ refundState?.order.trade_no }}</b>
        </p>
        <div v-if="refundInfo" class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          实付 <b class="text-foreground tabular-nums">¥{{ refundInfo.realmoney.toFixed(2) }}</b>，
          已退 <b class="text-foreground tabular-nums">¥{{ refundInfo.refunded.toFixed(2) }}</b>，
          最多可退 <b class="text-destructive tabular-nums">¥{{ refundInfo.refundable.toFixed(2) }}</b>
        </div>
        <div class="row-field">
          <label class="lbl">退款金额</label>
          <div class="flex flex-1 items-center gap-2">
            <span class="text-sm text-muted-foreground">¥</span>
            <input v-model="refundForm.money" placeholder="退款金额" class="field-input flex-1" />
          </div>
        </div>
        <div v-if="refundState?.api" class="row-field">
          <label class="lbl">支付密码</label>
          <input v-model="refundForm.password" type="password" placeholder="原路退款需输入支付密码" class="field-input flex-1" />
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          <template v-if="refundState?.api">
            原路退款将调用渠道退款接口（真实渠道退款待凭证），并按手续费策略从商户余额扣减。
          </template>
          <template v-else>
            手动退款仅从商户余额扣减分成，需线下退款给买家；不调用渠道退款接口。
          </template>
        </p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="refundState = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="busy" @click="submitRefund">确认退款</Button>
      </template>
    </Modal>

    <!-- 批量操作确认 -->
    <Modal :model-value="batchConfirm" title="批量操作" @update:model-value="(v) => { if (!v) batchConfirm = false }">
      <p class="text-sm text-muted-foreground">
        确定对已选的 <b class="text-foreground">{{ selected.size }}</b> 条订单执行
        「<b class="text-foreground">{{ batchOptions.find((b) => b.value === batchAction)?.label }}</b>」吗？
        <span v-if="batchAction === 4" class="text-destructive">删除不可恢复。</span>
        <span v-else-if="batchAction === 2 || batchAction === 3">将对满足前置状态的订单调整商户余额。</span>
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="batchConfirm = false">取消</Button>
        <Button :variant="batchAction === 4 ? 'destructive' : 'default'" size="sm" :disabled="busy" @click="runBatch">执行</Button>
      </template>
    </Modal>

    <!-- 批量 API 退款（对齐 epay batch_apirefund：逐条退款 + 支付密码）-->
    <Modal :model-value="batchRefundDialog" title="批量 API 退款" @update:model-value="(v) => { if (!v) batchRefundDialog = false }">
      <p class="mb-3 text-sm text-muted-foreground">
        将对选中订单中 <b class="text-foreground">{{ refundableSelected.length }}</b> 条可退订单（已完成/解冻）逐条发起原路退款，
        退款金额为各订单实付金额。此操作不可撤销。
      </p>
      <div>
        <label class="mb-1 block text-sm text-muted-foreground">支付密码</label>
        <input v-model="batchRefundPwd" type="password" placeholder="请输入管理员支付密码" class="field-input w-full" />
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="batchRefundDialog = false">取消</Button>
        <Button variant="destructive" size="sm" :disabled="batchRefundRunning" @click="runBatchRefund">确认退款</Button>
      </template>
    </Modal>

    <!-- 一键拉黑（对齐 epay order.php addBlackList：type 1=IP/0=账号，days=0 永久）-->
    <Modal :model-value="blackDialog" :title="`加入黑名单 · ${blackTitle}`" @update:model-value="(v) => { if (!v) blackDialog = false }">
      <div class="space-y-3">
        <div>
          <label class="mb-1 block text-sm text-muted-foreground">{{ blackTitle }}</label>
          <input v-model="blackForm.content" class="field-input w-full font-mono" />
        </div>
        <div>
          <label class="mb-1 block text-sm text-muted-foreground">有效期（天）</label>
          <input v-model.number="blackForm.days" type="number" min="0" placeholder="0" class="field-input w-full tabular-nums" />
          <p class="mt-1 text-xs dim">0 为永久拉黑</p>
        </div>
        <div>
          <label class="mb-1 block text-sm text-muted-foreground">备注</label>
          <input v-model="blackForm.remark" placeholder="选填" class="field-input w-full" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="blackDialog = false">取消</Button>
        <Button variant="destructive" size="sm" :disabled="blackSubmitting" @click="submitBlacklist">加入黑名单</Button>
      </template>
    </Modal>
  </div>
</template>
