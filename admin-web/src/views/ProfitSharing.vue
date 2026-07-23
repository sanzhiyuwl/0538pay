<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import {
  Search, RotateCcw, Download, BarChart3, MoreHorizontal,
  Send, Undo2, RefreshCw, XCircle, Plus, Pencil, Trash2,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, DateRange, Pagination, Modal, Drawer, Switch } from '@/components/ui'
import {
  fetchPsOrders, fetchPsStats, operatePsOrder,
  fetchPsReceivers, createPsReceiver, updatePsReceiver, setPsReceiverStatus, deletePsReceiver,
  type PsOrder, type PsStats, type PsReceiver, type PsReceiverReq,
} from '@/lib/api/profitsharing'
import { fetchChannels } from '@/lib/api/channels'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { shouldDropUp } from '@/composables/useRowMenu'
import { formatMoney, exportCsv } from '@/lib/utils'

const toast = useToast()

// 状态字典（对齐后端 0待分账/1已提交/2成功/3失败/4取消）
const psStatus: Record<number, { text: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }> = {
  0: { text: '待分账', variant: 'default' },
  1: { text: '已提交', variant: 'warning' },
  2: { text: '分账成功', variant: 'success' },
  3: { text: '分账失败', variant: 'destructive' },
  4: { text: '已取消', variant: 'muted' },
}
const columnOptions = [
  { value: 'trade_no', label: '系统订单号' },
  { value: 'api_trade_no', label: '接口订单号' },
  { value: 'money', label: '分账金额' },
]
const statusOptions = [
  { value: -1, label: '全部状态' },
  ...Object.entries(psStatus).map(([k, s]) => ({ value: Number(k), label: s.text })),
]

// ===== 筛选 =====
const filters = reactive({ column: 'trade_no', value: '', rid: '', starttime: '', endtime: '', dstatus: -1 })

// ===== 分页 + 数据 =====
const page = ref(1)
const pageSize = 15
const total = ref(0)
const rows = ref<PsOrder[]>([])
const loading = ref(false)

function buildParams() {
  const ridNum = Number(filters.rid.trim())
  return {
    page: page.value,
    pageSize,
    rid: filters.rid.trim() && !Number.isNaN(ridNum) ? ridNum : undefined,
    status: filters.dstatus > -1 ? filters.dstatus : undefined,
    column: filters.value.trim() ? filters.column : undefined,
    value: filters.value.trim() || undefined,
    starttime: filters.starttime || undefined,
    endtime: filters.endtime || undefined,
  }
}
async function load() {
  loading.value = true
  try {
    const res = await fetchPsOrders(buildParams())
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载分账记录失败')
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}
// 导出（按当前筛选条件从后端拉全量再生成 CSV）
const exporting = ref(false)
async function exportList() {
  if (exporting.value) return
  exporting.value = true
  try {
    const res = await fetchPsOrders({ ...buildParams(), page: 1, pageSize: 10000 })
    const list = res.list
    if (!list.length) { toast.error('没有可导出的分账记录'); return }
    const headers = ['系统订单号', '接口订单号', '分账规则', '通道', '接收方', '分账金额', '创建时间', '状态', '结果']
    const data = list.map((o) => [
      o.trade_no, o.api_trade_no, o.rulename, o.channelname, o.receiver, o.money, o.addtime,
      psStatus[o.status]?.text ?? o.status, o.result,
    ])
    exportCsv(`分账记录_${new Date().toISOString().slice(0, 10)}`, headers, data)
    toast.success(`已导出 ${list.length} 条分账记录`)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '导出失败')
  } finally {
    exporting.value = false
  }
}

