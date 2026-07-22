<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Search, RotateCcw, Download } from 'lucide-vue-next'
import { Panel, Button, Select, Pagination } from '@/components/ui'
import { searchColumns, calcRecordStats, type FundRecord } from '@/lib/mock/merchant/records'
import { fetchMerchantRecords } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { formatMoney, exportCsv } from '@/lib/utils'

const route = useRoute()
const toast = useToast()
const columnOptions = searchColumns.map((c) => ({ value: c.value, label: c.label }))

// ===== 真接口数据（一次拉当前商户流水，客户端筛选/分页）=====
const fundRecords = ref<FundRecord[]>([])
async function loadRecords() {
  try {
    const res = await fetchMerchantRecords({ page: 1, pageSize: 100 })
    fundRecords.value = res.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '流水加载失败')
    fundRecords.value = []
  }
}

// ===== 筛选 =====
const filters = ref({ column: 'type', value: '' })
// 从订单页跳转带入 kw（按关联订单号搜）
onMounted(async () => {
  await loadRecords()
  const kw = route.query.kw
  if (typeof kw === 'string' && kw) {
    filters.value = { column: 'trade_no', value: kw }
  }
})

const filtered = computed(() =>
  fundRecords.value.filter((r) => {
    if (!filters.value.value.trim()) return true
    const v = filters.value.value.trim()
    const field = (r as any)[filters.value.column]
    return field != null && String(field).includes(v)
  }),
)
function resetFilters() {
  filters.value = { column: 'type', value: '' }
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

const stats = computed(() => calcRecordStats(filtered.value))

// ===== 导出（按当前筛选结果导出全部）=====
function exportRecords() {
  const rows = filtered.value
  if (!rows.length) { toast.error('没有可导出的流水'); return }
  const headers = ['时间', '操作类型', '收支', '变更金额', '变更前余额', '变更后余额', '关联订单号']
  const data = rows.map((r) => [
    r.date, r.type, r.action === 1 ? '收入' : '支出',
    (r.action === 1 ? '+' : '-') + r.money, r.oldmoney, r.newmoney, r.trade_no,
  ])
  exportCsv(`资金明细_${new Date().toISOString().slice(0, 10)}`, headers, data)
  toast.success(`已导出 ${rows.length} 条流水`)
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="资金明细" :subtitle="`共 ${total} 条流水`">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">筛选</label>
          <Select v-model="filters.column" :options="columnOptions" class="w-32" />
          <input v-model="filters.value" placeholder="搜索内容" class="field-input w-52" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 概况 -->
    <Panel title="资金概况" subtitle="按当前筛选条件">
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">流水笔数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.count }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">收入合计</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success"><span class="text-sm opacity-70">¥</span>{{ formatMoney(stats.income) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">支出合计</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-destructive"><span class="text-sm opacity-70">¥</span>{{ formatMoney(stats.expense) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">净变动</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-primary"><span class="text-sm opacity-70">¥</span>{{ formatMoney(stats.net) }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="流水明细" :subtitle="`${total} 条`">
      <template #actions>
        <Button variant="outline" size="sm" @click="exportRecords"><Download class="size-4" />导出</Button>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[16%]">操作类型</th>
              <th class="num w-[14%]">变更金额</th>
              <th class="num w-[14%]">变更前</th>
              <th class="num w-[14%]">变更后</th>
              <th class="w-[18%]">时间</th>
              <th class="w-[18%]">关联订单号</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in pageRows" :key="r.id">
              <td>{{ r.type }}</td>
              <td class="num tabular-nums font-medium" :class="r.action === 2 ? 'text-destructive' : 'text-success'">
                {{ r.action === 2 ? '-' : '+' }}{{ formatMoney(r.money) }}
              </td>
              <td class="num tabular-nums dim">{{ formatMoney(r.oldmoney) }}</td>
              <td class="num tabular-nums">{{ formatMoney(r.newmoney) }}</td>
              <td class="text-xs">{{ r.date }}</td>
              <td>
                <RouterLink v-if="r.trade_no" :to="{ path: '/m/orders' }" class="font-mono text-[13px] text-primary hover:underline">
                  {{ r.trade_no }}
                </RouterLink>
                <span v-else class="dim">—</span>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="6" class="py-10 text-center dim">没有符合条件的流水</td>
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
