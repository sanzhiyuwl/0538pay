<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Search, RotateCcw } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, DateRange, Pagination } from '@/components/ui'
import { fetchBuyerStat, type BuyerStatRow } from '@/lib/api/stats'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { formatMoney } from '@/lib/utils'

const toast = useToast()

// 付款人维度（对齐 epay buyerStat method：0付款账号 1IP 2手机号）
const methodOptions = [
  { value: 0, label: '按付款账号' },
  { value: 1, label: '按付款IP' },
  { value: 2, label: '按手机号' },
]

function today(): string {
  return new Date().toISOString().slice(0, 10)
}
function daysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

// ===== 查询条件 =====
const query = ref({ startday: daysAgo(7), endday: today(), method: 0, kw: '' })
const rows = ref<BuyerStatRow[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const { list } = await fetchBuyerStat({
      method: query.value.method,
      startday: query.value.startday,
      endday: query.value.endday,
    })
    rows.value = list
    page.value = 1
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '查询失败')
    rows.value = []
  } finally {
    loading.value = false
  }
}

function resetQuery() {
  query.value = { startday: daysAgo(7), endday: today(), method: 0, kw: '' }
  load()
}

// 本地按关键词过滤（付款人标识）
const filtered = computed(() => {
  const kw = query.value.kw.trim()
  if (!kw) return rows.value
  return rows.value.filter((b) => b.user.includes(kw))
})

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

// 概况
const stats = computed(() => {
  const count = filtered.value.reduce((s, b) => s + b.count, 0)
  const amount = filtered.value.reduce((s, b) => s + Number(b.amount), 0)
  return { buyers: filtered.value.length, count, amount, avg: count ? amount / count : 0 }
})

const userColLabel = computed(() =>
  query.value.method === 1 ? '付款 IP' : query.value.method === 2 ? '手机号' : '付款账号',
)

onMounted(load)
</script>

<template>
  <div class="space-y-2.5">
    <!-- 查询条件 -->
    <Panel title="支付用户统计" subtitle="按付款人维度聚合：付款次数 / 累计金额 / 黑名单标记（对齐 epay buyerStat）">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">查询日期</label>
          <DateRange v-model:start="query.startday" v-model:end="query.endday" class="w-[328px]" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">统计维度</label>
          <Select v-model="query.method" :options="methodOptions" class="w-36" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">付款人</label>
          <input v-model="query.kw" placeholder="按标识过滤（本地）" class="field-input w-44" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" :disabled="loading" @click="load"><Search />立即查询</Button>
          <Button variant="outline" size="sm" @click="resetQuery"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 概况 -->
    <Panel title="付款人概况" subtitle="按当前筛选条件">
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">付款人数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.buyers }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">付款总次数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.count }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">累计金额</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-primary">
            <span class="text-sm opacity-70">¥</span>{{ formatMoney(stats.amount) }}
          </div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">笔均金额</div>
          <div class="mt-1 text-xl font-normal tabular-nums">
            <span class="text-sm opacity-70">¥</span>{{ formatMoney(stats.avg) }}
          </div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="付款人明细" :subtitle="`${total} 人（上限 500）`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[46%]">{{ userColLabel }}</th>
              <th class="num w-[16%]">付款次数</th>
              <th class="num w-[22%]">累计金额</th>
              <th class="w-[16%]">黑名单</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="b in pageRows" :key="b.user">
              <td class="truncate font-mono text-[13px]">{{ b.user }}</td>
              <td class="num tabular-nums">{{ b.count }}</td>
              <td class="num tabular-nums font-medium">
                <span class="dim text-xs">¥</span>{{ formatMoney(Number(b.amount)) }}
              </td>
              <td>
                <Badge v-if="b.is_black" variant="destructive">已拉黑</Badge>
                <span v-else class="dim">—</span>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="4" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!pageRows.length">
              <td colspan="4" class="py-10 text-center dim">暂无付款人统计数据</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        按付款人（付款账号 / IP / 手机号）聚合已支付订单，命中黑名单的付款人标红。同一付款人跨商户合并统计。
      </p>
    </Panel>
  </div>
</template>
