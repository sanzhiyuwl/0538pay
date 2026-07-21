<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Search } from 'lucide-vue-next'
import { Panel, Button, Select, DateRange } from '@/components/ui'
import { fetchPayStat, type StatResult } from '@/lib/api/stats'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { formatMoney } from '@/lib/utils'

const toast = useToast()

const methodOptions = [
  { value: 'type', label: '以支付方式查看' },
  { value: 'channel', label: '以支付通道查看' },
]
const statTypeOptions = [
  { value: 0, label: '订单金额统计' },
  { value: 1, label: '支付金额统计' },
  { value: 2, label: '分成金额统计' },
  { value: 3, label: '手续费利润统计' },
  { value: 4, label: '代付金额统计' },
]

// ===== 查询条件 =====
const query = reactive({ startday: '', endday: '', method: 'type', type: 0 })
const appliedMethod = ref('type')
const appliedType = ref(0)

const result = ref<StatResult>({ columns: [], rows: [], totals: {}, grand: 0 })
const loading = ref(false)

async function runQuery() {
  loading.value = true
  try {
    result.value = await fetchPayStat({
      method: query.method,
      type: query.type,
      startday: query.startday || undefined,
      endday: query.endday || undefined,
    })
    appliedMethod.value = query.method
    appliedType.value = query.type
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '统计查询失败')
    result.value = { columns: [], rows: [], totals: {}, grand: 0 }
  } finally {
    loading.value = false
  }
}
onMounted(runQuery)

const grandRows = computed(() => result.value.rows.length)
const statTypeLabel = computed(() => statTypeOptions.find((o) => o.value === appliedType.value)?.label ?? '')
</script>

<template>
  <div class="space-y-2.5">
    <!-- 查询条件 -->
    <Panel title="商户支付统计" subtitle="按支付方式 / 通道交叉透视各商户的金额统计（默认当天）">
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
    <Panel :title="statTypeLabel" :subtitle="`${grandRows} 个商户 · ${appliedMethod === 'channel' ? '按支付通道' : '按支付方式'}`">
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
            <tr v-if="loading">
              <td :colspan="result.columns.length + 3" class="py-10 text-center dim">统计中…</td>
            </tr>
            <tr v-else-if="!result.rows.length">
              <td :colspan="result.columns.length + 3" class="py-10 text-center dim">暂无统计数据</td>
            </tr>
          </tbody>
          <tfoot v-if="result.rows.length">
            <tr class="font-medium">
              <td>合计</td>
              <td class="dim">{{ grandRows }} 个商户</td>
              <td v-for="col in result.columns" :key="col.key" class="num font-semibold tabular-nums">
                <span class="dim text-xs">¥</span>{{ formatMoney(result.totals[col.key] || 0) }}
              </td>
              <td class="num font-semibold tabular-nums text-primary">
                <span class="text-xs opacity-70">¥</span>{{ formatMoney(result.grand) }}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </Panel>
  </div>
</template>
