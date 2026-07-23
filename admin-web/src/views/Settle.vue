<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import {
  Search,
  RotateCcw,
  Download,
  Plus,
  Send,
  CheckCircle2,
  Layers,
  MoreHorizontal,
  Clock,
  Undo2,
  Trash2,
  Pencil,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination, Modal, Drawer, DateRange } from '@/components/ui'
import {
  settleTypes,
  settleStatus,
  batchStatus,
  type SettleRecord,
  type SettleBatch,
} from '@/lib/mock/settle'
import {
  fetchSettles,
  fetchSettleStats,
  exportSettles,
  type SettleListParams,
  type SettleStats,
  fetchSettleBatches,
  createSettleBatch,
  completeSettleBatch,
  setSettleStatus,
  exportSettleBatch,
} from '@/lib/api/settle'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { shouldDropUp } from '@/composables/useRowMenu'
import { formatMoney } from '@/lib/utils'

const toast = useToast()

// ===== 真接口数据 =====
// 批次列表：epay 批次数量本就少（settle.php 批次表全量），一次拉取即可。
// 明细列表：改服务端分页/筛选（对齐 epay ajax_settle settleList 的 offset/limit + WHERE 下推），
// 概况另走全量聚合接口（epay 明细列表本身无金额合计，我方概况须覆盖全部匹配记录而非当前页）。
const batches = ref<SettleBatch[]>([])

async function loadBatches() {
  try {
    const res = await fetchSettleBatches({ page: 1, pageSize: 100 })
    batches.value = res.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '批次加载失败')
    batches.value = []
  }
}
async function reload() {
  await Promise.all([loadBatches(), loadRecords()])
}
const route = useRoute()
onMounted(() => {
  // 从商户页「查看结算」快捷跳转而来：预置商户号筛选
  const qu = route.query.uid
  if (qu != null && String(qu).trim()) filters.value.uid = String(qu).trim()
  reload()
})

// ===== 结算方式 / 状态下拉 =====
const typeOptions = [
  { value: 0, label: '所有结算方式' },
  ...Object.entries(settleTypes).map(([k, t]) => ({ value: Number(k), label: t.showname })),
]
const statusOptions = [
  { value: -1, label: '全部状态' },
  ...Object.entries(settleStatus).map(([k, s]) => ({ value: Number(k), label: s.text })),
]
const batchOptions = computed(() => [
  { value: '', label: '全部批次' },
  ...batches.value.map((b) => ({ value: b.batch, label: b.batch })),
])

// ===== 筛选 =====
const filters = ref({
  value: '', // 结算账号/姓名
  uid: '',
  batch: '',
  type: 0,
  dstatus: -1,
})

// ===== 分页（服务端）=====
const page = ref(1)
const pageSize = 15
const total = ref(0)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))

// 把当前筛选映射为后端 SettleListParams（列表分页与筛选整体下推，对齐 epay ajax_settle）。
function buildParams(): SettleListParams {
  const f = filters.value
  const p: SettleListParams = { page: page.value, pageSize }
  if (f.value.trim()) p.keyword = f.value.trim()
  if (f.uid.trim()) p.uid = Number(f.uid.trim()) || 0
  if (f.batch) p.batch = f.batch
  if (f.type) p.type = f.type
  if (f.dstatus > -1) p.status = f.dstatus
  return p
}

// ===== 数据源：明细列表（服务端分页）+ 概况（全量聚合，同筛选）=====
const pageRows = ref<SettleRecord[]>([])
const stats = ref<SettleStats>({
  totalMoney: '0', realMoney: '0', doneMoney: '0',
  totalCount: 0, doneCount: 0, pendingCount: 0, processingCount: 0, failCount: 0,
})
async function loadRecords() {
  try {
    const p = buildParams()
    const res = await fetchSettles(p)
    pageRows.value = res.list
    total.value = res.total
    const { page: _p, pageSize: _ps, ...f } = p
    stats.value = await fetchSettleStats(f)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '结算明细加载失败')
    pageRows.value = []
    total.value = 0
  }
}

function resetFilters() {
  filters.value = { value: '', uid: '', batch: '', type: 0, dstatus: -1 }
  page.value = 1
  selected.value.clear()
  loadRecords()
}
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
  loadRecords()
}

