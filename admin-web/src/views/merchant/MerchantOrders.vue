<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search, RotateCcw, BarChart3, MoreHorizontal, Undo2, Bell, ListTree } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, DateRange, Pagination, Modal } from '@/components/ui'
import {
  orders as allOrders,
  orderStatus,
  payTypes,
  searchColumns,
  calcStats,
  type Order,
} from '@/lib/mock/merchant/orders'
import { formatMoney } from '@/lib/utils'

const router = useRouter()

const columnOptions = searchColumns.map((c) => ({ value: c.value, label: c.label }))
const typeOptions = [{ value: 0, label: '全部方式' }, ...payTypes.map((t) => ({ value: t.id, label: t.showname }))]
const statusOptions = [
  { value: -1, label: '全部状态' },
  ...Object.entries(orderStatus).map(([k, s]) => ({ value: Number(k), label: s.text })),
]

// ===== 筛选 =====
const filters = ref({ column: 'trade_no', value: '', type: 0, starttime: '', endtime: '', dstatus: -1 })
const filtered = computed(() =>
  allOrders.filter((o) => {
    if (filters.value.type && o.type !== filters.value.type) return false
    if (filters.value.dstatus > -1 && o.status !== filters.value.dstatus) return false
    if (filters.value.value.trim()) {
      const v = filters.value.value.trim()
      const field = (o as any)[filters.value.column]
      if (field == null || !String(field).includes(v)) return false
    }
    return true
  }),
)
function resetFilters() {
  filters.value = { column: 'trade_no', value: '', type: 0, starttime: '', endtime: '', dstatus: -1 }
  page.value = 1
}

// ===== 分页 =====
const page = ref(1)
const pageSize = 15
const total = computed(() => filtered.value.length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))
const pageRows = computed(() => filtered.value.slice((safePage.value - 1) * pageSize, safePage.value * pageSize))
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
}
watch(filters, () => { page.value = 1 }, { deep: true })

// ===== 统计 =====
const showStats = ref(false)
const stats = computed(() => calcStats(filtered.value))

// ===== 行操作菜单 =====
const openMenu = ref<string | null>(null)
function toggleMenu(id: string) {
  openMenu.value = openMenu.value === id ? null : id
}
function closeMenu() {
  openMenu.value = null
}
onMounted(() => window.addEventListener('click', closeMenu))
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
function onAction(a: string, o: Order) {
  openMenu.value = null
  if (a === '明细') router.push({ path: '/m/records', query: { kw: o.trade_no } })
  else if (a === '退款') openRefund(o)
  // 重新通知：原型仅提示
}

// ===== 退款弹窗 =====
const refundOpen = ref(false)
const refundOrder = ref<Order | null>(null)
const refundForm = ref({ money: '', paypwd: '' })
function openRefund(o: Order) {
  refundOrder.value = o
  refundForm.value = { money: o.realmoney ?? '', paypwd: '' }
  refundOpen.value = true
}
function submitRefund() {
  refundOpen.value = false
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="订单记录" :subtitle="`共 ${total} 笔订单`">
      <template #actions>
        <Button variant="outline" size="sm" @click="showStats = !showStats"><BarChart3 />统计</Button>
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
            <Button size="sm" @click="page = 1"><Search />搜索</Button>
            <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
          </div>
        </div>
      </div>
    </Panel>

    <!-- 统计概况 -->
    <Panel v-if="showStats" title="订单统计概况" subtitle="按当前筛选条件">
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
          <div class="text-[13px] text-muted-foreground">净收入</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-primary"><span class="mr-0.5 text-xs text-muted-foreground">¥</span>{{ formatMoney(stats.income) }}</div>
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
    <Panel title="订单列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
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
            <tr v-for="(o, si) in pageRows" :key="o.trade_no">
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
                <Badge :variant="orderStatus[o.status].variant">{{ orderStatus[o.status].text }}</Badge>
                <div v-if="o.status === 2 && +o.refundmoney > 0 && +o.refundmoney < +(o.realmoney ?? 0)" class="mt-1 text-xs text-destructive">
                  部分退款 ¥{{ o.refundmoney }}
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(o.trade_no)">
                    操作 <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === o.trade_no"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="si >= pageRows.length - 3 && pageRows.length > 3 ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
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
            <tr v-if="!pageRows.length">
              <td colspan="7" class="py-10 text-center dim">没有符合条件的订单</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 退款弹窗 -->
    <Modal v-model="refundOpen" title="申请退款" width="max-w-md">
      <div v-if="refundOrder" class="space-y-3.5">
        <div class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          订单号 <span class="font-mono text-foreground">{{ refundOrder.trade_no }}</span> · 实付
          <b class="text-foreground">¥{{ refundOrder.realmoney }}</b>
        </div>
        <div class="row-field">
          <label class="lbl">退款金额</label>
          <input v-model="refundForm.money" class="field-input flex-1" placeholder="可小于实付金额（部分退款）" />
        </div>
        <div class="row-field">
          <label class="lbl">登录密码</label>
          <input v-model="refundForm.paypwd" type="password" class="field-input flex-1" placeholder="验证身份" />
        </div>
        <p class="text-xs text-muted-foreground">退款将原路退回买家，成功后从可用余额扣除相应金额。</p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="refundOpen = false">取消</Button>
        <Button size="sm" :disabled="!refundForm.money || !refundForm.paypwd" @click="submitRefund"><Undo2 />确认退款</Button>
      </template>
    </Modal>
  </div>
</template>
