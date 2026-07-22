<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
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
} from '@/lib/api/orders'
import { ApiError } from '@/lib/api/client'
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
  Trash2,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, DateRange, Pagination, Drawer, Modal } from '@/components/ui'
import {
  orderStatus,
  payTypes,
  searchColumns,
  calcStats,
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

// ===== 数据源：从后端 API 加载 =====
const allOrders = ref<Order[]>([])
const loading = ref(false)
const loadError = ref('')

async function loadOrders() {
  loading.value = true
  loadError.value = ''
  try {
    // 起步阶段后端返回全量分页，客户端仍做筛选；后续可把筛选下推到后端
    const res = await fetchOrders({ page: 1, pageSize: 100 })
    allOrders.value = res.list
  } catch (e) {
    loadError.value = e instanceof ApiError ? e.message : '加载订单失败'
    allOrders.value = []
  } finally {
    loading.value = false
  }
}

const filtered = computed(() => {
  return allOrders.value.filter((o) => {
    if (filters.value.uid && String(o.uid) !== filters.value.uid.trim()) return false
    if (filters.value.type && o.type !== filters.value.type) return false
    if (filters.value.channel && String(o.channel) !== filters.value.channel.trim()) return false
    if (filters.value.dstatus > -1 && o.status !== filters.value.dstatus) return false
    if (filters.value.value.trim()) {
      const v = filters.value.value.trim()
      const field = (o as any)[filters.value.column]
      if (field == null || !String(field).includes(v)) return false
    }
    return true
  })
})

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
}

// ===== 分页 =====
const page = ref(1)
const pageSize = 15
const total = computed(() => filtered.value.length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
// 当前页做 clamp，避免筛选后结果变少、page 停留在旧页导致表格空白
const safePage = computed(() => Math.min(page.value, pageCount.value))
const pageRows = computed(() =>
  filtered.value.slice((safePage.value - 1) * pageSize, safePage.value * pageSize),
)
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
}

// ===== 统计 =====
const showStats = ref(false)
const stats = computed(() => calcStats(filtered.value))

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
onMounted(() => {
  window.addEventListener('click', closeMenu)
  loadOrders()
})
onUnmounted(() => window.removeEventListener('click', closeMenu))

function actionsFor(o: Order): string[] {
  if (o.status === 1) return ['改未完成', 'API退款', '手动退款', '冻结订单', '-', '重新通知', '删除订单']
  if (o.status === 2) return ['改未完成', 'API退款', '改已完成', '-', '重新通知', '删除订单']
  if (o.status === 3) return ['解冻订单', 'API退款', '-', '重新通知', '删除订单']
  const base = ['改已完成', '-', '手动补单', '删除订单']
  if (o.status === 4) return ['授权资金支付', '授权资金解冻', '-', ...base]
  return base
}

// 操作项 → 图标
const actionIcons: Record<string, any> = {
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
  if (r.api && !refundForm.value.password) return toast.error('原路退款需输入管理员密码')
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

// 菜单项点击调度
function onAction(label: string, o: Order) {
  switch (label) {
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
  { value: 4, label: '批量删除' },
]
const batchConfirm = ref(false)
function askBatch() {
  if (selected.value.size === 0) return toast.info('请先勾选订单')
  batchConfirm.value = true
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
// 按导出条件过滤订单（count 与实际导出共用同一套逻辑，避免"预估≠导出"）
function filterForExport(): Order[] {
  const f = exportForm.value
  const start = f.starttime // 'YYYY-MM-DD'
  const end = f.endtime
  return allOrders.value.filter((o) => {
    // 时间范围：按 addtime 的日期部分闭区间比较（含起止当天）
    const day = (o.addtime || '').slice(0, 10)
    if (start && day < start) return false
    if (end && day > end) return false
    if (f.uid && String(o.uid) !== f.uid.trim()) return false
    if (f.type && o.type !== f.type) return false
    if (f.channel && String(o.channel) !== f.channel.trim()) return false
    if (f.dstatus > -1 && o.status !== f.dstatus) return false
    return true
  })
}
// 导出预估命中条数（给用户导出前的量级参考）
const exportCount = computed(() => filterForExport().length)
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
// CSV 单元格转义：含逗号/引号/换行时用双引号包裹并转义内部引号
function csvCell(v: string | number | null | undefined): string {
  const s = v == null ? '' : String(v)
  if (/[",\n]/.test(s)) return '"' + s.replace(/"/g, '""') + '"'
  return s
}
function submitExport() {
  const rows = filterForExport()
  if (rows.length === 0) {
    toast.info('当前条件下没有可导出的订单')
    return
  }
  const headers = [
    '系统订单号', '商户订单号', '接口订单号', '商户号', '商品名称',
    '订单金额', '实际支付', '商户分成', '已退款', '手续费利润',
    '支付方式', '通道ID', '插件', '支付IP', '支付账号',
    '创建时间', '完成时间', '订单状态',
  ]
  const lines = rows.map((o) =>
    [
      o.trade_no, o.out_trade_no, o.api_trade_no, o.uid, o.name,
      o.money, o.realmoney ?? '', o.getmoney, o.refundmoney, o.profitmoney,
      o.typeshowname, o.channel, o.plugin, o.ip, o.buyer,
      o.addtime, o.endtime ?? '', orderStatus[o.status]?.text ?? o.status,
    ].map(csvCell).join(','),
  )
  // 加 BOM，保证 Excel 打开中文不乱码
  const csv = '﻿' + [headers.join(','), ...lines].join('\r\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  const range = `${exportForm.value.starttime}_${exportForm.value.endtime}`
  a.href = url
  a.download = `订单导出_${range}.csv`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  toast.success(`已导出 ${rows.length} 条订单`)
  exportOpen.value = false
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选（标题即页面标题，参考图“标签+控件”横排样式） -->
    <Panel title="订单管理" :subtitle="`共 ${total} 笔订单`">
      <template #actions>
        <Button variant="outline" size="sm" @click="showStats = !showStats">
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
            <Button size="sm" @click="page = 1"><Search />搜索</Button>
            <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
          </div>
        </div>
      </div>
    </Panel>

    <!-- 统计概况（可展开） -->
    <Panel v-if="showStats" title="订单统计概况" subtitle="按当前筛选条件">
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
              <th class="w-[14%]">订单号 / 商户单号</th>
              <th class="w-[12%]">商户 / 域名</th>
              <th class="w-[13%]">商品 / 金额</th>
              <th class="w-[11%]">实付 / 分成</th>
              <th class="w-[13%]">支付方式</th>
              <th class="w-[14%]">创建 / 完成时间</th>
              <th class="col-center w-[9%]">状态</th>
              <th class="col-center w-[8%]">操作</th>
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
              <td colspan="9" class="py-10 text-center dim">没有符合条件的订单</td>
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
          时间范围为必填。按当前条件预计导出
          <b class="text-foreground tabular-nums">{{ exportCount }}</b> 条订单。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="exportOpen = false">取消</Button>
        <Button size="sm" :disabled="!exportForm.starttime || !exportForm.endtime" @click="submitExport">
          <Download />导出 CSV
        </Button>
      </template>
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
          <label class="lbl">管理员密码</label>
          <input v-model="refundForm.password" type="password" placeholder="原路退款需二次校验" class="field-input flex-1" />
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
  </div>
</template>
