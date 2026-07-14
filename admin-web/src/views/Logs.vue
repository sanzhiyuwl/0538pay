<script setup lang="ts">
import { ref, computed } from 'vue'
import { Search, RotateCcw, ExternalLink } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination } from '@/components/ui'
import { loginLogs, searchColumns } from '@/lib/mock/settings'

const columnOptions = searchColumns.map((c) => ({ value: c.value, label: c.label }))

// ===== 筛选 =====
const filters = ref({ column: 'uid', value: '' })

const filtered = computed(() => {
  return loginLogs.filter((l) => {
    if (filters.value.value.trim()) {
      const v = filters.value.value.trim()
      const field = (l as any)[filters.value.column]
      if (field == null || !String(field).includes(v)) return false
    }
    return true
  })
})

function resetFilters() {
  filters.value = { column: 'uid', value: '' }
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

// 操作类型 → Badge 变体
function typeVariant(type: string): 'default' | 'success' | 'destructive' | 'muted' {
  if (type === '登录失败') return 'destructive'
  if (type === '管理员登录') return 'default'
  if (type === '退出登录') return 'muted'
  return 'success'
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="登录日志" :subtitle="`共 ${total} 条`">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">日志搜索</label>
          <Select v-model="filters.column" :options="columnOptions" class="w-28" />
          <input v-model="filters.value" placeholder="搜索内容（0 为管理员）" class="field-input w-52" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="登录日志列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[10%]">ID</th>
              <th class="w-[14%]">商户号</th>
              <th class="w-[20%]">操作类型</th>
              <th class="w-[28%]">操作 IP</th>
              <th class="w-[28%]">时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in pageRows" :key="l.id">
              <td class="tabular-nums dim">{{ l.id }}</td>
              <td>
                <span v-if="l.uid > 0" class="font-medium tabular-nums text-primary">{{ l.uid }}</span>
                <span v-else class="dim">管理员</span>
              </td>
              <td><Badge :variant="typeVariant(l.type)">{{ l.type }}</Badge></td>
              <td>
                <a
                  :href="`https://www.ip138.com/iplookup.php?ip=${l.ip}`"
                  target="_blank"
                  rel="noreferrer"
                  class="inline-flex items-center gap-1 tabular-nums hover:text-primary"
                >
                  {{ l.ip }}<ExternalLink class="size-3 opacity-50" />
                </a>
              </td>
              <td class="text-xs">{{ l.date }}</td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="5" class="py-10 text-center dim">没有符合条件的登录日志</td>
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
