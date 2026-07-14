<script setup lang="ts">
import { ref, computed } from 'vue'
import { Search, RotateCcw, Download } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination } from '@/components/ui'
import {
  billOrders,
  billTypeText,
  billTypeVariant,
  billStatusText,
  calcBillingStats,
  type BillType,
  type BillStatus,
} from '@/lib/mock/console'
import { sitePlanText, type SitePlan } from '@/lib/mock/sites'
import { formatMoney } from '@/lib/utils'

const stats = computed(() => calcBillingStats(billOrders))

// ===== 筛选 =====
const typeOptions = [
  { value: '', label: '全部类型' },
  ...(Object.keys(billTypeText) as BillType[]).map((k) => ({ value: k, label: billTypeText[k] })),
]
const statusOptions = [
  { value: -1, label: '全部状态' },
  ...Object.entries(billStatusText).map(([k, s]) => ({ value: Number(k), label: s.text })),
]

const filters = ref({ kw: '', type: '', status: -1 })
const filtered = computed(() =>
  billOrders.filter((b) => {
    if (filters.value.type && b.type !== filters.value.type) return false
    if (filters.value.status > -1 && b.status !== filters.value.status) return false
    if (filters.value.kw.trim()) {
      const v = filters.value.kw.trim()
      if (!`${b.id}${b.siteName}${b.domain}`.includes(v)) return false
    }
    return true
  }),
)
function resetFilters() {
  filters.value = { kw: '', type: '', status: -1 }
  page.value = 1
}

// ===== 分页 =====
const page = ref(1)
const pageSize = 10
const total = computed(() => filtered.value.length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))
const pageRows = computed(() =>
  filtered.value.slice((safePage.value - 1) * pageSize, safePage.value * pageSize),
)
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 概况 -->
    <Panel title="租户计费" subtitle="分站开通 / 续费 / 升级的账单记录，收入计入平台营收">
      <template #actions>
        <Button size="sm" variant="outline"><Download />导出账单</Button>
      </template>
      <div class="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3 lg:grid-cols-5">
        <div>
          <div class="text-[13px] text-muted-foreground">累计收入</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-primary"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.totalRevenue) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">本月收入</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-success"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.monthRevenue) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已支付账单</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums">{{ stats.paidCount }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">待支付</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-warning">{{ stats.pendingCount }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">退款金额</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-muted-foreground"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.refundAmount) }}</div>
        </div>
      </div>
    </Panel>

    <!-- 筛选 -->
    <Panel title="账单筛选" :subtitle="`共 ${total} 条`">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">账单搜索</label>
          <input v-model="filters.kw" placeholder="账单号 / 站点名 / 域名" class="field-input w-52" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">类型</label>
          <Select v-model="filters.type" :options="typeOptions" class="w-28" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.status" :options="statusOptions" class="w-28" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 账单列表 -->
    <Panel title="账单列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full">
          <thead>
            <tr>
              <th>账单号</th>
              <th>分站</th>
              <th class="col-center">套餐</th>
              <th class="col-center">类型</th>
              <th class="num">时长</th>
              <th class="num">金额</th>
              <th>支付方式</th>
              <th class="col-center">状态</th>
              <th>创建时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="b in pageRows" :key="b.id">
              <td class="font-mono text-[13px] tabular-nums">{{ b.id }}</td>
              <td>
                <div class="font-medium">{{ b.siteName }}</div>
                <div class="text-xs dim">{{ b.domain }}</div>
              </td>
              <td class="col-center"><Badge variant="outline">{{ sitePlanText[b.plan as SitePlan] }}</Badge></td>
              <td class="col-center"><Badge :variant="billTypeVariant[b.type]">{{ billTypeText[b.type] }}</Badge></td>
              <td class="num tabular-nums">{{ b.months }} 个月</td>
              <td class="num tabular-nums font-medium">
                <span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(b.amount) }}
              </td>
              <td class="text-sm text-muted-foreground">{{ b.payMethod }}</td>
              <td class="col-center"><Badge :variant="billStatusText[b.status as BillStatus].variant">{{ billStatusText[b.status as BillStatus].text }}</Badge></td>
              <td class="text-xs text-muted-foreground tabular-nums">{{ b.createTime }}</td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="9" class="py-10 text-center dim">没有符合条件的账单</td>
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