async function loadStats() {
  if (!showStats.value) return
  try {
    const { page: _p, pageSize: _ps, ...rest } = buildParams()
    void _p; void _ps
    stats.value = await fetchPsStats(rest)
  } catch {
    stats.value = null
  }
}
async function reload() {
  await Promise.all([load(), loadStats()])
}
function applySearch() {
  page.value = 1
  reload()
}
function resetFilters() {
  filters.column = 'trade_no'
  filters.value = ''
  filters.rid = ''
  filters.starttime = ''
  filters.endtime = ''
  filters.dstatus = -1
  applySearch()
}
function go(p: number) {
  page.value = p
  load()
}
onMounted(load)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// ===== 统计 =====
const showStats = ref(false)
const stats = ref<PsStats | null>(null)
async function toggleStats() {
  showStats.value = !showStats.value
  if (showStats.value) await loadStats()
}

// ===== 行操作菜单 =====
const openMenu = ref<number | null>(null)
const dropUp = ref(false)
function toggleMenu(id: number, ev?: MouseEvent) {
  if (openMenu.value === id) { openMenu.value = null; return }
  openMenu.value = id
  dropUp.value = shouldDropUp(ev)
}
function closeMenu() {
  openMenu.value = null
}
onMounted(() => window.addEventListener('click', closeMenu))
onUnmounted(() => window.removeEventListener('click', closeMenu))

// ===== 多选 + 批量操作（对齐 epay ps_order batch_act：submit/return/cancel + 删除）=====
const selected = ref<Set<number>>(new Set())
const pageAllChecked = computed(
  () => rows.value.length > 0 && rows.value.every((r) => selected.value.has(r.id)),
)
function toggleAll() {
  if (pageAllChecked.value) rows.value.forEach((r) => selected.value.delete(r.id))
  else rows.value.forEach((r) => selected.value.add(r.id))
  selected.value = new Set(selected.value)
}
function toggleOne(id: number) {
  if (selected.value.has(id)) selected.value.delete(id)
  else selected.value.add(id)
  selected.value = new Set(selected.value)
}
// 批量动作的可执行状态门槛（对齐后端 Operate 状态校验）
const batchGate: Record<string, number[]> = {
  submit: [0, 3], // 待分账/失败可提交
  cancel: [0, 3], // 待分账/失败可取消
  return: [2],    // 成功可回退
  delete: [0, 3, 4], // 未扣款态可删（已扣款后端会拦）
}
const batchLabels: Record<string, string> = {
  submit: '批量提交分账', cancel: '批量取消', return: '批量回退', delete: '批量删除',
}
const bulkAction = ref<'submit' | 'cancel' | 'return' | 'delete'>('submit')
const bulkOptions = [
  { value: 'submit', label: '提交分账' },
  { value: 'return', label: '分账回退' },
  { value: 'cancel', label: '取消' },
  { value: 'delete', label: '删除' },
]
const bulkConfirm = ref(false)
// 选中行里满足当前动作门槛的
const bulkEligible = computed(() =>
  rows.value.filter((r) => selected.value.has(r.id) && batchGate[bulkAction.value].includes(r.status)),
)
function askBulk() {
  if (!selected.value.size) return toast.info('请先勾选分账记录')
  if (!bulkEligible.value.length) {
    return toast.info(`选中记录中没有可「${bulkOptions.find((o) => o.value === bulkAction.value)?.label}」的记录`)
  }
  bulkConfirm.value = true
}
async function runBulk() {
  if (busy.value) return
  busy.value = true
  let ok = 0
  const targets = bulkEligible.value
  try {
    for (const r of targets) {
      try {
        await operatePsOrder(r.id, bulkAction.value)
        ok++
      } catch {
        /* 单条失败继续 */
      }
    }
    toast.success(`${batchLabels[bulkAction.value]}完成：${ok}/${targets.length} 条成功`)
    bulkConfirm.value = false
    selected.value = new Set()
    await reload()
  } finally {
    busy.value = false
  }
}