// 搜索 / 切换筛选：回到第一页、清空跨筛选选中、重新拉后端
function applySearch() {
  page.value = 1
  selected.value.clear()
  loadRecords()
}
function viewBatch(batch: string) {
  filters.value.batch = batch
  applySearch()
}

// ===== 多选（批量修改状态）=====
const selected = ref<Set<number>>(new Set())
const pageAllChecked = computed(
  () => pageRows.value.length > 0 && pageRows.value.every((r) => selected.value.has(r.id)),
)
function toggleAll() {
  if (pageAllChecked.value) {
    pageRows.value.forEach((r) => selected.value.delete(r.id))
  } else {
    pageRows.value.forEach((r) => selected.value.add(r.id))
  }
}
function toggleOne(id: number) {
  if (selected.value.has(id)) selected.value.delete(id)
  else selected.value.add(id)
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
  exportMenuBatch.value = null
}
onMounted(() => window.addEventListener('click', closeMenu))
onUnmounted(() => window.removeEventListener('click', closeMenu))

// 结算方式对应图标底色（用文字缩写块，无需外部图标资源）
function typeInitial(type: number) {
  return settleTypes[type]?.showname?.[0] ?? '?'
}

// ===== 写操作（调真接口，成功后重拉）=====
const busy = ref(false)

// 生成结算批次：收当前所有待结算记录
async function doCreateBatch() {
  if (busy.value) return
  busy.value = true
  try {
    const res = await createSettleBatch()
    toast.success(`已生成批次 ${res.batch}，收入 ${res.count} 条待结算记录`)
    await reload()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '生成批次失败')
  } finally {
    busy.value = false
  }
}

// 批次一键完成
async function doCompleteBatch(batch: string) {
  if (busy.value) return
  busy.value = true
  try {
    const res = await completeSettleBatch(batch)
    toast.success(`批次 ${batch} 已完成，${res.affected} 条置为已完成`)
    await reload()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '批次完成失败')
  } finally {
    busy.value = false
  }
}

// 银行专用打款导出（C-4）
const exportMenuBatch = ref<string | null>(null)
const exportTmpls = [
  { tmpl: 'mybank', label: '网商银行（支付宝+银行卡）' },
  { tmpl: 'alipay', label: '支付宝批量转账' },
  { tmpl: 'wxpay', label: '微信付款到零钱' },
  { tmpl: 'common', label: '通用明细' },
]
async function doExportBatch(batch: string, tmpl: string) {
  exportMenuBatch.value = null
  try {
    await exportSettleBatch(batch, tmpl)
    toast.success('导出成功')
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '导出失败')
  }
}

// 单条状态变更（0待结算/1已完成/2正在结算）。改结算失败(3)走失败原因弹窗。
async function changeStatus(id: number, status: number) {
  openMenu.value = null
  if (status === 3) return openResult(rowById(id), true)
  if (busy.value) return
  busy.value = true
  try {
    await setSettleStatus(id, status)
    toast.success('状态已更新')
    await reload()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '状态更新失败')
  } finally {
    busy.value = false
  }
}

function rowById(id: number): SettleRecord | null {
  return pageRows.value.find((r) => r.id === id) ?? null
}

// ===== 结算失败原因编辑（J-6，对齐 epay settle_setresult：写 result 字段）=====
// toFail=true 时是「改为结算失败」（同时置状态3+写原因）；否则是「编辑失败原因」（保持状态3，仅改原因文本）。
const resultTarget = ref<SettleRecord | null>(null)
const resultText = ref('')
const resultSaving = ref(false)
function openResult(r: SettleRecord | null, toFail: boolean) {
  openMenu.value = null
  if (!r) return
  void toFail
  resultTarget.value = r
  resultText.value = r.result ?? ''
}
async function saveResult() {
  const r = resultTarget.value
  if (!r || resultSaving.value) return
  resultSaving.value = true
  try {
    // 统一走 setSettleStatus(3, result)：既置结算失败又写失败原因（对齐 epay setResult 落在 result 字段）
    await setSettleStatus(r.id, 3, resultText.value.trim())
    toast.success('结算失败原因已保存')
    resultTarget.value = null
    await reload()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    resultSaving.value = false
  }
}

