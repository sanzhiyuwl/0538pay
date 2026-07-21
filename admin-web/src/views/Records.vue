<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Search, RotateCcw, Download, BarChart3, ArrowUpRight, ArrowDownRight } from 'lucide-vue-next'
import { Panel, Button, Select, DateRange, Pagination } from '@/components/ui'
import { fetchRecords, fetchRecordStats, type FundRecord, type RecordStats } from '@/lib/api/records'
import { ApiError } from '@/lib/api/client'
import { formatMoney } from '@/lib/utils'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 搜索字段下拉（对齐 epay record.php 的 column 选项）
const columnOptions = [
  { value: 'type', label: '操作类型' },
  { value: 'money', label: '变更金额' },
  { value: 'trade_no', label: '关联订单号' },
]

// 类型下拉（收入在前、支出在后），对齐易支付资金流水场景
const incTypes = ['订单收入', '余额充值', '邀请返现', '保证金解冻', '手动增加']
const decTypes = ['结算扣款', '提现转账', '订单退款', '保证金冻结', '手动扣除']
const typeOptions = [
  { value: '', label: '全部类型' },
  ...incTypes.map((t) => ({ value: t, label: t })),
  ...decTypes.map((t) => ({ value: t, label: t })),
]

// ===== 筛选 =====
const filters = reactive({
  column: 'type',
  value: '',
  uid: '',
  type: '',
  starttime: '',
  endtime: '',
})

// ===== 分页 + 数据 =====
const page = ref(1)
const pageSize = 15
const total = ref(0)
const rows = ref<FundRecord[]>([])
const loading = ref(false)

function buildParams() {
  const uidNum = Number(filters.uid.trim())
  return {
    page: page.value,
    pageSize,
    uid: filters.uid.trim() && !Number.isNaN(uidNum) ? uidNum : undefined,
    type: filters.type || undefined,
    column: filters.value.trim() ? filters.column : undefined,
    value: filters.value.trim() || undefined,
    starttime: filters.starttime || undefined,
    endtime: filters.endtime || undefined,
  }
}

async function load() {
  loading.value = true
  try {
    const res = await fetchRecords(buildParams())
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载资金明细失败')
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  load()
}

function resetFilters() {
  filters.column = 'type'
  filters.value = ''
  filters.uid = ''
  filters.type = ''
  filters.starttime = ''
  filters.endtime = ''
  page.value = 1
  load()
}

function go(p: number) {
  page.value = p
  load()
}

onMounted(load)

// ===== 统计（按当前筛选条件向后端拉取） =====
const showStats = ref(false)
const stats = ref<RecordStats | null>(null)
const statsLoading = ref(false)

async function toggleStats() {
  showStats.value = !showStats.value
  if (showStats.value) {
    statsLoading.value = true
    try {
      // 统计不受分页影响，只带筛选条件
      const { page: _p, pageSize: _ps, ...rest } = buildParams()
      void _p
      void _ps
      stats.value = await fetchRecordStats(rest)
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : '统计失败')
      stats.value = null
    } finally {
      statsLoading.value = false
    }
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="资金明细" :subtitle="`共 ${total} 条流水`">
      <template #actions>
        <Button variant="outline" size="sm" @click="toggleStats"><BarChart3 />统计</Button>
        <Button variant="outline" size="sm"><Download />导出列表</Button>
      </template>
      <div class="space-y-3">
        <div class="filter-bar">
          <div class="filter-item">
            <label class="filter-label">明细搜索</label>
            <Select v-model="filters.column" :options="columnOptions" class="w-32" />
            <input v-model="filters.value" placeholder="搜索内容" class="field-input w-44" @keyup.enter="search" />
          </div>
          <div class="filter-item">
            <label class="text-sm text-muted-foreground">商户号</label>
            <input v-model="filters.uid" placeholder="请输入商户号" class="field-input w-36" @keyup.enter="search" />
          </div>
          <div class="filter-item">
            <label class="text-sm text-muted-foreground">操作类型</label>
            <Select v-model="filters.type" :options="typeOptions" class="w-32" />
          </div>
        </div>
        <div class="filter-bar">
          <div class="filter-item">
            <label class="filter-label">变更时间</label>
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
    <Panel v-if="showStats" title="资金明细统计" subtitle="按当前筛选条件">
      <div v-if="statsLoading" class="py-6 text-center dim">统计中…</div>
      <template v-else-if="stats">
        <div class="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3">
          <div>
            <div class="text-[13px] text-muted-foreground">增加金额</div>
            <div class="mt-1.5 text-xl font-normal tabular-nums text-success">+ ¥{{ formatMoney(stats.incMoney) }}</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">减少金额</div>
            <div class="mt-1.5 text-xl font-normal tabular-nums text-destructive">- ¥{{ formatMoney(stats.decMoney) }}</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">净变更金额</div>
            <div class="mt-1.5 text-xl font-normal tabular-nums" :class="stats.totalMoney >= 0 ? 'text-foreground' : 'text-destructive'">
              {{ stats.totalMoney >= 0 ? '+' : '-' }} ¥{{ formatMoney(Math.abs(stats.totalMoney)) }}
            </div>
          </div>
        </div>
        <div class="mt-5 flex flex-wrap gap-x-8 gap-y-2 border-t border-border/70 pt-4 text-sm">
          <span class="text-muted-foreground">流水总数 <b class="text-foreground">{{ stats.totalCount }}</b></span>
          <span class="text-muted-foreground">入账 <b class="text-success">{{ stats.incCount }}</b></span>
          <span class="text-muted-foreground">出账 <b class="text-destructive">{{ stats.decCount }}</b></span>
        </div>
      </template>
    </Panel>

    <!-- 列表 -->
    <Panel title="资金流水" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[10%]">商户号</th>
              <th class="w-[16%]">操作类型</th>
              <th class="w-[15%]">变更金额</th>
              <th class="w-[14%]">变更前余额</th>
              <th class="w-[14%]">变更后余额</th>
              <th class="w-[16%]">时间</th>
              <th class="w-[15%]">关联订单号</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in rows" :key="r.id">
              <td class="font-medium tabular-nums">{{ r.uid }}</td>
              <td>
                <span
                  class="inline-flex items-center gap-1"
                  :class="r.action === 1 ? 'text-success' : 'text-destructive'"
                >
                  <ArrowUpRight v-if="r.action === 1" class="size-3.5" />
                  <ArrowDownRight v-else class="size-3.5" />
                  {{ r.type }}
                </span>
              </td>
              <td class="tabular-nums" :class="r.action === 1 ? 'text-success' : 'text-destructive'">
                <b>{{ r.action === 1 ? '+' : '-' }} <span class="text-xs font-normal opacity-70">¥</span>{{ formatMoney(r.money) }}</b>
              </td>
              <td class="tabular-nums dim">{{ formatMoney(r.oldmoney) }}</td>
              <td class="tabular-nums">{{ formatMoney(r.newmoney) }}</td>
              <td class="text-xs">{{ r.date }}</td>
              <td>
                <span v-if="r.trade_no" class="truncate text-primary">{{ r.trade_no }}</span>
                <span v-else class="dim">无</span>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="7" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="7" class="py-10 text-center dim">没有符合条件的资金明细</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="Math.max(1, Math.ceil(total / pageSize))" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>
  </div>
</template>
