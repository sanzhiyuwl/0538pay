<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Search, RotateCcw, Download } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, DateRange, Pagination } from '@/components/ui'
import {
  buyerStats,
  channelMeta,
  channelOptions,
  sortOptions,
  sortBuyers,
  calcBuyerStats,
  formatMoney,
} from '@/lib/mock/buyerstat'

// ===== 查询条件 =====
const query = ref({
  startday: '2026-07-01',
  endday: '2026-07-12',
  channel: 'all',
  sort: 'amount',
  kw: '',
})

const filtered = computed(() => {
  let list = buyerStats.filter((b) => {
    if (query.value.channel !== 'all' && b.channel !== query.value.channel) return false
    if (query.value.kw.trim() && !b.buyerId.includes(query.value.kw.trim())) return false
    return true
  })
  return sortBuyers(list, query.value.sort)
})

function resetQuery() {
  query.value = { startday: '2026-07-01', endday: '2026-07-12', channel: 'all', sort: 'amount', kw: '' }
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

const stats = computed(() => calcBuyerStats(filtered.value))

watch(query, () => { page.value = 1 }, { deep: true })
</script>

<template>
  <div class="space-y-2.5">
    <!-- 查询条件 -->
    <Panel title="支付用户统计" subtitle="按付款人维度聚合：付款次数 / 累计金额 / 关联商户">
      <template #actions>
        <Button variant="outline" size="sm"><Download />导出</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">查询日期</label>
          <DateRange v-model:start="query.startday" v-model:end="query.endday" class="w-[328px]" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">付款渠道</label>
          <Select v-model="query.channel" :options="channelOptions" class="w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">排序</label>
          <Select v-model="query.sort" :options="sortOptions" class="w-36" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">付款人</label>
          <input v-model="query.kw" placeholder="付款用户标识" class="field-input w-48" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm"><Search />立即查询</Button>
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
    <Panel title="付款人明细" :subtitle="`${total} 人`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[24%]">付款用户标识</th>
              <th class="w-[10%]">渠道</th>
              <th class="num w-[10%]">付款次数</th>
              <th class="num w-[14%]">累计金额</th>
              <th class="num w-[10%]">关联商户</th>
              <th class="w-[16%]">首次付款</th>
              <th class="w-[16%]">最近付款</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="b in pageRows" :key="b.buyerId">
              <td class="truncate font-mono text-[13px]">{{ b.buyerId }}</td>
              <td>
                <Badge :variant="channelMeta[b.channel].variant">{{ channelMeta[b.channel].text }}</Badge>
              </td>
              <td class="num tabular-nums">{{ b.count }}</td>
              <td class="num tabular-nums font-medium">
                <span class="dim text-xs">¥</span>{{ formatMoney(b.amount) }}
              </td>
              <td class="num tabular-nums">{{ b.merchants }}</td>
              <td class="text-xs">{{ b.firstTime }}</td>
              <td class="text-xs">{{ b.lastTime }}</td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="7" class="py-10 text-center dim">暂无付款人统计数据</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        付款用户标识来源于微信 openid / 支付宝用户 ID，已脱敏展示。同一用户在不同商户的付款会合并统计。
      </p>
    </Panel>
  </div>
</template>