// 删除并退回余额（二次确认 + 管理员支付密码校验）
const delTarget = ref<SettleRecord | null>(null)
const delPassword = ref('')
function askDelete(r: SettleRecord) {
  openMenu.value = null
  delPassword.value = ''
  delTarget.value = r
}
async function confirmDelete() {
  const r = delTarget.value
  if (!r || busy.value) return
  if (!delPassword.value) {
    toast.error('请输入支付密码')
    return
  }
  busy.value = true
  try {
    await setSettleStatus(r.id, 4, '', delPassword.value)
    toast.success(`已删除记录 #${r.id}，结算金额 ¥${r.money} 退回商户余额`)
    delTarget.value = null
    await reload()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  } finally {
    busy.value = false
  }
}

// 批量修改状态（对选中记录逐条调接口）
async function bulkStatus(status: number) {
  if (busy.value || !selected.value.size) return
  busy.value = true
  const ids = [...selected.value]
  let ok = 0
  try {
    for (const id of ids) {
      try {
        await setSettleStatus(id, status)
        ok++
      } catch {
        /* 单条失败继续，末尾汇总 */
      }
    }
    toast.success(`批量处理完成：${ok}/${ids.length} 条成功`)
    selected.value.clear()
    await reload()
  } finally {
    busy.value = false
  }
}

// 批量删除记录（对齐 epay opslist status=4；逐条删除并退回结算金额，需管理员支付密码）
const bulkDelDialog = ref(false)
const bulkDelPwd = ref('')
function askBulkDelete() {
  if (!selected.value.size) return toast.info('请先勾选记录')
  bulkDelPwd.value = ''
  bulkDelDialog.value = true
}
async function runBulkDelete() {
  if (busy.value) return
  if (!bulkDelPwd.value) return toast.error('请输入支付密码')
  busy.value = true
  const ids = [...selected.value]
  let ok = 0
  try {
    for (const id of ids) {
      try {
        await setSettleStatus(id, 4, '', bulkDelPwd.value)
        ok++
      } catch {
        /* 单条失败继续，末尾汇总 */
      }
    }
    toast.success(`批量删除完成：${ok}/${ids.length} 条成功（结算金额已退回商户余额）`)
    bulkDelDialog.value = false
    selected.value.clear()
    await reload()
  } finally {
    busy.value = false
  }
}