// ===== 分账金额编辑（J-6，对齐 epay editmoney：仅 status=0 待分账可改）=====
const editMoneyRow = ref<PsOrder | null>(null)
const editMoneyVal = ref('')
const editMoneySaving = ref(false)
function openEditMoney(r: PsOrder) {
  openMenu.value = null
  editMoneyRow.value = r
  editMoneyVal.value = r.money
}
async function saveEditMoney() {
  const r = editMoneyRow.value
  if (!r || editMoneySaving.value) return
  const v = Number(editMoneyVal.value)
  if (!(v > 0)) return toast.error('请输入有效的分账金额')
  editMoneySaving.value = true
  try {
    await operatePsOrder(r.id, 'editmoney', editMoneyVal.value.trim())
    toast.success('分账金额已更新')
    editMoneyRow.value = null
    await reload()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '修改失败')
  } finally {
    editMoneySaving.value = false
  }
}

// 状态可执行操作（对齐 epay ps_order + 后端能力）
function psActions(status: number): { key: string; label: string }[] {
  if (status === 0) return [{ key: 'submit', label: '提交分账' }, { key: 'editmoney', label: '修改金额' }, { key: 'cancel', label: '取消' }, { key: 'delete', label: '删除' }]
  if (status === 1) return [{ key: 'query', label: '查询结果' }]
  if (status === 2) return [{ key: 'return', label: '分账回退' }]
  if (status === 3) return [{ key: 'submit', label: '重试' }, { key: 'cancel', label: '取消' }, { key: 'delete', label: '删除' }]
  if (status === 4) return [{ key: 'delete', label: '删除' }]
  return []
}
const actionIcons: Record<string, any> = {
  submit: Send, query: RefreshCw, return: Undo2, cancel: XCircle, editmoney: Pencil, delete: Trash2,
}

// ===== 操作确认（资金相关：submit 扣款 / return/cancel 退回）=====
const busy = ref(false)
const confirmOpen = ref(false)
const confirmRow = ref<PsOrder | null>(null)
const confirmAction = ref('')
const confirmLabel = ref('')
const confirmText = computed(() => {
  const r = confirmRow.value
  if (!r) return ''
  const m = `¥${formatMoney(r.money)}`
  switch (confirmAction.value) {
    case 'submit':
      return `确认提交分账？若该规则绑定了商户，将从其余额扣除分账金额 ${m}。真实渠道分账 API 待接入，当前直接置为成功。`
    case 'query':
      return '确认向渠道查询该笔分账结果？'
    case 'return':
      return `确认回退该笔已成功分账？将把 ${m} 退回原扣款商户余额，状态置为已取消。`
    case 'cancel':
      return `确认取消该笔分账？若已扣款将退回商户 ${m}，状态置为已取消。`
    case 'delete':
      return '确认删除该笔分账记录？已扣款的记录需先回退再删除，此操作不可恢复。'
    default:
      return ''
  }
})

function onAction(r: PsOrder, key: string, label: string) {
  openMenu.value = null
  // 改金额走独立弹窗
  if (key === 'editmoney') return openEditMoney(r)
  confirmRow.value = r
  confirmAction.value = key
  confirmLabel.value = label
  confirmOpen.value = true
}
async function doConfirm() {
  const r = confirmRow.value
  if (!r || busy.value) return
  busy.value = true
  try {
    await operatePsOrder(r.id, confirmAction.value)
    toast.success(`${confirmLabel.value}成功`)
    confirmOpen.value = false
    await reload()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  } finally {
    busy.value = false
  }
}

// ===== 分账规则管理（ps_receiver，C-1）=====
const rules = ref<PsReceiver[]>([])
const rulesLoading = ref(false)
const channelOpts = ref<{ value: number; label: string }[]>([])

async function loadRules() {
  rulesLoading.value = true
  try {
    const { list } = await fetchPsReceivers()
    rules.value = list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '获取分账规则失败')
  } finally {
    rulesLoading.value = false
  }
}
async function loadChannelOpts() {
  try {
    const res = await fetchChannels({ pageSize: 200 })
    channelOpts.value = res.list.map((c) => ({ value: c.id, label: `${c.name}（#${c.id}）` }))
  } catch {
    channelOpts.value = []
  }
}

