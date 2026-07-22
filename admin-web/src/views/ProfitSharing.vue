<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import {
  Search, RotateCcw, Download, BarChart3, MoreHorizontal,
  Send, Undo2, RefreshCw, XCircle,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, DateRange, Pagination, Modal } from '@/components/ui'
import {
  fetchPsOrders, fetchPsStats, operatePsOrder,
  type PsOrder, type PsStats,
} from '@/lib/api/profitsharing'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { shouldDropUp } from '@/composables/useRowMenu'
import { formatMoney } from '@/lib/utils'

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

// 状态可执行操作（对齐 epay ps_order + 后端能力）
function psActions(status: number): { key: string; label: string }[] {
  if (status === 0) return [{ key: 'submit', label: '提交分账' }, { key: 'cancel', label: '取消' }]
  if (status === 1) return [{ key: 'query', label: '查询结果' }]
  if (status === 2) return [{ key: 'return', label: '分账回退' }]
  if (status === 3) return [{ key: 'submit', label: '重试' }, { key: 'cancel', label: '取消' }]
  return []
}
const actionIcons: Record<string, any> = {
  submit: Send, query: RefreshCw, return: Undo2, cancel: XCircle,
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
    default:
      return ''
  }
})

function onAction(r: PsOrder, key: string, label: string) {
  openMenu.value = null
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
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="分账记录" :subtitle="`共 ${total} 笔分账`">
      <template #actions>
        <Button variant="outline" size="sm" @click="toggleStats"><BarChart3 />统计</Button>
        <Button variant="outline" size="sm"><Download />导出列表</Button>
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
    <Panel title="分账列表" :subtitle="`${total} 条`">
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[20%]">系统订单号</th>
              <th class="w-[17%]">分账规则 / 接收方</th>
              <th class="w-[15%]">支付通道</th>
              <th class="w-[11%]">分账金额</th>
              <th class="w-[15%]">时间</th>
              <th class="col-center w-[10%]">状态</th>
              <th class="col-center w-[12%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in rows" :key="o.id">
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
                      :class="a.key === 'cancel' && 'menu-item-danger'"
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
              <td colspan="7" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="7" class="py-10 text-center dim">没有符合条件的分账记录</td>
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
  </div>
</template>
