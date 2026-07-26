<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search, RotateCcw, BarChart3, MoreHorizontal, Undo2, Bell, ListTree } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, DateRange, Pagination, Modal } from '@/components/ui'
import {
  orderStatus,
  payTypes,
  searchColumns,
  type Order,
} from '@/lib/mock/merchant/orders'
import {
  fetchMerchantOrders,
  fetchMerchantOrderStats,
  refundOrder,
  renotifyOrder,
  type MerchantOrderParams,
  type MerchantOrderStats,
} from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { shouldDropUp } from '@/composables/useRowMenu'
import { formatMoney, exportCsv } from '@/lib/utils'
import { Download } from 'lucide-vue-next'

const router = useRouter()
const toast = useToast()

const columnOptions = searchColumns.map((c) => ({ value: c.value, label: c.label }))
const typeOptions = [{ value: 0, label: '全部方式' }, ...payTypes.map((t) => ({ value: t.id, label: t.showname }))]
const statusOptions = [
  { value: -1, label: '全部状态' },
  ...Object.entries(orderStatus).map(([k, s]) => ({ value: Number(k), label: s.text })),
]

// ===== 筛选（提交后才生效，服务端筛选）=====
const filters = ref({ column: 'trade_no', value: '', type: 0, starttime: '', endtime: '', dstatus: -1 })

// ===== 分页（服务端）=====
const page = ref(1)
const pageSize = 15
const total = ref(0)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const rows = ref<Order[]>([])
const loading = ref(false)

// 把当前筛选转成后端 OrderQuery 参数（对齐 epay orderList）。
function buildParams(extra: Partial<MerchantOrderParams> = {}): MerchantOrderParams {
  const f = filters.value
  const p: MerchantOrderParams = {
    page: page.value,
    pageSize,
    type: f.type || undefined,
    starttime: f.starttime || undefined,
    endtime: f.endtime || undefined,
    ...extra,
  }
  if (f.dstatus > -1) p.status = f.dstatus
  if (f.value.trim()) { p.column = f.column; p.keyword = f.value.trim() }
  return p
}

async function loadOrders() {
  loading.value = true
  try {
    const res = await fetchMerchantOrders(buildParams())
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '订单加载失败')
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  loadOrders()
  if (showStats.value) loadStats()
}

function resetFilters() {
  filters.value = { column: 'trade_no', value: '', type: 0, starttime: '', endtime: '', dstatus: -1 }
  page.value = 1
  loadOrders()
  if (showStats.value) loadStats()
}

function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
  loadOrders()
}

// ===== 统计（服务端聚合，含 platformProfit）=====
const showStats = ref(false)
const stats = ref<MerchantOrderStats | null>(null)
const statsLoading = ref(false)
async function loadStats() {
  statsLoading.value = true
  try {
    // 统计不受分页影响，只带筛选条件
    const { page: _p, pageSize: _ps, ...rest } = buildParams()
    void _p; void _ps
    stats.value = await fetchMerchantOrderStats(rest)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '统计失败')
    stats.value = null
  } finally {
    statsLoading.value = false
  }
}
function toggleStats() {
  showStats.value = !showStats.value
  if (showStats.value && !stats.value) loadStats()
}

// ===== 导出（按当前筛选条件从后端拉全量再生成 CSV，对齐 epay order.php 导出）=====
const exporting = ref(false)
async function exportOrders() {
  if (exporting.value) return
  exporting.value = true
  try {
    const res = await fetchMerchantOrders(buildParams({ page: 1, pageSize: 10000 }))
    const list = res.list
    if (!list.length) { toast.error('没有可导出的订单'); return }
    const headers = ['系统订单号', '商户订单号', '接口订单号', '商品名称', '商品金额', '实付金额', '已退款', '支付方式', '支付账号', '支付IP', '创建时间', '完成时间', '状态']
    const data = list.map((o) => [
      o.trade_no, o.out_trade_no, o.api_trade_no, o.name, o.money, o.realmoney ?? '', o.refundmoney,
      o.typeshowname, o.account, o.ip, o.addtime, o.endtime ?? '',
      (orderStatus as Record<number, { text: string }>)[o.status]?.text ?? o.status,
    ])
    exportCsv(`订单记录_${new Date().toISOString().slice(0, 10)}`, headers, data)
    toast.success(`已导出 ${list.length} 条订单`)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '导出失败')
  } finally {
    exporting.value = false
  }
}

// ===== 行操作菜单 =====
const openMenu = ref<string | null>(null)
const dropUp = ref(false)
function toggleMenu(id: string, ev?: MouseEvent) {
  if (openMenu.value === id) { openMenu.value = null; return }
  openMenu.value = id
  dropUp.value = shouldDropUp(ev)
}
function closeMenu() {
  openMenu.value = null
}
onMounted(() => {
  loadOrders()
  window.addEventListener('click', closeMenu)
})
onUnmounted(() => window.removeEventListener('click', closeMenu))

