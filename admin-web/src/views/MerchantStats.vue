<script setup lang="ts">
import { ref, computed } from 'vue'
import { Search, Download } from 'lucide-vue-next'
import { Panel, Button, Select, DateRange } from '@/components/ui'
import {
  methodOptions,
  statTypeOptions,
  buildStat,
  columnTotals,
} from '@/lib/mock/paystat'
import { formatMoney } from '@/lib/utils'

// ===== 查询条件 =====
const query = ref({
  startday: '2026-07-01',
  endday: '2026-07-12',
  method: 'type',
  type: 0,
})

// 已应用的查询（点"查询"才刷新，模拟 loadTable）
const applied = ref({ ...query.value })
function runQuery() {
  applied.value = { ...query.value }
}

const result = computed(() => buildStat(applied.value.method, applied.value.type))
const totals = computed(() => columnTotals(result.value.columns, result.value.rows))
const grandRows = computed(() => result.value.rows.length)

const statTypeLabel = computed(
  () => statTypeOptions.find((o) => o.value === applied.value.type)?.label ?? '',
)
</script>

<template>
  <div class="space-y-2.5">
    <!-- 查询条件 -->
    <Panel title="商户支付统计" subtitle="按支付方式 / 通道交叉透视各商户的金额统计">
      <template #actions>
        <Button variant="outline" size="sm"><Download />导出</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">查询日期</label>
          <DateRange v-model:start="query.startday" v-model:end="query.endday" class="w-[328px]" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">查看维度</label>
          <Select v-model="query.method" :options="methodOptions" class="w-40" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">统计口径</label>
          <Select v-model="query.type" :options="statTypeOptions" class="w-36" />
        </div>
        <div class="ml-auto flex items-center">
          <Button size="sm" @click="runQuery"><Search />立即查询</Button>
        </div>
      </div>
    </Panel>

    <!-- 透视统计表 -->
    <Panel :title="statTypeLabel" :subtitle="`${grandRows} 个商户 · ${applied.method === 'channel' ? '按支付通道' : '按支付方式'}`">
      <div class="overflow-x-auto">
        <table class="tbl w-full">
          <thead>
            <tr>
              <th class="w-[10%]">商户号</th>
              <th class="w-[16%]">商户</th>
              <th v-for="col in result.columns" :key="col.key" class="num">{{ col.label }}</th>
              <th class="num">合计</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in result.rows" :key="r.uid">
              <td class="font-medium tabular-nums">
                <span v-if="r.uid === 0">管理员</span>
                <span v-else class="text-primary">{{ r.uid }}</span>
              </td>
              <td class="truncate">{{ r.name }}</td>
              <td v-for="col in result.columns" :key="col.key" class="num">
                <span v-if="r.values[col.key] > 0" class="tabular-nums">
                  <span class="dim text-xs">¥</span>{{ formatMoney(r.values[col.key]) }}
                </span>
                <span v-else class="dim">—</span>
              </td>
              <td class="num font-semibold tabular-nums">
                <span class="dim text-xs">¥</span>{{ formatMoney(r.total) }}
              </td>
            </tr>
            <tr v-if="!result.rows.length">
              <td :colspan="result.columns.length + 3" class="py-10 text-center dim">暂无统计数据</td>
            </tr>
          </tbody>
          <tfoot v-if="result.rows.length">
            <tr class="font-medium">
              <td>合计</td>
              <td class="dim">{{ grandRows }} 个商户</td>
              <td v-for="col in result.columns" :key="col.key" class="num font-semibold tabular-nums">
                <span class="dim text-xs">¥</span>{{ formatMoney(totals.totals[col.key]) }}
              </td>
              <td class="num font-semibold tabular-nums text-primary">
                <span class="text-xs opacity-70">¥</span>{{ formatMoney(totals.grand) }}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </Panel>
  </div>
</template>
