<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import {
  Search,
  RotateCcw,
  Download,
  BarChart3,
  MoreHorizontal,
  Send,
  Undo2,
  RefreshCw,
  XCircle,
  CheckCircle2,
  Ban,
  Trash2,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, DateRange, Pagination } from '@/components/ui'
import {
  psOrders,
  psStatus,
  searchColumns,
  psActions,
  calcPsStats,
} from '@/lib/mock/profitsharing'
import { formatMoney } from '@/lib/utils'

// ===== 下拉选项 =====
const columnOptions = searchColumns.map((c) => ({ value: c.value, label: c.label }))
const statusOptions = [
  { value: -1, label: '全部状态' },
  ...Object.entries(psStatus).map(([k, s]) => ({ value: Number(k), label: s.text })),
]

// ===== 筛选 =====
const filters = ref({
  column: 'trade_no',
  value: '',
  rid: '',
  starttime: '',
  endtime: '',
  dstatus: -1,
})

const filtered = computed(() => {
  return psOrders.filter((o) => {
    if (filters.value.rid && String(o.rid) !== filters.value.rid.trim()) return false
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
  filters.value = { column: 'trade_no', value: '', rid: '', starttime: '', endtime: '', dstatus: -1 }
  applySearch()
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

// ===== 多选 =====
const selected = ref<Set<number>>(new Set())
const pageAllChecked = computed(
  () => pageRows.value.length > 0 && pageRows.value.every((r) => selected.value.has(r.id)),
)
function toggleAll() {
  if (pageAllChecked.value) pageRows.value.forEach((r) => selected.value.delete(r.id))
  else pageRows.value.forEach((r) => selected.value.add(r.id))
}
function toggleOne(id: number) {
  if (selected.value.has(id)) selected.value.delete(id)
  else selected.value.add(id)
}

// 搜索 / 切换筛选：回第一页 + 清空跨筛选选中
function applySearch() {
  page.value = 1
  selected.value.clear()
}
// 筛选即时变化（改下拉/输入）也回第1页并清空选中，避免批量操作命中不可见行
watch(filters, applySearch, { deep: true })

// ===== 统计 =====
const showStats = ref(false)
const stats = computed(() => calcPsStats(filtered.value))

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

// 操作项 → 图标
const actionIcons: Record<string, any> = {
  提交分账: Send,
  查询结果: RefreshCw,
  分账回退: Undo2,
  重试: RefreshCw,
  取消: XCircle,
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="分账记录" :subtitle="`共 ${total} 笔分账`">
      <template #actions>
        <Button variant="outline" size="sm" @click="showStats = !showStats"><BarChart3 />统计</Button>
        <Button variant="outline" size="sm"><Download />导出列表</Button>
      </template>
      <div class="space-y-3">
        <div class="filter-bar">
          <div class="filter-item">
            <label class="filter-label">分账信息</label>
            <Select v-model="filters.column" :options="columnOptions" class="w-32" />
            <input v-model="filters.value" placeholder="搜索内容" class="field-input w-48" />
          </div>
          <div class="filter-item">
            <label class="text-sm text-muted-foreground">分账规则</label>
            <input v-model="filters.rid" placeholder="规则 ID" class="field-input w-28" />
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

    <!-- 统计概况（可展开） -->
    <Panel v-if="showStats" title="分账统计概况" subtitle="按当前筛选条件">
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
    <Panel title="分账列表" :subtitle="selected.size ? `已选 ${selected.size} 笔` : `${total} 条`">
      <template v-if="selected.size" #actions>
        <span class="text-sm text-muted-foreground">批量：</span>
        <Button variant="outline" size="sm" @click="selected.clear()"><Send />批量分账</Button>
        <Button variant="outline" size="sm" @click="selected.clear()"><Undo2 />批量回退</Button>
        <Button variant="outline" size="sm" @click="selected.clear()"><Ban />批量取消</Button>
        <Button variant="outline" size="sm" @click="selected.clear()"><CheckCircle2 />改为成功</Button>
        <Button variant="outline" size="sm" @click="selected.clear()"><Trash2 />删除</Button>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="col-center w-[4%]">
                <input type="checkbox" :checked="pageAllChecked" @change="toggleAll" />
              </th>
              <th class="w-[18%]">系统订单号</th>
              <th class="w-[16%]">分账规则 / 接收方</th>
              <th class="w-[14%]">支付通道</th>
              <th class="w-[11%]">分账金额</th>
              <th class="w-[15%]">时间</th>
              <th class="col-center w-[10%]">状态</th>
              <th class="col-center w-[12%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(o, si) in pageRows" :key="o.id">
              <td class="col-center">
                <input type="checkbox" :checked="selected.has(o.id)" @change="toggleOne(o.id)" />
              </td>
              <td>
                <div class="truncate font-medium text-primary">{{ o.trade_no }}</div>
                <div class="truncate text-xs dim">{{ o.api_trade_no }}</div>
              </td>
              <td>
                <div class="truncate">{{ o.rulename }}</div>
                <div class="truncate text-xs dim">{{ o.receiver }}</div>
              </td>
              <td>
                <div>{{ o.channelname }}</div>
                <div class="text-xs dim tabular-nums">#{{ o.channelid }}</div>
              </td>
              <td>
                <div class="tabular-nums"><span class="dim text-xs">¥</span><b>{{ o.money }}</b></div>
              </td>
              <td>
                <div class="text-xs">{{ o.addtime }}</div>
              </td>
              <td class="col-center">
                <Badge :variant="psStatus[o.status].variant">{{ psStatus[o.status].text }}</Badge>
                <div v-if="o.status === 3" class="mt-1 truncate text-xs text-destructive" :title="o.result">
                  {{ o.result }}
                </div>
              </td>
              <td class="col-center">
                <div v-if="psActions(o.status).length" class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(o.id)">
                    操作 <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === o.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="si >= pageRows.length - 3 && pageRows.length > 3
                      ? 'bottom-full mb-1.5'
                      : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <template v-for="(a, ai) in psActions(o.status)" :key="ai">
                      <button
                        class="menu-item"
                        :class="a === '取消' && 'menu-item-danger'"
                        @click="openMenu = null"
                      >
                        <component :is="actionIcons[a]" class="size-4 shrink-0 opacity-70" />
                        <span class="flex-1">{{ a }}</span>
                      </button>
                    </template>
                  </div>
                </div>
                <span v-else class="dim">—</span>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="8" class="py-10 text-center dim">没有符合条件的分账记录</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>
  </div>
</template>