// 规则新增/编辑抽屉
const ruleDrawer = ref(false)
const ruleEditID = ref<number | null>(null)
const ruleSaving = ref(false)
const ruleForm = reactive<PsReceiverReq>({ channel: 0, subchannel: 0, uid: 0, account: '', name: '', rate: '30', minmoney: '0' })

function openRuleCreate() {
  ruleEditID.value = null
  Object.assign(ruleForm, { channel: channelOpts.value[0]?.value ?? 0, subchannel: 0, uid: 0, account: '', name: '', rate: '30', minmoney: '0' })
  ruleDrawer.value = true
}
function openRuleEdit(r: PsReceiver) {
  ruleEditID.value = r.id
  Object.assign(ruleForm, { channel: r.channel, subchannel: r.subchannel, uid: r.uid, account: r.account, name: r.name, rate: r.rate, minmoney: r.minmoney })
  ruleDrawer.value = true
}
async function saveRule() {
  if (!ruleForm.channel || !ruleForm.account.trim()) {
    toast.error('支付通道和接收方账号为必填项')
    return
  }
  ruleSaving.value = true
  try {
    if (ruleEditID.value !== null) {
      await updatePsReceiver(ruleEditID.value, { ...ruleForm })
      toast.success('修改分账规则成功')
    } else {
      await createPsReceiver({ ...ruleForm })
      toast.success('新增分账规则成功')
    }
    ruleDrawer.value = false
    await loadRules()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    ruleSaving.value = false
  }
}
async function toggleRule(r: PsReceiver) {
  const next = r.status === 1 ? 0 : 1
  try {
    await setPsReceiverStatus(r.id, next)
    r.status = next
    toast.success('状态已更新')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
}
const ruleDelTarget = ref<PsReceiver | null>(null)
async function confirmDelRule() {
  if (!ruleDelTarget.value) return
  try {
    await deletePsReceiver(ruleDelTarget.value.id)
    toast.success('删除成功')
    ruleDelTarget.value = null
    await loadRules()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  }
}

// 多接收方拆分展示（P0-a，对齐 epay | 分隔）：account/name/rate 按 | 拆分对齐，
// rate 段缺省复用首段。返回逐接收方 { account, name, rate } 供表格分行展示。
function splitReceivers(r: PsReceiver): { account: string; name: string; rate: string }[] {
  const accounts = r.account.split('|').map((s) => s.trim()).filter((s) => s !== '')
  const names = r.name.split('|')
  const rates = r.rate.split('|')
  return accounts.map((account, i) => ({
    account,
    name: (names[i] ?? '').trim(),
    rate: (rates[i] ?? rates[0] ?? '').trim(),
  }))
}
function isMultiReceiver(r: PsReceiver): boolean {
  return r.account.includes('|')
}

onMounted(() => {
  loadRules()
  loadChannelOpts()
})
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="分账记录" :subtitle="`共 ${total} 笔分账`">
      <template #actions>
        <Button variant="outline" size="sm" @click="toggleStats"><BarChart3 />统计</Button>
        <Button variant="outline" size="sm" :disabled="exporting" @click="exportList"><Download />导出列表</Button>
      </template>
      <div class="space-y-3">
        <div class="filter-bar">
          <div class="filter-item">
            <label class="filter-label">分账信息</label>
            <Select v-model="filters.column" :options="columnOptions" class="w-32" />
            <input v-model="filters.value" placeholder="搜索内容" class="field-input w-48" @keyup.enter="applySearch" />
          </div>
          <div class="filter-item">
            <label class="text-sm text-muted-foreground">分账规则</label>
            <input v-model="filters.rid" placeholder="规则 ID" class="field-input w-28" @keyup.enter="applySearch" />
          </div>
          <div class="filter-item">
            <label class="text-sm text-muted-foreground">分账状态</label>
            <Select v-model="filters.dstatus" :options="statusOptions" class="w-28" />
          </div>
        </div>
        <div class="filter-bar">
          <div class="filter-item">
            <label class="filter-label">分账时间</label>
            <DateRange v-model:start="filters.starttime" v-model:end="filters.endtime" class="w-[328px]" />
          </div>
          <div class="ml-auto flex items-center gap-2">
            <Button size="sm" @click="applySearch"><Search />搜索</Button>
            <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
          </div>
        </div>
      </div>
    </Panel>

    <!-- 统计概况 -->
    <Panel v-if="showStats && stats" title="分账统计概况" subtitle="按当前筛选条件">
      <div class="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3">
        <div>
          <div class="text-[13px] text-muted-foreground">分账总金额</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.totalMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">成功分账金额</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-success"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.successMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">失败金额</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-destructive"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.failMoney) }}</div>
        </div>
      </div>
      <div class="mt-5 flex flex-wrap gap-x-8 gap-y-2 border-t border-border/70 pt-4 text-sm">
        <span class="text-muted-foreground">分账总数 <b class="text-foreground">{{ stats.totalCount }}</b></span>
        <span class="text-muted-foreground">成功 <b class="text-foreground">{{ stats.successCount }}</b></span>
        <span class="text-muted-foreground">失败 <b class="text-foreground">{{ stats.failCount }}</b></span>
        <span class="text-muted-foreground">成功率 <b class="text-primary">{{ stats.successRate }}%</b></span>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="分账列表" :subtitle="selected.size ? `已选 ${selected.size} 条` : `${total} 条`">
      <template v-if="selected.size" #actions>
        <span class="text-sm text-muted-foreground">批量操作：</span>
        <Select v-model="bulkAction" :options="bulkOptions" class="w-32" />
        <Button size="sm" :disabled="busy" @click="askBulk">执行</Button>
        <Button variant="ghost" size="sm" @click="selected = new Set()">清空选择</Button>
      </template>
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[4%] col-center">
                <input type="checkbox" :checked="pageAllChecked" class="align-middle" @change="toggleAll" />
              </th>
              <th class="w-[18%]">系统订单号</th>
              <th class="w-[16%]">分账规则 / 接收方</th>
              <th class="w-[14%]">支付通道</th>
              <th class="w-[11%]">分账金额</th>
              <th class="w-[14%]">时间</th>
              <th class="col-center w-[10%]">状态</th>
              <th class="col-center w-[12%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in rows" :key="o.id">
              <td class="col-center">
                <input type="checkbox" :checked="selected.has(o.id)" class="align-middle" @change="toggleOne(o.id)" />
              </td>
              <td>
                <div class="truncate font-medium text-primary">{{ o.trade_no }}</div>
                <div class="truncate text-xs dim">{{ o.api_trade_no || '—' }}</div>
              </td>
              <td>
                <div class="truncate">{{ o.rulename }}</div>
                <div class="truncate text-xs dim">{{ o.receiver }}</div>
              </td>
              <td>
                <div>{{ o.channelname || '—' }}</div>
                <div class="text-xs dim tabular-nums">#{{ o.channelid }}</div>
              </td>
              <td>
                <div class="tabular-nums"><span class="dim text-xs">¥</span><b>{{ formatMoney(o.money) }}</b></div>
              </td>
              <td>
                <div class="text-xs">{{ o.addtime }}</div>
              </td>
              <td class="col-center">
                <Badge :variant="psStatus[o.status].variant">{{ psStatus[o.status].text }}</Badge>
                <div v-if="o.status === 3 && o.result" class="mt-1 truncate text-xs text-destructive" :title="o.result">
                  {{ o.result }}
                </div>
              </td>
              <td class="col-center">
                <div v-if="psActions(o.status).length" class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(o.id, $event)">
                    操作 <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === o.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button
                      v-for="(a, ai) in psActions(o.status)"
                      :key="ai"
                      class="menu-item"
                      :class="(a.key === 'cancel' || a.key === 'delete') && 'menu-item-danger'"
                      @click="onAction(o, a.key, a.label)"
                    >
                      <component :is="actionIcons[a.key]" class="size-4 shrink-0 opacity-70" />
                      <span class="flex-1">{{ a.label }}</span>
                    </button>
                  </div>
                </div>
                <span v-else class="dim">—</span>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="8" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="8" class="py-10 text-center dim">没有符合条件的分账记录</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 操作确认弹窗 -->
    <Modal v-model="confirmOpen" :title="`${confirmLabel}确认`" width="max-w-md">
      <p class="text-sm text-muted-foreground">{{ confirmText }}</p>
      <template #footer>
        <Button variant="outline" size="sm" @click="confirmOpen = false">取消</Button>
        <Button size="sm" :disabled="busy" @click="doConfirm">确认{{ confirmLabel }}</Button>
      </template>
    </Modal>

    <!-- 批量操作确认（对齐 epay ps_order batch_act）-->
    <Modal :model-value="bulkConfirm" title="批量操作确认" @update:model-value="(v) => { if (!v) bulkConfirm = false }">
      <p class="text-sm text-muted-foreground">
        将对选中记录中 <b class="text-foreground">{{ bulkEligible.length }}</b> 条可执行记录执行
        「<b class="text-foreground">{{ bulkOptions.find((o) => o.value === bulkAction)?.label }}</b>」。
        <span v-if="bulkAction === 'submit'">提交后若规则绑定商户将从其余额扣除分账金额。</span>
        <span v-else-if="bulkAction === 'return'" class="text-destructive">回退将把已成功分账金额退回原扣款商户。</span>
        <span v-else-if="bulkAction === 'delete'" class="text-destructive">删除不可恢复，已扣款记录会被跳过。</span>
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="bulkConfirm = false">取消</Button>
        <Button :variant="bulkAction === 'delete' ? 'destructive' : 'default'" size="sm" :disabled="busy" @click="runBulk">执行</Button>
      </template>
    </Modal>

    <!-- 分账金额编辑（J-6，对齐 epay editmoney：仅待分账可改）-->
    <Modal :model-value="!!editMoneyRow" title="修改分账金额" @update:model-value="(v) => { if (!v) editMoneyRow = null }">
      <p class="mb-3 text-sm text-muted-foreground">
        订单 <b class="text-foreground">{{ editMoneyRow?.trade_no }}</b>，仅待分账状态可修改。
      </p>
      <div class="row-field">
        <label class="lbl">分账金额</label>
        <input v-model="editMoneyVal" class="field-input flex-1 tabular-nums" placeholder="请输入分账金额" @keyup.enter="saveEditMoney" />
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="editMoneyRow = null">取消</Button>
        <Button size="sm" :disabled="editMoneySaving" @click="saveEditMoney">保存</Button>
      </template>
    </Modal>

    <!-- 分账规则管理（ps_receiver，C-1）-->
    <Panel title="分账规则管理" :subtitle="`共 ${rules.length} 条规则`">
      <template #actions>
        <Button size="sm" @click="openRuleCreate"><Plus />新增规则</Button>
      </template>
      <p class="mb-3 text-xs text-muted-foreground">
        每个「通道 + 商户」只能配置一条规则。规则开启后下单命中即按比例创建分账单。
        绑定商户的规则从其余额扣款；真实渠道分账接收方同步待渠道凭证。
      </p>
      <div class="overflow-x-auto">
        <table class="tbl w-full">
          <thead>
            <tr>
              <th class="w-16">ID</th>
              <th>通道</th>
              <th>绑定商户</th>
              <th>接收方账号 / 姓名</th>
              <th class="w-24">分账比例</th>
              <th class="w-28">订单门槛</th>
              <th class="w-20 col-center">状态</th>
              <th class="w-28 col-center">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in rules" :key="r.id">
              <td class="dim tabular-nums">#{{ r.id }}</td>
              <td>{{ r.channel_name || `通道#${r.channel}` }}<span v-if="r.subchannel" class="ml-1 text-xs dim">子#{{ r.subchannel }}</span></td>
              <td>
                <span v-if="r.uid">商户 {{ r.uid }}</span>
                <span v-else class="dim">通道级全局</span>
              </td>
              <td>
                <template v-if="isMultiReceiver(r)">
                  <div v-for="(rc, ri) in splitReceivers(r)" :key="ri" class="truncate" :title="rc.account">
                    <span>{{ rc.account }}</span>
                    <span v-if="rc.name" class="ml-1 text-xs dim">（{{ rc.name }}）</span>
                    <span class="ml-1 text-xs dim tabular-nums">{{ rc.rate }}%</span>
                  </div>
                </template>
                <template v-else>
                  <div class="truncate" :title="r.account">{{ r.account }}</div>
                  <div v-if="r.name" class="truncate text-xs dim">{{ r.name }}</div>
                </template>
              </td>
              <td class="tabular-nums">
                <span v-if="isMultiReceiver(r)" class="text-xs dim">多接收方</span>
                <span v-else>{{ r.rate }}%</span>
              </td>
              <td class="tabular-nums"><span class="dim text-xs">¥</span>{{ r.minmoney }}</td>
              <td class="col-center">
                <Switch :model-value="r.status === 1" @update:model-value="toggleRule(r)" />
              </td>
              <td class="col-center">
                <button class="inline-flex size-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground" title="编辑" @click="openRuleEdit(r)"><Pencil class="size-4" /></button>
                <button class="inline-flex size-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-destructive" title="删除" @click="ruleDelTarget = r"><Trash2 class="size-4" /></button>
              </td>
            </tr>
            <tr v-if="rulesLoading"><td colspan="8" class="py-8 text-center dim">加载中…</td></tr>
            <tr v-else-if="!rules.length"><td colspan="8" class="py-8 text-center dim">暂无分账规则，点击右上角新增</td></tr>
          </tbody>
        </table>
      </div>
    </Panel>

    <!-- 规则新增/编辑抽屉 -->
    <Drawer
      v-model="ruleDrawer"
      :title="ruleEditID !== null ? '编辑分账规则' : '新增分账规则'"
      subtitle="配置通道、接收方与分账比例"
    >
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">支付通道<span class="text-destructive">*</span></label>
          <Select v-model="ruleForm.channel" :options="channelOpts" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">子通道 ID</label>
          <input v-model.number="ruleForm.subchannel" type="number" min="0" placeholder="0=不限" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">绑定商户 UID</label>
          <input v-model.number="ruleForm.uid" type="number" min="0" placeholder="0=通道级全局（不绑定商户）" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">接收方账号<span class="text-destructive">*</span></label>
          <input v-model="ruleForm.account" placeholder="多接收方用 | 分隔" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">接收方姓名</label>
          <input v-model="ruleForm.name" placeholder="可空，多个用 | 分隔" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">分账比例 %</label>
          <input v-model="ruleForm.rate" placeholder="默认 30；多接收方用 | 分隔" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">订单最小金额</label>
          <input v-model="ruleForm.minmoney" placeholder="0=不限，订单金额≥此值才分账" class="field-input flex-1" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="ruleDrawer = false">取消</Button>
        <Button size="sm" :disabled="ruleSaving" @click="saveRule">{{ ruleEditID !== null ? '保存' : '创建' }}</Button>
      </template>
    </Drawer>

    <!-- 规则删除确认 -->
    <Modal :model-value="!!ruleDelTarget" title="删除分账规则" @update:model-value="(v) => { if (!v) ruleDelTarget = null }">
      <p class="text-sm text-muted-foreground">
        确定删除规则 <b class="text-foreground">#{{ ruleDelTarget?.id }}</b>（接收方 {{ ruleDelTarget?.account }}）吗？此操作不可恢复。
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="ruleDelTarget = null">取消</Button>
        <Button variant="destructive" size="sm" @click="confirmDelRule">删除</Button>
      </template>
    </Modal>
  </div>
</template>
