<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Search, RotateCcw } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination } from '@/components/ui'
import { fetchRisks, type RiskRecord } from '@/lib/api/risk'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const riskType: Record<number, { text: string; variant: 'destructive' | 'warning' | 'default' | 'muted' }> = {
  0: { text: '关键词屏蔽', variant: 'destructive' },
  1: { text: '订单成功率', variant: 'warning' },
  2: { text: '连续通知失败', variant: 'default' },
  3: { text: '订单投诉率', variant: 'muted' },
}
const columnOptions = [
  { value: 'uid', label: '商户号' },
  { value: 'url', label: '风控网址' },
  { value: 'content', label: '风控内容' },
]
const typeOptions = [
  { value: -1, label: '风控类型' },
  ...Object.entries(riskType).map(([k, t]) => ({ value: Number(k), label: t.text })),
]

// ===== 筛选（对齐 epay：精确等值搜索）=====
const filters = reactive({ column: 'uid', value: '', type: -1 })

const page = ref(1)
const pageSize = 15
const total = ref(0)
const rows = ref<RiskRecord[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetchRisks({
      page: page.value,
      pageSize,
      column: filters.value.trim() ? filters.column : undefined,
      value: filters.value.trim() || undefined,
      type: filters.type > -1 ? filters.type : undefined,
    })
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载风控记录失败')
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}
function applySearch() {
  page.value = 1
  load()
}
function resetFilters() {
  filters.column = 'uid'
  filters.value = ''
  filters.type = -1
  applySearch()
}
function go(p: number) {
  page.value = p
  load()
}
onMounted(load)
const pageCount = () => Math.max(1, Math.ceil(total.value / pageSize))
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="风控记录" :subtitle="`共 ${total} 条 · 系统自动触发，只读`">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">风控搜索</label>
          <Select v-model="filters.column" :options="columnOptions" class="w-28" />
          <input v-model="filters.value" placeholder="精确匹配" class="field-input w-48" @keyup.enter="applySearch" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">风控类型</label>
          <Select v-model="filters.type" :options="typeOptions" class="w-36" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="applySearch"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
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
            <tr v-for="r in rows" :key="r.id">
              <td class="tabular-nums dim">{{ r.id }}</td>
              <td class="font-medium tabular-nums text-primary">{{ r.uid }}</td>
              <td>
                <Badge :variant="riskType[r.type].variant">{{ riskType[r.type].text }}</Badge>
              </td>
              <td class="truncate" :title="r.content">{{ r.content }}</td>
              <td class="truncate dim">{{ r.url || '—' }}</td>
              <td class="text-xs">{{ r.date }}</td>
            </tr>
            <tr v-if="loading">
              <td colspan="6" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="6" class="py-10 text-center dim">没有符合条件的风控记录</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount()" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>
  </div>
</template>