// ===== 高级导出抽屉（服务端全量导出 CSV，按时间范围/商户/方式/状态组合，对齐 Orders.vue）=====
// 导出走后端流式接口，覆盖全部匹配记录（修正旧版仅导出前端已加载的前 100 条）。
const exportOpen = ref(false)
const exporting = ref(false)
const exportForm = ref({
  starttime: '',
  endtime: '',
  uid: '',
  type: 0,
  dstatus: -1,
})
// 预估条数：向后端概况接口按导出筛选取 totalCount（与实际导出同口径）。
const exportCount = ref(0)
async function refreshExportCount() {
  try {
    const s = await fetchSettleStats(exportParams())
    exportCount.value = s.totalCount
  } catch {
    exportCount.value = 0
  }
}
function exportParams(): Omit<SettleListParams, 'page' | 'pageSize'> & {
  starttime?: string
  endtime?: string
} {
  const f = exportForm.value
  const p: Omit<SettleListParams, 'page' | 'pageSize'> & { starttime?: string; endtime?: string } = {}
  if (f.starttime) p.starttime = f.starttime
  if (f.endtime) p.endtime = f.endtime
  if (f.uid.trim()) p.uid = Number(f.uid.trim()) || 0
  if (f.type) p.type = f.type
  if (f.dstatus > -1) p.status = f.dstatus
  return p
}
function openExport() {
  // 带入当前列表筛选作为默认导出条件，减少重复输入
  exportForm.value = {
    starttime: '',
    endtime: '',
    uid: filters.value.uid,
    type: filters.value.type,
    dstatus: filters.value.dstatus,
  }
  exportOpen.value = true
  refreshExportCount()
}
// 导出筛选变化时刷新预估条数
watch(exportForm, refreshExportCount, { deep: true })
async function submitExport() {
  exporting.value = true
  try {
    await exportSettles(exportParams())
    toast.success('已开始导出结算明细')
    exportOpen.value = false
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '导出失败')
  } finally {
    exporting.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 结算批次 -->
    <Panel title="结算管理" subtitle="按批次归集待结算记录，通过转账接口或手动打款完成结算">
      <template #actions>
        <Button size="sm" :disabled="busy" @click="doCreateBatch"><Plus />生成结算批次</Button>
      </template>
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div
          v-for="b in batches"
          :key="b.batch"
          class="group flex flex-col gap-3.5 bg-muted/40 p-4 transition-colors hover:bg-muted/70"
        >
          <!-- 标题行：批次号（标题）+ 状态 -->
          <div class="flex items-center justify-between gap-2">
            <div class="flex min-w-0 items-center gap-1.5">
              <Layers class="size-3.5 shrink-0 text-muted-foreground" />
              <span class="truncate text-[13px] font-semibold tracking-tight tabular-nums">{{ b.batch }}</span>
            </div>
            <Badge :variant="batchStatus[b.status].variant">{{ batchStatus[b.status].text }}</Badge>
          </div>

          <!-- 焦点：批次金额（唯一视觉重心）；笔数/时间降为下方辅助 meta -->
          <div>
            <div class="text-xl font-semibold tracking-tight tabular-nums">
              <span class="mr-0.5 text-sm font-normal text-muted-foreground">¥</span>{{ formatMoney(+b.allmoney) }}
            </div>
            <div class="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground">
              <span class="tabular-nums">{{ b.count }} 笔</span>
              <span class="text-border">·</span>
              <Clock class="size-3" />
              <span class="tabular-nums">{{ b.time }}</span>
            </div>
          </div>

          <!-- 操作 -->
          <div class="mt-auto flex flex-wrap gap-1.5 pt-0.5">
            <Button variant="outline" size="sm" @click="viewBatch(b.batch)">
              结算列表
            </Button>
            <Button
              v-if="b.status !== 1"
              variant="outline"
              size="sm"
              :disabled="busy"
              @click="doCompleteBatch(b.batch)"
            >
              <CheckCircle2 />标记完成
            </Button>
            <div class="relative inline-block">
              <Button variant="outline" size="sm" @click.stop="exportMenuBatch = exportMenuBatch === b.batch ? null : b.batch">
                <Download />导出打款
              </Button>
              <div
                v-if="exportMenuBatch === b.batch"
                class="menu-panel absolute left-0 top-full z-20 mt-1.5 w-52"
                @click.stop
              >
                <button
                  v-for="t in exportTmpls"
                  :key="t.tmpl"
                  class="menu-item"
                  @click="doExportBatch(b.batch, t.tmpl)"
                >
                  <Download class="size-4 shrink-0 opacity-70" />
                  <span class="flex-1">{{ t.label }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        结算标准：可结算余额大于 <b class="text-foreground">100</b> 元，或商户主动申请。
        生成批次后所有「待结算」记录转为「正在结算」，转账成功后自动置为「已完成」；手动打款需手动改为完成。
      </p>
    </Panel>

    <!-- 明细汇总 -->
    <Panel title="结算概况" subtitle="按当前筛选条件">
      <div class="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3 lg:grid-cols-6">
        <div>
          <div class="text-[13px] text-muted-foreground">结算总金额</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.totalMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">实际到账</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-primary"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.realMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已完成金额</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-success"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.doneMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">记录总数</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums">{{ stats.totalCount }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">正在结算 / 失败</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums">
            <span class="text-warning">{{ stats.processingCount }}</span>
            <span class="text-sm text-muted-foreground">/</span>
            <span class="text-destructive">{{ stats.failCount }}</span>
          </div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已完成笔数</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-success">{{ stats.doneCount }}</div>
        </div>
      </div>
    </Panel>

    <!-- 筛选 -->
    <Panel title="结算明细" :subtitle="`共 ${total} 条记录`">
      <template #actions>
        <Button variant="outline" size="sm" @click="openExport"><Download />导出列表</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="whitespace-nowrap text-sm text-muted-foreground">账号 / 姓名</label>
          <input v-model="filters.value" placeholder="结算账号或姓名" class="field-input w-48" />
        </div>
        <div class="filter-item">
          <label class="whitespace-nowrap text-sm text-muted-foreground">商户号</label>
          <input v-model="filters.uid" placeholder="请输入商户号" class="field-input w-36" />
        </div>
        <div class="filter-item">
          <label class="whitespace-nowrap text-sm text-muted-foreground">批次</label>
          <Select v-model="filters.batch" :options="batchOptions" class="w-44" @change="applySearch" />
        </div>
        <div class="filter-item">
          <label class="whitespace-nowrap text-sm text-muted-foreground">结算方式</label>
          <Select v-model="filters.type" :options="typeOptions" class="w-32" @change="applySearch" />
        </div>
        <div class="filter-item">
          <label class="whitespace-nowrap text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.dstatus" :options="statusOptions" class="w-28" @change="applySearch" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="applySearch"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 明细列表 -->
    <Panel title="结算记录" :subtitle="selected.size ? `已选 ${selected.size} 条` : `${total} 条`">
      <template v-if="selected.size" #actions>
        <span class="text-sm text-muted-foreground">批量修改为：</span>
        <Button variant="outline" size="sm" :disabled="busy" @click="bulkStatus(1)"><CheckCircle2 />已完成</Button>
        <Button variant="outline" size="sm" :disabled="busy" @click="bulkStatus(2)"><Clock />正在结算</Button>
        <Button variant="outline" size="sm" :disabled="busy" @click="bulkStatus(0)"><RotateCcw />待结算</Button>
        <Button variant="outline" size="sm" class="text-destructive hover:text-destructive" :disabled="busy" @click="askBulkDelete"><Trash2 />删除并退回</Button>
      </template>
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[4%] col-center">
                <input type="checkbox" :checked="pageAllChecked" @change="toggleAll" />
              </th>
              <th class="w-[8%]">ID</th>
              <th class="w-[13%]">商户</th>
              <th class="w-[14%]">结算方式</th>
              <th class="w-[18%]">结算账号 / 姓名</th>
              <th class="w-[14%]">结算金额 / 到账</th>
              <th class="w-[15%]">创建 / 完成时间</th>
              <th class="col-center w-[8%]">状态</th>
              <th class="col-center w-[6%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in pageRows" :key="r.id">
              <td class="col-center">
                <input type="checkbox" :checked="selected.has(r.id)" @change="toggleOne(r.id)" />
              </td>
              <td class="font-medium tabular-nums">{{ r.id }}</td>
              <td>
                <div class="truncate">{{ r.merchant }}</div>
                <div class="text-xs dim tabular-nums">{{ r.uid }}</div>
              </td>
              <td>
                <div class="flex items-center gap-1.5">
                  <span class="grid size-5 place-items-center rounded bg-primary/10 text-[11px] font-medium text-primary">
                    {{ typeInitial(r.type) }}
                  </span>
                  <span>{{ settleTypes[r.type].showname }}</span>
                  <span v-if="!r.auto" class="text-xs dim">[手动]</span>
                </div>
              </td>
              <td>
                <div class="truncate tabular-nums">{{ r.account }}</div>
                <div class="text-xs dim">{{ r.username }}</div>
              </td>
              <td>
                <div class="tabular-nums"><span class="dim text-xs">¥</span><b>{{ r.money }}</b></div>
                <div class="text-xs dim tabular-nums">到账 ¥{{ r.realmoney }}</div>
              </td>
              <td>
                <div class="text-xs">{{ r.addtime }}</div>
                <div class="text-xs dim">{{ r.endtime ?? '—' }}</div>
              </td>
              <td class="col-center">
                <Badge :variant="settleStatus[r.status].variant">{{ settleStatus[r.status].text }}</Badge>
                <div v-if="r.status === 3" class="mt-1 truncate text-xs text-destructive" :title="r.result">
                  {{ r.result }}
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(r.id, $event)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === r.id"
                    class="menu-panel absolute right-0 z-20 w-36"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="changeStatus(r.id, 1)">
                      <CheckCircle2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">改为已完成</span>
                    </button>
                    <button class="menu-item" @click="changeStatus(r.id, 2)">
                      <Clock class="size-4 shrink-0 opacity-70" /><span class="flex-1">改为正在结算</span>
                    </button>
                    <button class="menu-item" @click="changeStatus(r.id, 0)">
                      <RotateCcw class="size-4 shrink-0 opacity-70" /><span class="flex-1">改为待结算</span>
                    </button>
                    <button class="menu-item" @click="changeStatus(r.id, 3)">
                      <Send class="size-4 shrink-0 opacity-70" /><span class="flex-1">改为结算失败</span>
                    </button>
                    <button v-if="r.status === 3" class="menu-item" @click="openResult(r, false)">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑失败原因</span>
                    </button>
                    <div class="menu-sep" />
                    <button class="menu-item menu-item-danger" @click="askDelete(r)">
                      <Undo2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除并退回</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="9" class="py-10 text-center dim">没有符合条件的结算记录</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 删除并退回确认 -->
    <Modal :model-value="!!delTarget" title="删除结算记录并退回余额" @update:model-value="(v) => { if (!v) delTarget = null }">
      <p class="text-sm text-muted-foreground">
        确定删除记录
        <b class="text-foreground">#{{ delTarget?.id }}</b>
        （{{ delTarget?.merchant }}）吗？删除后结算金额
        <b class="text-foreground tabular-nums">¥{{ delTarget?.money }}</b>
        将退回该商户余额，此操作不可恢复。
      </p>
      <div class="mt-3 row-field">
        <label class="lbl">支付密码</label>
        <input v-model="delPassword" type="password" placeholder="请输入管理员支付密码" class="field-input flex-1" @keyup.enter="confirmDelete" />
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="delTarget = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="busy" @click="confirmDelete">删除并退回</Button>
      </template>
    </Modal>

    <!-- 批量删除并退回（对齐 epay opslist 批量删除，逐条退回结算金额）-->
    <Modal :model-value="bulkDelDialog" title="批量删除结算记录" @update:model-value="(v) => { if (!v) bulkDelDialog = false }">
      <p class="text-sm text-muted-foreground">
        确定删除已选的 <b class="text-foreground">{{ selected.size }}</b> 条结算记录吗？
        删除后每条结算金额将退回对应商户余额，此操作不可恢复。
      </p>
      <div class="mt-3 row-field">
        <label class="lbl">支付密码</label>
        <input v-model="bulkDelPwd" type="password" placeholder="请输入管理员支付密码" class="field-input flex-1" @keyup.enter="runBulkDelete" />
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="bulkDelDialog = false">取消</Button>
        <Button variant="destructive" size="sm" :disabled="busy" @click="runBulkDelete">批量删除并退回</Button>
      </template>
    </Modal>

    <!-- 结算失败原因编辑（J-6，对齐 epay settle_setresult）-->
    <Modal :model-value="!!resultTarget" title="结算失败原因" @update:model-value="(v) => { if (!v) resultTarget = null }">
      <p class="mb-3 text-sm text-muted-foreground">
        记录 <b class="text-foreground">#{{ resultTarget?.id }}</b>（{{ resultTarget?.merchant }}）。
        保存后该记录将标记为结算失败并记录失败原因。
      </p>
      <div>
        <label class="mb-1 block text-sm text-muted-foreground">失败原因</label>
        <textarea v-model="resultText" rows="3" placeholder="请输入结算失败原因（如：银行卡号有误、账户异常等）" class="field-input w-full resize-none"></textarea>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="resultTarget = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="resultSaving" @click="saveResult">保存</Button>
      </template>
    </Modal>

    <!-- 高级导出抽屉：按时间范围/商户/方式/状态组合导出 CSV -->
    <Drawer v-model="exportOpen" title="导出结算明细" subtitle="按条件批量导出结算记录为 CSV 文件" width="max-w-md">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">创建时间<span class="text-destructive">*</span></label>
          <DateRange v-model:start="exportForm.starttime" v-model:end="exportForm.endtime" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">商户号</label>
          <input v-model="exportForm.uid" placeholder="留空为全部商户" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">结算方式</label>
          <Select v-model="exportForm.type" :options="typeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">状态</label>
          <Select v-model="exportForm.dstatus" :options="statusOptions" class="flex-1" />
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          时间范围留空则导出全部。按当前条件预计导出
          <b class="text-foreground tabular-nums">{{ exportCount }}</b> 条结算记录。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="exportOpen = false">取消</Button>
        <Button size="sm" :disabled="exporting || exportCount === 0" @click="submitExport">
          <Download />导出 CSV
        </Button>
      </template>
    </Drawer>
  </div>
</template>
