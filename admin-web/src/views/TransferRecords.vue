<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { RouterLink } from 'vue-router'
import {
  Search,
  RotateCcw,
  Plus,
  MoreHorizontal,
  CheckCircle2,
  XCircle,
  Undo2,
  Copy,
  Trash2,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination, Modal } from '@/components/ui'
import {
  fetchTransfers,
  fetchTransferStats,
  setTransferStatus,
  refundTransfer,
  deleteTransfer,
  type TransferRecord,
  type TransferStats,
} from '@/lib/api/transfer'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { shouldDropUp } from '@/composables/useRowMenu'
import { formatMoney } from '@/lib/utils'

const toast = useToast()

const transferTypeText: Record<string, string> = {
  alipay: '支付宝', wxpay: '微信', qqpay: 'QQ钱包', bank: '银行卡',
}
const transferStatus: Record<number, { text: string; variant: 'warning' | 'success' | 'destructive' }> = {
  0: { text: '正在处理', variant: 'warning' },
  1: { text: '转账成功', variant: 'success' },
  2: { text: '转账失败', variant: 'destructive' },
}
const typeOptions = [
  { value: '', label: '所有付款方式' },
  { value: 'alipay', label: '支付宝' },
  { value: 'wxpay', label: '微信' },
  { value: 'qqpay', label: 'QQ钱包' },
  { value: 'bank', label: '银行卡' },
]
const statusOptions = [
  { value: -1, label: '全部状态' },
  { value: 0, label: '正在处理' },
  { value: 1, label: '转账成功' },
  { value: 2, label: '转账失败' },
]

// ===== 筛选 =====
const filters = reactive({ value: '', uid: '', type: '', dstatus: -1 })

// ===== 分页 + 数据 =====
const page = ref(1)
const pageSize = 15
const total = ref(0)
const rows = ref<TransferRecord[]>([])
const loading = ref(false)

function buildParams() {
  const uidNum = Number(filters.uid.trim())
  return {
    page: page.value,
    pageSize,
    keyword: filters.value.trim() || undefined,
    uid: filters.uid.trim() && !Number.isNaN(uidNum) ? uidNum : undefined,
    type: filters.type || undefined,
    status: filters.dstatus > -1 ? filters.dstatus : undefined,
  }
}

