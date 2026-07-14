<script setup lang="ts">
import { ref, computed } from 'vue'
import { Search, RotateCcw, Download, BarChart3, ArrowUpRight, ArrowDownRight } from 'lucide-vue-next'
import { Panel, Button, Select, DateRange, Pagination } from '@/components/ui'
import {
  fundRecords,
  searchColumns,
  typeOptions,
  calcRecordStats,
} from '@/lib/mock/records'
import { formatMoney } from '@/lib/utils'

const columnOptions = searchColumns.map((c) => ({ value: c.value, label: c.label }))

// ===== 筛选 =====
const filters = ref({
  column: 'type',
  value: '',
  uid: '',
  type: '',
  starttime: '',
  endtime: '',
})

const filtered = computed(() => {
  return fundRecords.filter((r) => {
    if (filters.value.uid && String(r.uid) !== filters.value.uid.trim()) return false
    if (filters.value.type && r.type !== filters.value.type) return false
    if (filters.value.value.trim()) {
      const v = filters.value.value.trim()
      const field = (r as any)[filters.value.column]
      if (field == null || !String(field).includes(v)) return false
    }
    return true
  })
})

function resetFilters() {
  filters.value = { column: 'type', value: '', uid: '', type: '', starttime: '', endtime: '' }
  page.value = 1
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

// ===== 统计 =====
const showStats = ref(false)
const stats = computed(() => calcRecordStats(filtered.value))
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="资金明细" :subtitle="`共 ${total} 条流水`">
      <template #actions>
        <Button variant="outline" size="sm" @click="showStats = !showStats"><BarChart3 />统计</Button>
        <Button variant="outline" size="sm"><Download />导出列表</Button>
      </template>
      <div class="space-y-3">
        <div class="filter-bar">
          <div class="filter-item">
            <label class="filter-label">明细搜索</label>
            <Select v-model="filters.column" :options="columnOptions" class="w-32" />
            <input v-model="filters.value" placeholder="搜索内容" class="field-input w-44" />
          </div>
          <div class="filter-item">
            <label class="text-sm text-muted-foreground">商户号</label>
            <input v-model="filters.uid" placeholder="请输入商户号" class="field-input w-36" />
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
            <Button size="sm" @click="page = 1"><Search />搜索</Button>
            <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
          </div>
        </div>
      </div>
    </Panel>

    <!-- 统计概况（可展开） -->
    <Panel v-if="showStats" title="资金明细统计" subtitle="按当前筛选条件">
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
            <tr v-for="r in pageRows" :key="r.id">
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
                <b>{{ r.action === 1 ? '+' : '-' }} <span class="text-xs font-normal opacity-70">¥</span>{{ r.money }}</b>
              </td>
              <td class="tabular-nums dim">{{ r.oldmoney }}</td>
              <td class="tabular-nums">{{ r.newmoney }}</td>
              <td class="text-xs">{{ r.date }}</td>
              <td>
                <span v-if="r.trade_no" class="truncate text-primary">{{ r.trade_no }}</span>
                <span v-else class="dim">无</span>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="7" class="py-10 text-center dim">没有符合条件的资金明细</td>
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
