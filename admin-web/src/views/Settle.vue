<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
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
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination, Modal, Drawer, DateRange } from '@/components/ui'
import {
  settleTypes,
  settleStatus,
  batchStatus,
  calcSettleStats,
  type SettleRecord,
  type SettleBatch,
} from '@/lib/mock/settle'
import {
  fetchSettles,
  fetchSettleBatches,
  createSettleBatch,
  completeSettleBatch,
  setSettleStatus,
} from '@/lib/api/settle'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { formatMoney } from '@/lib/utils'

const toast = useToast()

// ===== 真接口数据（一次拉取，客户端筛选/分页，对齐 Channels.vue 模式）=====
const batches = ref<SettleBatch[]>([])
const settleRecords = ref<SettleRecord[]>([])

async function loadBatches() {
  try {
    const res = await fetchSettleBatches({ page: 1, pageSize: 100 })
    batches.value = res.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '批次加载失败')
    batches.value = []
  }
}
async function loadRecords() {
  try {
    const res = await fetchSettles({ page: 1, pageSize: 100 })
    settleRecords.value = res.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '结算明细加载失败')
    settleRecords.value = []
  }
}
async function reload() {
  await Promise.all([loadBatches(), loadRecords()])
}
onMounted(reload)

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

const filtered = computed(() => {
  return settleRecords.value.filter((r) => {
    if (filters.value.uid && String(r.uid) !== filters.value.uid.trim()) return false
    if (filters.value.batch && r.batch !== filters.value.batch) return false
    if (filters.value.type && r.type !== filters.value.type) return false
    if (filters.value.dstatus > -1 && r.status !== filters.value.dstatus) return false
    if (filters.value.value.trim()) {
      const v = filters.value.value.trim()
      if (!`${r.account}${r.username}`.includes(v)) return false
    }
    return true
  })
})

function resetFilters() {
  filters.value = { value: '', uid: '', batch: '', type: 0, dstatus: -1 }
  page.value = 1
  selected.value.clear()
}

// ===== 分页 =====
const page = ref(1)
const pageSize = 15
const total = computed(() => filtered.value.length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))
const pageRows = computed(() =>
  filtered.value.slice((safePage.value - 1) * pageSize, safePage.value * pageSize),
)
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
}

// 搜索 / 切换筛选：回到第一页并清空跨筛选的选中，避免误批量操作不可见记录
function applySearch() {
  page.value = 1
  selected.value.clear()
}
function viewBatch(batch: string) {
  filters.value.batch = batch
  applySearch()
}

// ===== 明细汇总 =====
const stats = computed(() => calcSettleStats(filtered.value))

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

// 筛选即时变化时回到第1页并清空选中，避免批量操作命中不可见行
watch(filters, () => {
  page.value = 1
  selected.value.clear()
}, { deep: true })

// ===== 行操作菜单 =====
const openMenu = ref<number | null>(null)
function toggleMenu(id: number) {
  openMenu.value = openMenu.value === id ? null : id
}
function closeMenu() {
  openMenu.value = null
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

// 单条状态变更（0待结算/1已完成/2正在结算/3失败）
async function changeStatus(id: number, status: number) {
  openMenu.value = null
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

// 删除并退回余额（二次确认）
const delTarget = ref<SettleRecord | null>(null)
function askDelete(r: SettleRecord) {
  openMenu.value = null
  delTarget.value = r
}
async function confirmDelete() {
  const r = delTarget.value
  if (!r || busy.value) return
  busy.value = true
  try {
    await setSettleStatus(r.id, 4)
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

// ===== 高级导出抽屉（按时间范围/商户/方式/状态组合导出 CSV，对齐 Orders.vue）=====
const exportOpen = ref(false)
const exportForm = ref({
  starttime: '',
  endtime: '',
  uid: '',
  type: 0,
  dstatus: -1,
})
// 按导出条件过滤（预估条数与实际导出共用，避免"预估≠导出"）
function filterForExport(): SettleRecord[] {
  const f = exportForm.value
  return settleRecords.value.filter((r) => {
    // 时间范围：按 addtime 日期部分闭区间（含起止当天）
    const day = (r.addtime || '').slice(0, 10)
    if (f.starttime && day < f.starttime) return false
    if (f.endtime && day > f.endtime) return false
    if (f.uid && String(r.uid) !== f.uid.trim()) return false
    if (f.type && r.type !== f.type) return false
    if (f.dstatus > -1 && r.status !== f.dstatus) return false
    return true
  })
}
const exportCount = computed(() => filterForExport().length)
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
}
function csvCell(v: string | number | null | undefined): string {
  const s = v == null ? '' : String(v)
  if (/[",\n]/.test(s)) return '"' + s.replace(/"/g, '""') + '"'
  return s
}
function submitExport() {
  const rows = filterForExport()
  if (rows.length === 0) {
    toast.info('当前条件下没有可导出的结算记录')
    return
  }
  const headers = [
    'ID', '批次号', '商户号', '商户', '结算方式', '是否自动',
    '结算账号', '结算姓名', '结算金额', '实际到账',
    '创建时间', '完成时间', '状态', '失败原因',
  ]
  const lines = rows.map((r) =>
    [
      r.id, r.batch, r.uid, r.merchant,
      settleTypes[r.type]?.showname ?? r.type, r.auto ? '自动' : '手动',
      r.account, r.username, r.money, r.realmoney,
      r.addtime, r.endtime ?? '', settleStatus[r.status]?.text ?? r.status, r.result,
    ].map(csvCell).join(','),
  )
  // 加 BOM，保证 Excel 打开中文不乱码
  const csv = '﻿' + [headers.join(','), ...lines].join('\r\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  const range = `${exportForm.value.starttime}_${exportForm.value.endtime}`
  a.href = url
  a.download = `结算明细_${range}.csv`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  toast.success(`已导出 ${rows.length} 条结算记录`)
  exportOpen.value = false
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
          <Select v-model="filters.batch" :options="batchOptions" class="w-44" />
        </div>
        <div class="filter-item">
          <label class="whitespace-nowrap text-sm text-muted-foreground">结算方式</label>
          <Select v-model="filters.type" :options="typeOptions" class="w-32" />
        </div>
        <div class="filter-item">
          <label class="whitespace-nowrap text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.dstatus" :options="statusOptions" class="w-28" />
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
      </template>
      <div class="overflow-x-auto">
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
            <tr v-for="(r, si) in pageRows" :key="r.id">
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
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(r.id)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === r.id"
                    class="menu-panel absolute right-0 z-20 w-36"
                    :class="si >= pageRows.length - 3 && pageRows.length > 3
                      ? 'bottom-full mb-1.5'
                      : 'top-full mt-1.5'"
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
      <template #footer>
        <Button variant="outline" size="sm" @click="delTarget = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="busy" @click="confirmDelete">删除并退回</Button>
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
          创建时间范围为必填。按当前条件预计导出
          <b class="text-foreground tabular-nums">{{ exportCount }}</b> 条结算记录。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="exportOpen = false">取消</Button>
        <Button size="sm" :disabled="!exportForm.starttime || !exportForm.endtime" @click="submitExport">
          <Download />导出 CSV
        </Button>
      </template>
    </Drawer>
  </div>
</template>