async function load() {
  loading.value = true
  try {
    const res = await fetchTransfers(buildParams())
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载付款记录失败')
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    const { page: _p, pageSize: _ps, ...rest } = buildParams()
    void _p
    void _ps
    stats.value = await fetchTransferStats(rest)
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
  filters.value = ''
  filters.uid = ''
  filters.type = ''
  filters.dstatus = -1
  applySearch()
}
function go(p: number) {
  page.value = p
  load()
}
onMounted(reload)

// ===== 统计 =====
const stats = ref<TransferStats | null>(null)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// ===== 行操作菜单 =====
const openMenu = ref<string | null>(null)
const dropUp = ref(false)
function toggleMenu(biz: string, ev?: MouseEvent) {
  if (openMenu.value === biz) { openMenu.value = null; return }
  openMenu.value = biz
  dropUp.value = shouldDropUp(ev)
}
function closeMenu() {
  openMenu.value = null
}
onMounted(() => window.addEventListener('click', closeMenu))
onUnmounted(() => window.removeEventListener('click', closeMenu))

// ===== 多选 + 批量操作（对齐 epay transfer.php operation：改成功(1)/改失败(2)/删除(3)，裸改状态不退款）=====
const selected = ref<Set<string>>(new Set())
const pageAllChecked = computed(
  () => rows.value.length > 0 && rows.value.every((r) => selected.value.has(r.biz_no)),
)
function toggleAll() {
  if (pageAllChecked.value) rows.value.forEach((r) => selected.value.delete(r.biz_no))
  else rows.value.forEach((r) => selected.value.add(r.biz_no))
  selected.value = new Set(selected.value)
}
function toggleOne(biz: string) {
  if (selected.value.has(biz)) selected.value.delete(biz)
  else selected.value.add(biz)
  selected.value = new Set(selected.value)
}
const bulkAction = ref<'success' | 'fail' | 'delete'>('success')
const bulkOptions = [
  { value: 'success', label: '改为成功' },
  { value: 'fail', label: '改为失败' },
  { value: 'delete', label: '删除' },
]
const bulkLabels: Record<string, string> = { success: '改为成功', fail: '改为失败', delete: '删除' }
const bulkConfirm = ref(false)
function askBulk() {
  if (!selected.value.size) return toast.info('请先勾选付款记录')
  bulkConfirm.value = true
}
async function runBulk() {
  if (busy.value) return
  busy.value = true
  const bizNos = [...selected.value]
  let ok = 0
  try {
    for (const biz of bizNos) {
      try {
        if (bulkAction.value === 'success') await setTransferStatus(biz, 1)
        else if (bulkAction.value === 'fail') await setTransferStatus(biz, 2, '后台批量置为失败')
        else await deleteTransfer(biz)
        ok++
      } catch {
        /* 单条失败继续 */
      }
    }
    toast.success(`${bulkLabels[bulkAction.value]}完成：${ok}/${bizNos.length} 条成功`)
    bulkConfirm.value = false
    selected.value = new Set()
    await reload()
  } finally {
    busy.value = false
  }
}

/** 状态可执行操作（对齐 epay transfer.php 操作列 + 后端能力） */
function rowActions(r: TransferRecord): string[] {
  if (r.status === 1) return ['改为失败', '复制交易号', '删除']
  if (r.status === 2) return ['改为成功', '复制交易号', '删除']
  // 处理中：商户发起可退回（退款），管理员发起可删除
  return r.uid > 0 ? ['退回', '复制交易号', '删除'] : ['查询状态', '复制交易号', '删除']
}
const actionIcons: Record<string, any> = {
  改为成功: CheckCircle2,
  改为失败: XCircle,
  退回: Undo2,
  查询状态: Search,
  复制交易号: Copy,
  删除: Trash2,
}

// ===== 写操作（调真接口）=====
const busy = ref(false)
// 二次确认弹窗（资金相关操作：退回 / 改状态 / 删除）
const confirmOpen = ref(false)
const confirmRow = ref<TransferRecord | null>(null)
const confirmAction = ref('')
const confirmText = computed(() => {
  const r = confirmRow.value
  if (!r) return ''
  const money = `¥${formatMoney(r.costmoney)}`
  switch (confirmAction.value) {
    case '退回':
      return `确认退回该笔代付？将把 ${money} 退回商户 ${r.uid} 余额，且状态置为失败。此操作不可撤销。`
    case '改为成功':
      return '确认将该笔代付标记为成功？（仅改状态，不产生资金变动）'
    case '改为失败':
      return '确认将该笔代付标记为失败？（仅改状态，不退款）'
    case '删除':
      return '确认删除该付款记录？删除不退款，请谨慎操作。'
    default:
      return ''
  }
})

function onAction(r: TransferRecord, action: string) {
  openMenu.value = null
  if (action === '复制交易号') {
    navigator.clipboard?.writeText(r.biz_no)
    toast.success('已复制交易号')
    return
  }
  if (action === '查询状态') {
    toast.info('渠道查询待真实渠道凭证接入')
    return
  }
  // 需二次确认的资金/状态操作
  confirmRow.value = r
  confirmAction.value = action
  confirmOpen.value = true
}

async function doConfirm() {
  const r = confirmRow.value
  if (!r || busy.value) return
  busy.value = true
  try {
    if (confirmAction.value === '退回') {
      await refundTransfer(r.biz_no)
      toast.success('已退回，款项已退回商户余额')
    } else if (confirmAction.value === '改为成功') {
      await setTransferStatus(r.biz_no, 1)
      toast.success('已标记为成功')
    } else if (confirmAction.value === '改为失败') {
      await setTransferStatus(r.biz_no, 2, '后台手动置为失败')
      toast.success('已标记为失败')
    } else if (confirmAction.value === '删除') {
      await deleteTransfer(r.biz_no)
      toast.success('已删除记录')
    }
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
    <Panel title="付款记录" :subtitle="`共 ${total} 笔付款`">
      <template #actions>
        <RouterLink to="/admin/transfer">
          <Button size="sm"><Plus />新增付款</Button>
        </RouterLink>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">付款搜索</label>
          <input v-model="filters.value" placeholder="交易号 / 收款账号 / 姓名" class="field-input w-52" @keyup.enter="applySearch" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">商户号</label>
          <input v-model="filters.uid" placeholder="请输入商户号" class="field-input w-32" @keyup.enter="applySearch" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">付款方式</label>
          <Select v-model="filters.type" :options="typeOptions" class="w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.dstatus" :options="statusOptions" class="w-28" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="applySearch"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 概况 -->
    <Panel v-if="stats" title="付款概况" subtitle="按当前筛选条件">
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">付款总额</div>
          <div class="mt-1 text-xl font-normal tabular-nums"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.totalMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">成功金额</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.successMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">成功笔数</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.successCount }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">处理中 / 失败</div>
          <div class="mt-1 text-xl font-normal tabular-nums">
            <span class="text-warning">{{ stats.processingCount }}</span>
            <span class="text-sm text-muted-foreground">/</span>
            <span class="text-destructive">{{ stats.failCount }}</span>
          </div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="付款记录列表" :subtitle="selected.size ? `已选 ${selected.size} 条` : `${total} 条`">
      <template v-if="selected.size" #actions>
        <span class="text-sm text-muted-foreground">批量操作：</span>
        <Select v-model="bulkAction" :options="bulkOptions" class="w-28" />
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
              <th class="w-[16%]">交易号 / 第三方号</th>
              <th class="w-[8%]">商户号</th>
              <th class="w-[14%]">付款方式 / 备注</th>
              <th class="w-[15%]">收款账号 / 姓名</th>
              <th class="w-[13%]">付款 / 花费</th>
              <th class="w-[14%]">提交 / 付款时间</th>
              <th class="col-center w-[8%]">状态</th>
              <th class="col-center w-[8%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in rows" :key="r.biz_no">
              <td class="col-center">
                <input type="checkbox" :checked="selected.has(r.biz_no)" class="align-middle" @change="toggleOne(r.biz_no)" />
              </td>
              <td>
                <div class="truncate font-medium tabular-nums">{{ r.biz_no }}</div>
                <div class="truncate text-xs dim">{{ r.pay_order_no || '—' }}</div>
              </td>
              <td>
                <span v-if="r.uid > 0" class="tabular-nums text-primary">{{ r.uid }}</span>
                <span v-else class="dim">管理员</span>
              </td>
              <td>
                <div>{{ transferTypeText[r.type] }} <span class="dim">({{ r.channel }})</span></div>
                <div v-if="r.desc" class="truncate text-xs dim">{{ r.desc }}</div>
              </td>
              <td>
                <div class="truncate tabular-nums">{{ r.account }}</div>
                <div class="text-xs dim">{{ r.username || '—' }}</div>
              </td>
              <td>
                <div class="tabular-nums"><span class="dim text-xs">¥</span><b>{{ formatMoney(r.money) }}</b></div>
                <div class="text-xs dim tabular-nums">花费 ¥{{ formatMoney(r.costmoney) }}</div>
              </td>
              <td>
                <div class="text-xs">{{ r.addtime }}</div>
                <div class="text-xs dim">{{ r.paytime ?? '—' }}</div>
              </td>
              <td class="col-center">
                <Badge :variant="transferStatus[r.status].variant">{{ transferStatus[r.status].text }}</Badge>
                <div v-if="r.status === 2 && r.result" class="mt-1 truncate text-xs text-destructive" :title="r.result">
                  {{ r.result }}
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(r.biz_no, $event)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === r.biz_no"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button
                      v-for="(a, ai) in rowActions(r)"
                      :key="ai"
                      class="menu-item"
                      :class="a === '删除' && 'menu-item-danger'"
                      @click="onAction(r, a)"
                    >
                      <component :is="actionIcons[a]" class="size-4 shrink-0 opacity-70" />
                      <span class="flex-1">{{ a }}</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="9" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="9" class="py-10 text-center dim">没有符合条件的付款记录</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 二次确认弹窗 -->
    <Modal v-model="confirmOpen" :title="`${confirmAction}确认`" width="max-w-md">
      <p class="text-sm text-muted-foreground">{{ confirmText }}</p>
      <template #footer>
        <Button variant="outline" size="sm" @click="confirmOpen = false">取消</Button>
        <Button size="sm" :disabled="busy" @click="doConfirm">确认{{ confirmAction }}</Button>
      </template>
    </Modal>

    <!-- 批量操作确认（对齐 epay transfer.php operation：裸改状态不退款）-->
    <Modal :model-value="bulkConfirm" title="批量操作确认" @update:model-value="(v) => { if (!v) bulkConfirm = false }">
      <p class="text-sm text-muted-foreground">
        将对已选的 <b class="text-foreground">{{ selected.size }}</b> 条付款记录执行
        「<b class="text-foreground">{{ bulkOptions.find((o) => o.value === bulkAction)?.label }}</b>」。
        <span v-if="bulkAction === 'delete'" class="text-destructive">删除不退款且不可恢复。</span>
        <span v-else>仅修改状态，不产生资金变动。</span>
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="bulkConfirm = false">取消</Button>
        <Button :variant="bulkAction === 'delete' ? 'destructive' : 'default'" size="sm" :disabled="busy" @click="runBulk">执行</Button>
      </template>
    </Modal>
  </div>
</template>