// 商户视角操作：已支付→退款/补单/明细；其余→明细
function actionsFor(o: Order): string[] {
  if (o.status === 1) return ['明细', '重新通知', '退款']
  if (o.status === 2) return ['明细', '重新通知']
  return ['明细']
}
const actionIcons: Record<string, any> = {
  明细: ListTree,
  重新通知: Bell,
  退款: Undo2,
}
async function onAction(a: string, o: Order) {
  openMenu.value = null
  if (a === '明细') router.push({ path: '/m/records', query: { kw: o.trade_no } })
  else if (a === '退款') openRefund(o)
  else if (a === '重新通知') await doRenotify(o)
}

// 重新通知（补单/重发回调）
const busy = ref(false)
async function doRenotify(o: Order) {
  if (busy.value) return
  busy.value = true
  try {
    await renotifyOrder(o.trade_no)
    toast.success('已重新发送通知')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '重新通知失败')
  } finally {
    busy.value = false
  }
}

// ===== 退款弹窗（后端为全额退款）=====
const refundOpen = ref(false)
const refundTarget = ref<Order | null>(null)
const refundMoney = ref('')     // 退款金额（空=全额退实付）
const refundPassword = ref('')  // 登录密码二次校验
// 剩余可退 = 实付 - 已退（refundmoney）
const refundMax = computed(() => {
  const o = refundTarget.value
  if (!o) return 0
  const real = +(o.realmoney ?? o.money ?? 0)
  const refunded = +(o.refundmoney ?? 0)
  return Math.max(0, +(real - refunded).toFixed(2))
})
function openRefund(o: Order) {
  refundTarget.value = o
  refundMoney.value = ''
  refundPassword.value = ''
  refundOpen.value = true
}
async function submitRefund() {
  const o = refundTarget.value
  if (!o || busy.value) return
  if (!refundPassword.value) { toast.error('请输入登录密码'); return }
  const money = refundMoney.value.trim()
  if (money) {
    const n = +money
    if (!(n > 0)) { toast.error('金额输入错误'); return }
    if (n > refundMax.value) { toast.error(`退款金额不能超过剩余可退 ¥${refundMax.value}`); return }
  }
  busy.value = true
  try {
    await refundOrder(o.trade_no, refundPassword.value, money || undefined)
    toast.success(money ? `订单 ${o.trade_no} 已退款 ¥${money}` : `订单 ${o.trade_no} 已全额退款`)
    refundOpen.value = false
    await loadOrders()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '退款失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="订单记录" :subtitle="`共 ${total} 笔订单`">
      <template #actions>
        <Button variant="outline" size="sm" @click="toggleStats"><BarChart3 />统计</Button>
      </template>
      <div class="space-y-3">
        <div class="filter-bar">
          <div class="filter-item">
            <label class="filter-label">订单信息</label>
            <Select v-model="filters.column" :options="columnOptions" class="w-32" />
            <input v-model="filters.value" placeholder="搜索内容" class="field-input w-48" />
          </div>
          <div class="filter-item">
            <label class="text-sm text-muted-foreground">支付方式</label>
            <Select v-model="filters.type" :options="typeOptions" class="w-28" />
          </div>
          <div class="filter-item">
            <label class="text-sm text-muted-foreground">订单状态</label>
            <Select v-model="filters.dstatus" :options="statusOptions" class="w-28" />
          </div>
        </div>
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

    <!-- 统计概况（服务端聚合，按当前筛选条件全量）-->
    <Panel v-if="showStats" title="订单统计概况" subtitle="按当前筛选条件（全量）">
      <div v-if="statsLoading" class="py-6 text-center dim">统计中…</div>
      <template v-else-if="stats">
        <div class="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3 lg:grid-cols-5">
          <div>
            <div class="text-[13px] text-muted-foreground">订单总金额</div>
            <div class="mt-1.5 text-xl font-normal tabular-nums"><span class="mr-0.5 text-xs text-muted-foreground">¥</span>{{ formatMoney(stats.totalMoney) }}</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">已支付金额</div>
            <div class="mt-1.5 text-xl font-normal tabular-nums text-success"><span class="mr-0.5 text-xs text-muted-foreground">¥</span>{{ formatMoney(stats.successMoney) }}</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">未支付金额</div>
            <div class="mt-1.5 text-xl font-normal tabular-nums"><span class="mr-0.5 text-xs text-muted-foreground">¥</span>{{ formatMoney(stats.unpaidMoney) }}</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">已退款金额</div>
            <div class="mt-1.5 text-xl font-normal tabular-nums text-destructive"><span class="mr-0.5 text-xs text-muted-foreground">¥</span>{{ formatMoney(stats.refundMoney) }}</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">总收入利润</div>
            <div class="mt-1.5 text-xl font-normal tabular-nums text-primary"><span class="mr-0.5 text-xs text-muted-foreground">¥</span>{{ formatMoney(stats.platformProfit) }}</div>
          </div>
        </div>
        <div class="mt-5 flex flex-wrap gap-x-8 gap-y-2 border-t border-border/70 pt-4 text-sm">
          <span class="text-muted-foreground">订单总数 <b class="text-foreground">{{ stats.totalCount }}</b></span>
          <span class="text-muted-foreground">已支付 <b class="text-foreground">{{ stats.successCount }}</b></span>
          <span class="text-muted-foreground">未支付 <b class="text-foreground">{{ stats.unpaidCount }}</b></span>
          <span class="text-muted-foreground">已退款 <b class="text-foreground">{{ stats.refundCount }}</b></span>
          <span class="text-muted-foreground">成功率 <b class="text-primary">{{ stats.successRate }}%</b></span>
        </div>
      </template>
    </Panel>

    <!-- 列表 -->
    <Panel title="订单列表" :subtitle="`${total} 条`">
      <template #actions>
        <Button variant="outline" size="sm" :disabled="exporting" @click="exportOrders"><Download class="size-4" />导出</Button>
      </template>
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[17%]">订单号 / 商户单号</th>
              <th class="w-[16%]">商品 / 金额</th>
              <th class="w-[12%]">实付</th>
              <th class="w-[15%]">支付方式</th>
              <th class="w-[16%]">创建 / 完成时间</th>
              <th class="col-center w-[10%]">状态</th>
              <th class="col-center w-[9%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in rows" :key="o.trade_no">
              <td>
                <div class="truncate font-medium text-primary">{{ o.trade_no }}</div>
                <div class="truncate text-xs dim">{{ o.out_trade_no }}</div>
              </td>
              <td>
                <div class="truncate">{{ o.name }}</div>
                <div class="text-xs"><span class="dim">¥</span><b>{{ o.money }}</b></div>
              </td>
              <td>
                <span v-if="o.realmoney"><span class="dim text-xs">¥</span>{{ o.realmoney }}</span>
                <span v-else class="dim">—</span>
              </td>
              <td>
                <div class="flex items-center gap-1.5">
                  <img :src="`/assets/icon/${o.typename}.ico`" class="size-4" onerror="this.style.display='none'" />
                  <span>{{ o.typeshowname }}</span>
                  <span class="dim">({{ o.channel }})</span>
                </div>
              </td>
              <td>
                <div class="text-xs">{{ o.addtime }}</div>
                <div class="text-xs dim">{{ o.endtime ?? '—' }}</div>
              </td>
              <td class="col-center">
                <Badge :variant="orderStatus[o.status]?.variant ?? 'muted'">{{ orderStatus[o.status]?.text ?? '未知' }}</Badge>
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
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button
                      v-for="a in actionsFor(o)"
                      :key="a"
                      class="menu-item"
                      :class="a === '退款' && 'menu-item-danger'"
                      @click="onAction(a, o)"
                    >
                      <component :is="actionIcons[a]" class="size-4 shrink-0 opacity-70" />
                      <span class="flex-1">{{ a }}</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="7" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="7" class="py-10 text-center dim">没有符合条件的订单</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 退款弹窗（全额退款）-->
    <Modal v-model="refundOpen" title="订单退款" width="max-w-md">
      <div v-if="refundTarget" class="space-y-3.5">
        <div class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          订单号 <span class="font-mono text-foreground">{{ refundTarget.trade_no }}</span> · 订单金额
          <b class="text-foreground">¥{{ refundTarget.money }}</b>
          · 剩余可退 <b class="text-foreground tabular-nums">¥{{ refundMax }}</b>
        </div>
        <div class="space-y-1.5">
          <label class="text-sm text-muted-foreground">退款金额</label>
          <input
            v-model="refundMoney"
            :placeholder="`留空按全额退 ¥${refundMax}`"
            class="field-input w-full"
            inputmode="decimal"
          />
          <p class="text-xs text-muted-foreground">支持部分退款，最多可退 ¥{{ refundMax }}；留空则全额退款。</p>
        </div>
        <div class="space-y-1.5">
          <label class="text-sm text-muted-foreground">登录密码</label>
          <input
            v-model="refundPassword"
            type="password"
            placeholder="请输入登录密码确认"
            class="field-input w-full"
            autocomplete="off"
          />
        </div>
        <p class="text-sm text-muted-foreground">
          退款后订单状态转为「已退款」，已入账的分成将按规则从可用余额扣回，此操作不可恢复。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="refundOpen = false">取消</Button>
        <Button variant="destructive" size="sm" :disabled="busy" @click="submitRefund"><Undo2 />确认退款</Button>
      </template>
    </Modal>
  </div>
</template>
