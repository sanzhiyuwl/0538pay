<script setup lang="ts">
import { ref, computed } from 'vue'
import { Search, RotateCcw } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination } from '@/components/ui'
import { riskRecords, riskType, searchColumns, typeOptions, calcRiskStats } from '@/lib/mock/risk'

const columnOptions = searchColumns.map((c) => ({ value: c.value, label: c.label }))

// ===== 筛选 =====
const filters = ref({ column: 'uid', value: '', type: -1 })

const filtered = computed(() => {
  return riskRecords.filter((r) => {
    if (filters.value.type > -1 && r.type !== filters.value.type) return false
    if (filters.value.value.trim()) {
      const v = filters.value.value.trim()
      const field = (r as any)[filters.value.column]
      if (field == null || !String(field).includes(v)) return false
    }
    return true
  })
})

function resetFilters() {
  filters.value = { column: 'uid', value: '', type: -1 }
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

const stats = computed(() => calcRiskStats(filtered.value))
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="风控记录" :subtitle="`共 ${total} 条`">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">风控搜索</label>
          <Select v-model="filters.column" :options="columnOptions" class="w-28" />
          <input v-model="filters.value" placeholder="搜索内容" class="field-input w-48" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">风控类型</label>
          <Select v-model="filters.type" :options="typeOptions" class="w-36" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 概况 -->
    <Panel title="风控概况" subtitle="按当前筛选条件">
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">命中总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">关键词屏蔽</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-destructive">{{ stats.keyword }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">订单成功率</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-warning">{{ stats.successRate }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">连续通知失败</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.notifyFail }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">订单投诉率</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.complaint }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="风控记录列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[8%]">ID</th>
              <th class="w-[10%]">商户号</th>
              <th class="w-[14%]">风控类型</th>
              <th class="w-[34%]">风控内容</th>
              <th class="w-[18%]">风控网址</th>
              <th class="w-[16%]">时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in pageRows" :key="r.id">
              <td class="tabular-nums dim">{{ r.id }}</td>
              <td class="font-medium tabular-nums text-primary">{{ r.uid }}</td>
              <td>
                <Badge :variant="riskType[r.type].variant">{{ riskType[r.type].text }}</Badge>
              </td>
              <td class="truncate" :title="r.content">{{ r.content }}</td>
              <td class="truncate dim">{{ r.url }}</td>
              <td class="text-xs">{{ r.date }}</td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="6" class="py-10 text-center dim">没有符合条件的风控记录</td>
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
